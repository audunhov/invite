package main

import (
	"context"
	"database/sql"
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

	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/swaggest/swgui/v5emb"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"invite/api"
	"invite/config"
	"invite/db"
	"invite/email"
	"invite/internal/app"
	"invite/internal/limiter"
	"invite/internal/logging"
	"invite/internal/seed"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	logging.Setup(cfg)
	slog.Info("Invite application starting...", slog.Int("port", cfg.Port))

	// 2. Setup Graceful Shutdown Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. Initialize Database
	dbConn := initDB(cfg)
	defer dbConn.Close()

	// Run migrations
	if err := db.Migrate(ctx, dbConn); err != nil {
		slog.Error("Failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	queries := db.New(dbConn)

	// Check for subcommand
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := seed.Run(ctx, dbConn, queries); err != nil {
			slog.Error("Failed to seed database", slog.Any("error", err))
			os.Exit(1)
		}
		return
	}

	emailService := email.NewService(cfg)
	application := app.New(dbConn, queries, emailService)

	// 4. Initialize API server
	ipLimiter := limiter.NewIPRateLimiter(rate.Every(time.Second), 5)
	server := &api.Server{
		Queries:                   queries,
		StartInviteFunc:           application.StartInviteProcess,
		GetProgressFunc:           application.GetPhaseProgress,
		HandleInviteeResponseFunc:  application.HandleInviteeResponse,
		InvalidateInviteFunc:      application.InvalidateInvite,
		InvalidatePhaseFunc:       application.InvalidatePhase,
		GetDashboardStatsFunc:    application.GetDashboardStats,
		CreateInviteDeepFunc:     application.CreateInviteDeep,
		Limiter:                   ipLimiter,
		EmailService:              emailService,
	}
	strictHandler := api.NewStrictHandler(server, nil)

	// Create main router
	mux := http.NewServeMux()

	// 1. Metadata & Documentation (no validation)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
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

	// 2. API Routes (with validation)
	swagger, err := api.GetSwagger()
	if err != nil {
		slog.Error("Error loading swagger spec", slog.Any("error", err))
		os.Exit(1)
	}

	apiMux := http.NewServeMux()
	api.HandlerFromMux(strictHandler, apiMux)

	validator := middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
		SilenceServersWarning: true,
	})

	handler := http.Handler(apiMux)
	protectedHandler := server.AuthMiddleware(handler)
	
	finalApiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		if strings.HasPrefix(path, "/auth/login") || 
		   strings.HasPrefix(path, "/auth/forgot-password") || 
		   strings.HasPrefix(path, "/respond/") {
			limiter := ipLimiter.GetLimiter(r.RemoteAddr)
			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
		}

		if strings.HasPrefix(path, "/auth/login") || 
		   strings.HasPrefix(path, "/auth/forgot-password") || 
		   strings.HasPrefix(path, "/auth/reset-password") || 
		   strings.HasPrefix(path, "/respond/") {
			handler.ServeHTTP(w, r)
			return
		}
		
		protectedHandler.ServeHTTP(w, r)
	})

	apiHandler := validator(http.StripPrefix("/api", finalApiHandler))
	mux.Handle("/api/", apiHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	// 5. Start Background Tasks (Orchestrator)
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("Starting Orchestrator...")
		return application.RunOrchestrator(gCtx)
	})

	// 6. Start HTTP Server
	g.Go(func() error {
		slog.Info("API server listening", slog.Int("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server error: %w", err)
		}
		return nil
	})

	<-ctx.Done()
	slog.Info("Shutdown signal received, shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", slog.Any("error", err))
	}

	if err := g.Wait(); err != nil {
		slog.Error("Error during shutdown", slog.Any("error", err))
	}

	slog.Info("Application stopped gracefully.")
}

func initDB(cfg *config.Config) *sql.DB {
	var dbConn *sql.DB
	var err error
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		dbConn, err = sql.Open("postgres", cfg.DatabaseURL)
		if err == nil {
			err = dbConn.Ping()
		}

		if err == nil {
			return dbConn
		}

		slog.Warn("Failed to connect to db, retrying...", slog.Int("attempt", i+1), slog.Any("error", err))
		time.Sleep(2 * time.Second)
	}

	slog.Error("Failed to connect to db after retries", slog.Any("error", err))
	os.Exit(1)
	return nil
}
