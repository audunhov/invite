package sprint

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"invite/db"
	"invite/models"
)

type SprintStrategy struct {
	Queries *db.Queries
	Inviter models.Inviter

	Active     bool
	Recipients []uuid.UUID
	Deadline   time.Time
}

func (ls *SprintStrategy) Kind() models.StrategyKind {
	return models.StrategyKindSprint
}

func (ls *SprintStrategy) Progress(state *models.PhaseState) string {
	if state == nil {
		return "Not started"
	}
	return "Invites sent, waiting for deadline"
}

func (ls *SprintStrategy) Execute(ctx context.Context, invite models.Invite, phase models.Phase) error {
	persons, err := ls.Queries.ResolveRecipients(ctx, ls.Recipients)
	if err != nil {
		return err
	}

	if len(persons) == 0 {
		return ls.Queries.CreatePhaseState(ctx, db.CreatePhaseStateParams{
			PhaseID:     phase.ID,
			Status:      "completed",
			NextCheckAt: sql.NullTime{Valid: false},
			Data:        json.RawMessage("{}"),
		})
	}

	g, gCtx := errgroup.WithContext(ctx)
	jobs := make(chan models.Person, len(persons))

	numWorkers := 10
	if len(persons) < numWorkers {
		numWorkers = len(persons)
	}

	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for {
				select {
				case <-gCtx.Done():
					return gCtx.Err()
				case p, ok := <-jobs:
					if !ok {
						return nil
					}
					_, err := ls.Inviter.InvitePerson(gCtx, invite, p)
					if err != nil {
						return err
					}
				}
			}
		})
	}

	for _, p := range persons {
		jobs <- models.Person{
			ID:    p.ID,
			Email: p.Email,
			Name:  p.Name,
		}
	}
	close(jobs)

	if err := g.Wait(); err != nil {
		return err
	}

	return ls.Queries.CreatePhaseState(ctx, db.CreatePhaseStateParams{
		PhaseID:     phase.ID,
		Status:      "active",
		NextCheckAt: sql.NullTime{Time: ls.Deadline, Valid: true},
		Data:        json.RawMessage("{}"),
	})
}

func (ls *SprintStrategy) Resume(ctx context.Context, invite models.Invite, phase models.Phase, state *models.PhaseState) error {
	if time.Now().After(ls.Deadline) || time.Now().Equal(ls.Deadline) {
		state.Status = "completed"
		state.NextCheckAt = sql.NullTime{Valid: false}
	}
	return nil
}

func (ls *SprintStrategy) HandleEvent(ctx context.Context, invite models.Invite, phase models.Phase, state *models.PhaseState, event models.Event) error {
	if event.Kind == "invitee_accepted" {
		state.Status = "completed"
		state.NextCheckAt = sql.NullTime{Valid: false}
	}
	return nil
}
