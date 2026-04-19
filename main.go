package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"invite/api"
	"invite/config"
	"invite/db"

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/swaggest/swgui/v5emb"
)

type Invite struct {
	ID          uuid.UUID
	Title       string
	Description string
	From        time.Time
	To          time.Time
	Duration    time.Duration
	CreatedAt   time.Time
	Status      string // pending, active, completed, cancelled
}

type PhaseState struct {
	PhaseID     uuid.UUID
	Status      string // active, completed, failed
	NextCheckAt *time.Time
	Data        json.RawMessage
}

type App struct {
	Queries *db.Queries
	DB      *sql.DB
}

func (app *App) InvitePerson(ctx context.Context, i Invite, p Person) (*Invitee, error) {
	inviteeID := uuid.New()
	err := app.Queries.CreateInvitee(ctx, db.CreateInviteeParams{
		ID:        inviteeID,
		InviteID:  i.ID,
		ContactID: p.ID,
		State:     string(InviteeStatePending),
		CreatedAt: time.Now(),
	})
	if err != nil {
		return nil, err
	}

	return &Invitee{
		ID:        inviteeID,
		InviteID:  i.ID,
		ContactID: p.ID,
		State:     InviteeStatePending,
		CreatedAt: time.Now(),
	}, nil
}

type Person = db.Person

type Group struct {
	ID          uuid.UUID
	Name        string
	Description string
}

func (app *App) AddMember(g Group, p Person) *GroupMember {

	// TODO db

	return &GroupMember{
		ID:        uuid.New(),
		ContactID: p.ID,
		GroupID:   g.ID,
	}
}

type GroupMember struct {
	ID        uuid.UUID
	ContactID uuid.UUID
	GroupID   uuid.UUID
}

type Phase struct {
	ID             uuid.UUID
	InviteID       uuid.UUID
	Order          int //UNIQUE
	StrategyKind   string
	StrategyConfig json.RawMessage
}

type InviteeState string

const (
	InviteeStateAccepted InviteeState = "accepted"
	InviteeStatePending  InviteeState = "pending"
	InviteeStateDeclined InviteeState = "declined"
	InviteeStateExpired  InviteeState = "expired"
)

type Invitee struct {
	ID        uuid.UUID
	InviteID  uuid.UUID //NOT NULL, should have index
	ContactID uuid.UUID //NOT NULL, should have index
	State     InviteeState
	CreatedAt time.Time
	// Unique (InviteID, ContactID)
}

type StrategyKind string

const (
	StrategyKindLadder StrategyKind = "ladder"
	StrategyKindSprint StrategyKind = "sprint"
)

type Event struct {
	Kind      string // e.g., "invitee_declined"
	InviteeID uuid.UUID
}

type Strategy interface {
	Kind() StrategyKind
	Execute(ctx context.Context, invite Invite, phase Phase) error
	Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error
	HandleEvent(ctx context.Context, invite Invite, phase Phase, state *PhaseState, event Event) error
}

// Factory function to turn DB data into usable logic
func LoadStrategy(app *App, p Phase) (Strategy, error) {
	kind := StrategyKind(p.StrategyKind)
	switch kind {
	case StrategyKindLadder:
		var cfg LadderStrategy
		if err := json.Unmarshal(p.StrategyConfig, &cfg); err != nil {
			return nil, err
		}
		cfg.app = app // Inject DB access
		return &cfg, nil

	case StrategyKindSprint:
		var cfg SprintStrategy
		if err := json.Unmarshal(p.StrategyConfig, &cfg); err != nil {
			return nil, err
		}
		cfg.app = app // Inject DB access
		return &cfg, nil

	default:
		return nil, fmt.Errorf("unknown strategy: %s", p.StrategyKind)
	}
}

type LadderStrategy struct {
	app     *App
	Active  bool
	List    []Person
	Current *Person
	Timeout time.Duration
}

func (ls *LadderStrategy) Kind() StrategyKind {
	return StrategyKindLadder
}

func (ls *LadderStrategy) Execute(ctx context.Context, invite Invite, phase Phase) error {
	if len(ls.List) == 0 {
		return nil
	}

	_, err := ls.app.InvitePerson(ctx, invite, ls.List[0])
	if err != nil {
		return err
	}

	nextCheckAt := time.Now().Add(ls.Timeout)
	data, _ := json.Marshal(map[string]int{"index": 0})

	return ls.app.Queries.CreatePhaseState(ctx, db.CreatePhaseStateParams{
		PhaseID:     phase.ID,
		Status:      "active",
		NextCheckAt: sql.NullTime{Time: nextCheckAt, Valid: true},
		Data:        data,
	})
}

func (ls *LadderStrategy) Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error {
	var data struct{ Index int }
	if err := json.Unmarshal(state.Data, &data); err != nil {
		return err
	}

	nextIndex := data.Index + 1
	if nextIndex >= len(ls.List) {
		return ls.app.Queries.UpdatePhaseState(ctx, db.UpdatePhaseStateParams{
			PhaseID:     phase.ID,
			Status:      "completed",
			NextCheckAt: sql.NullTime{Valid: false},
			Data:        state.Data,
		})
	}

	_, err := ls.app.InvitePerson(ctx, invite, ls.List[nextIndex])
	if err != nil {
		return err
	}

	nextCheckAt := time.Now().Add(ls.Timeout)
	newData, _ := json.Marshal(map[string]int{"index": nextIndex})

	return ls.app.Queries.UpdatePhaseState(ctx, db.UpdatePhaseStateParams{
		PhaseID:     phase.ID,
		Status:      "active",
		NextCheckAt: sql.NullTime{Time: nextCheckAt, Valid: true},
		Data:        newData,
	})
}

func (ls *LadderStrategy) HandleEvent(ctx context.Context, invite Invite, phase Phase, state *PhaseState, event Event) error {
	if event.Kind != "invitee_declined" {
		return nil
	}

	var data struct{ Index int }
	if err := json.Unmarshal(state.Data, &data); err != nil {
		return err
	}

	if data.Index >= len(ls.List) {
		return nil
	}

	invitee, err := ls.app.Queries.GetInvitee(ctx, event.InviteeID)
	if err != nil {
		return err
	}

	currentPerson := ls.List[data.Index]
	if invitee.ContactID != currentPerson.ID {
		return nil // Not the current person in the ladder
	}

	// Trigger immediate Resume
	return ls.Resume(ctx, invite, phase, state)
}

type SprintStrategy struct {
	app        *App
	Active     bool
	Recipients []uuid.UUID // Can be both groups and individuals
	Deadline   time.Time
}

func (ls *SprintStrategy) Kind() StrategyKind {
	return StrategyKindSprint
}

func (ls *SprintStrategy) Execute(ctx context.Context, invite Invite, phase Phase) error {
	persons, err := ls.app.Queries.ResolveRecipients(ctx, ls.Recipients)
	if err != nil {
		return err
	}

	g, gCtx := errgroup.WithContext(ctx)
	jobs := make(chan Person, len(persons))

	numWorkers := 10
	if len(persons) < numWorkers {
		numWorkers = len(persons)
	}

	for range numWorkers {
		g.Go(func() error {
			for {
				select {
				case <-gCtx.Done():
					return gCtx.Err()
				case p, ok := <-jobs:
					if !ok {
						return nil
					}
					_, err := ls.app.InvitePerson(gCtx, invite, p)
					if err != nil {
						return err
					}
				}
			}
		})
	}

	for _, p := range persons {
		jobs <- p
	}
	close(jobs)

	if err := g.Wait(); err != nil {
		return err
	}

	return ls.app.Queries.CreatePhaseState(ctx, db.CreatePhaseStateParams{
		PhaseID:     phase.ID,
		Status:      "active",
		NextCheckAt: sql.NullTime{Time: ls.Deadline, Valid: true},
		Data:        json.RawMessage("{}"),
	})
}

func (ls *SprintStrategy) Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error {
	if time.Now().After(ls.Deadline) || time.Now().Equal(ls.Deadline) {
		return ls.app.Queries.UpdatePhaseState(ctx, db.UpdatePhaseStateParams{
			PhaseID:     phase.ID,
			Status:      "completed",
			NextCheckAt: sql.NullTime{Valid: false},
			Data:        state.Data,
		})
	}
	return nil
}

func (ls *SprintStrategy) HandleEvent(ctx context.Context, invite Invite, phase Phase, state *PhaseState, event Event) error {
	return nil
}

func setupLogger(cfg *config.Config) {
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if strings.ToLower(cfg.LogFormat) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	setupLogger(cfg)
	slog.Info("Invite application starting...", slog.Int("port", cfg.Port))

	// 2. Setup Graceful Shutdown Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Initialize Database
	dbConn, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to db", slog.Any("error", err))
		os.Exit(1)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		slog.Error("Failed to ping db", slog.Any("error", err))
		os.Exit(1)
	}

	// Run migrations
	if err := db.Migrate(ctx, dbConn); err != nil {
		slog.Error("Failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	app := &App{
		Queries: db.New(dbConn),
		DB:      dbConn,
	}

	// 4. Initialize API server
	server := &api.Server{Queries: app.Queries}
	strictHandler := api.NewStrictHandler(server, nil)
	mux := http.NewServeMux()
	api.HandlerFromMux(strictHandler, mux)

	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		swagger, err := api.GetSwagger()
		if err != nil {
			http.Error(w, "Failed to load swagger spec", http.StatusInternalServerError)
			return
		}
		data, _ := swagger.MarshalJSON()
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})
	mux.Handle("GET /swagger/", v5emb.New("Invite API", "/openapi.json", "/swagger/"))

	// Add request validation middleware
	swagger, err := api.GetSwagger()
	if err != nil {
		slog.Error("Error loading swagger spec", slog.Any("error", err))
		os.Exit(1)
	}

	validator := middleware.OapiRequestValidator(swagger)(mux)
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip validation for Swagger UI and OpenAPI spec
		if r.URL.Path == "/openapi.json" || strings.HasPrefix(r.URL.Path, "/swagger/") {
			mux.ServeHTTP(w, r)
			return
		}
		validator.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: handler,
	}

	// 5. Start Background Tasks (Orchestrator)
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("Starting Orchestrator...")
		return app.RunOrchestrator(gCtx)
	})

	// 6. Start HTTP Server
	g.Go(func() error {
		slog.Info("API server listening", slog.Int("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server error: %w", err)
		}
		return nil
	})

	// 7. Wait for Shutdown Signal
	<-ctx.Done()
	slog.Info("Shutdown signal received, shutting down gracefully...")

	// Create a timeout context for the HTTP server shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", slog.Any("error", err))
	}

	// Wait for background tasks (like orchestrator) to finish
	if err := g.Wait(); err != nil {
		slog.Error("Error during shutdown", slog.Any("error", err))
	}

	slog.Info("Application stopped gracefully.")
}
