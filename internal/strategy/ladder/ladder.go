package ladder

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"invite/db"
	"invite/models"
)

type LadderStrategy struct {
	Queries *db.Queries
	Inviter models.Inviter

	Active  bool
	List    []models.Person
	Current *models.Person
	Timeout time.Duration
}

func (ls *LadderStrategy) Kind() models.StrategyKind {
	return models.StrategyKindLadder
}

func (ls *LadderStrategy) Progress(state *models.PhaseState) string {
	if state == nil || state.Data == nil {
		return "Not started"
	}
	var data struct{ Index int }
	if err := json.Unmarshal(state.Data, &data); err != nil {
		return "Error parsing state"
	}
	return fmt.Sprintf("Waiting for person %d of %d", data.Index+1, len(ls.List))
}

func (ls *LadderStrategy) Execute(ctx context.Context, invite models.Invite, phase models.Phase) error {
	if len(ls.List) == 0 {
		return nil
	}

	_, err := ls.Inviter.InvitePerson(ctx, invite, ls.List[0])
	if err != nil {
		return err
	}

	nextCheckAt := time.Now().Add(ls.Timeout)
	data, _ := json.Marshal(map[string]int{"index": 0})

	return ls.Queries.CreatePhaseState(ctx, db.CreatePhaseStateParams{
		PhaseID:     phase.ID,
		Status:      "active",
		NextCheckAt: sql.NullTime{Time: nextCheckAt, Valid: true},
		Data:        data,
	})
}

func (ls *LadderStrategy) Resume(ctx context.Context, invite models.Invite, phase models.Phase, state *models.PhaseState) error {
	var data struct{ Index int }
	if err := json.Unmarshal(state.Data, &data); err != nil {
		return err
	}

	nextIndex := data.Index + 1
	if nextIndex >= len(ls.List) {
		return ls.Queries.UpdatePhaseState(ctx, db.UpdatePhaseStateParams{
			PhaseID:     phase.ID,
			Status:      "completed",
			NextCheckAt: sql.NullTime{Valid: false},
			Data:        state.Data,
		})
	}

	_, err := ls.Inviter.InvitePerson(ctx, invite, ls.List[nextIndex])
	if err != nil {
		return err
	}

	nextCheckAt := time.Now().Add(ls.Timeout)
	newData, _ := json.Marshal(map[string]int{"index": nextIndex})

	return ls.Queries.UpdatePhaseState(ctx, db.UpdatePhaseStateParams{
		PhaseID:     phase.ID,
		Status:      "active",
		NextCheckAt: sql.NullTime{Time: nextCheckAt, Valid: true},
		Data:        newData,
	})
}

func (ls *LadderStrategy) HandleEvent(ctx context.Context, invite models.Invite, phase models.Phase, state *models.PhaseState, event models.Event) error {
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

	invitee, err := ls.Queries.GetInvitee(ctx, event.InviteeID)
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
