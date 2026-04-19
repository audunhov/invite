package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"invite/db"
)

func (app *App) RunOrchestrator(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := app.ProcessActivePhases(ctx); err != nil {
				fmt.Printf("Error processing phases: %v\n", err)
			}
		}
	}
}

func (app *App) ProcessActivePhases(ctx context.Context) error {
	// 1. Query DB for active phases where next_check_at <= NOW()
	rows, err := app.Queries.GetActivePhasesToProcess(ctx, sql.NullTime{Time: time.Now(), Valid: true})
	if err != nil {
		return fmt.Errorf("failed to get active phases: %w", err)
	}

	for _, row := range rows {
		// 2. For each phase:
		//    a. Load Invite and Phase data
		invite := Invite{
			ID:          row.InviteID,
			Title:       row.Title,
			Description: row.Description.String,
			From:        row.From,
			To:          row.To.Time,
			Duration:    time.Duration(row.Duration.Int64),
			CreatedAt:   row.CreatedAt,
			Status:      row.InviteStatus,
		}

		phase := Phase{
			ID:             row.PhaseID,
			InviteID:       row.InviteID,
			Order:          int(row.Order),
			StrategyKind:   row.StrategyKind,
			StrategyConfig: row.StrategyConfig,
		}

		state := &PhaseState{
			PhaseID:     row.PhaseID,
			Status:      row.PhaseStatus,
			NextCheckAt: nil,
			Data:        row.PhaseData,
		}
		if row.NextCheckAt.Valid {
			state.NextCheckAt = &row.NextCheckAt.Time
		}

		//    b. Load Strategy via LoadStrategy
		strategy, err := LoadStrategy(app, phase)
		if err != nil {
			fmt.Printf("Error loading strategy for phase %s: %v\n", row.PhaseID, err)
			continue
		}

		//    c. Call strategy.Resume(...)
		if err := strategy.Resume(ctx, invite, phase, state); err != nil {
			fmt.Printf("Error resuming strategy for phase %s: %v\n", row.PhaseID, err)
			continue
		}

		//    d. Update state in DB
		updateParams := db.UpdatePhaseStateParams{
			PhaseID: state.PhaseID,
			Status:  state.Status,
			Data:    state.Data,
		}
		if state.NextCheckAt != nil {
			updateParams.NextCheckAt = sql.NullTime{Time: *state.NextCheckAt, Valid: true}
		}

		if err := app.Queries.UpdatePhaseState(ctx, updateParams); err != nil {
			fmt.Printf("Error updating phase state for phase %s: %v\n", row.PhaseID, err)
		}
	}

	return nil
}
