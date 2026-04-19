package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type Invite struct {
	ID          uuid.UUID // Primary key
	Title       string    //Allowed to be blank (will be displayed as "New invite", can be edited later)
	Description string    //Allowed to be blank
	From        time.Time // Not null
	To          time.Time
	Duration    time.Duration
	CreatedAt   time.Time // Not null
	// Can be created with only from and to, or from and duration. to or duration must be present.
}

type App struct {
	DB *sql.Conn
}

func (app *App) InvitePerson(i Invite, p Person) *Invitee {

	// TODO db

	return &Invitee{
		ID:        uuid.New(),
		InviteID:  i.ID,
		ContactID: p.ID,
		State:     "pending",
		CreatedAt: time.Now(),
	}
}

type Person struct {
	ID    uuid.UUID // PRIMARY KEY
	Email string    // UNIQUE NOT NULL, NOT BLANK. Not primary key as you might change email
	Name  string    // Email displayed if not set
}

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

type Strategy interface {
	Kind() StrategyKind
	Status() string                                   // Status to be displayed
	Execute(ctx context.Context, invite Invite) error // Starts the sending process for a given strategy
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

func (ls *LadderStrategy) Status() string {
	return "pending"
}

func (ls *LadderStrategy) Execute(ctx context.Context, invite Invite) error {
	return errors.New("Not implemented")
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

func (ls *SprintStrategy) Status() string {
	return "pending"
}

func (ls *SprintStrategy) Execute(ctx context.Context, invite Invite) error {
	g, gCtx := errgroup.WithContext(ctx)

	jobs := make(chan uuid.UUID, len(ls.Recipients))

	numWorkers := 10

	for range numWorkers {
		g.Go(func() error {
			// Re-use the SMTP connection here if possible
			// smtpClient := connectToSMTP()
			// defer smtpClient.Quit()

			for {
				select {
				case <-gCtx.Done():
					return gCtx.Err()
				case id, ok := <-jobs:
					if !ok {
						return nil
					} // Channel closed

					fmt.Printf("Worker sending to %s\n", id)
				}
				time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
			}
		})
	}

	for _, id := range ls.Recipients {
		jobs <- id
	}
	close(jobs)

	return g.Wait()
}

func main() {

}
