package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"invite/db"
	"invite/internal/strategy"
	"invite/models"
)

func (app *App) InvitePerson(ctx context.Context, i models.Invite, phase models.Phase, p models.Person) (*models.Invitee, error) {
	inviteeID := uuid.New()
	magicToken := uuid.New()

	phaseID := uuid.NullUUID{UUID: phase.ID, Valid: true}
	if phase.ID == uuid.Nil {
		phaseID.Valid = false
	}

	err := app.Queries.CreateInvitee(ctx, db.CreateInviteeParams{
		ID:         inviteeID,
		InviteID:   i.ID,
		PhaseID:    phaseID,
		ContactID:  p.ID,
		State:      string(models.InviteeStatePending),
		CreatedAt:  time.Now(),
		MagicToken: magicToken,
	})
	if err != nil {
		return nil, err
	}

	go func() {
		s, err := app.Queries.GetPerson(context.Background(), i.FromPersonID)
		if err == nil {
			sender := models.Person{
				ID:                     s.ID,
				Email:                  s.Email,
				Name:                   s.Name,
				PasswordHash:           s.PasswordHash,
				PasswordResetToken:     s.PasswordResetToken,
				PasswordResetExpiresAt: s.PasswordResetExpiresAt,
			}
			if err := app.EmailService.SendInvite(context.Background(), app.Queries, p, sender, i.Title, i.Description, magicToken.String(), inviteeID); err != nil {
				slog.Error("Failed to send invite email", slog.Any("error", err), slog.String("invite_id", i.ID.String()), slog.String("recipient", p.Email))
			}
		} else {
			slog.Error("Failed to fetch sender for email", slog.Any("error", err), slog.String("person_id", i.FromPersonID.String()))
		}
	}()

	return &models.Invitee{
		ID:         inviteeID,
		InviteID:   i.ID,
		ContactID:  p.ID,
		State:      models.InviteeStatePending,
		CreatedAt:  time.Now(),
		MagicToken: magicToken,
	}, nil
}

type CreateInviteDeepParams struct {
	Invite db.CreateInviteParams
	TagIDs []uuid.UUID
	Phases []db.CreateInvitePhaseParams
}

func (app *App) CreateInviteDeep(ctx context.Context, params CreateInviteDeepParams) (*db.Invite, error) {
	tx, err := app.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := app.Queries.WithTx(tx)

	// 1. Create Invite
	i, err := qtx.CreateInvite(ctx, params.Invite)
	if err != nil {
		return nil, fmt.Errorf("creating invite: %w", err)
	}

	// 2. Add Tags
	if len(params.TagIDs) > 0 {
		err = qtx.AddInviteTags(ctx, db.AddInviteTagsParams{
			InviteID: i.ID,
			Column2:  params.TagIDs,
		})
		if err != nil {
			return nil, fmt.Errorf("adding tags: %w", err)
		}
	}

	// 3. Add Phases
	for _, p := range params.Phases {
		p.InviteID = i.ID // Ensure ID matches
		_, err = qtx.CreateInvitePhase(ctx, p)
		if err != nil {
			return nil, fmt.Errorf("creating phase %d: %w", p.Order, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &i, nil
}

func (app *App) StartInviteProcess(ctx context.Context, inviteID uuid.UUID) error {
	// 1. Get Invite
	i, err := app.Queries.GetInvite(ctx, inviteID)
	if err != nil {
		return fmt.Errorf("failed to get invite: %w", err)
	}

	if i.Status != "pending" {
		return fmt.Errorf("invite is already %s", i.Status)
	}

	// 2. Get First Phase
	p, err := app.Queries.GetFirstInvitePhase(ctx, inviteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("invite has no phases")
		}
		return fmt.Errorf("failed to get first phase: %w", err)
	}

	// 3. Map DB models to internal models
	inviteModel := models.Invite{
		ID:           i.ID,
		Title:        i.Title,
		Description:  i.Description.String,
		From:         i.From,
		To:           i.To.Time,
		Duration:     time.Duration(i.Duration.Int64),
		CreatedAt:    i.CreatedAt,
		Status:       i.Status,
		FromPersonID: i.FromPersonID.UUID,
	}

	phaseModel := models.Phase{
		ID:             p.ID,
		InviteID:       p.InviteID,
		Order:          int(p.Order),
		StrategyKind:   p.StrategyKind,
		StrategyConfig: p.StrategyConfig,
	}

	// 4. Load Strategy
	s, err := strategy.LoadStrategy(app.Queries, app, phaseModel)
	if err != nil {
		return fmt.Errorf("failed to load strategy: %w", err)
	}

	// 5. Update Invite Status to Active
	_, err = app.Queries.UpdateInvite(ctx, db.UpdateInviteParams{
		ID:     inviteID,
		Status: sql.NullString{String: "active", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to mark invite active: %w", err)
	}

	// 6. Execute Strategy
	if err := s.Execute(ctx, inviteModel, phaseModel); err != nil {
		slog.Error("Failed to execute initial strategy", slog.Any("error", err))
		return fmt.Errorf("strategy execution failed: %w", err)
	}

	// 7. Check if phase completed immediately
	_, err = app.Queries.GetActivePhaseForInvite(ctx, inviteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			app.AdvanceToNextPhase(ctx, inviteModel, phaseModel)
		}
	}

	return nil
}

func (app *App) GetPhaseProgress(ctx context.Context, row db.GetActivePhaseForInviteRow) (string, error) {
	phase := models.Phase{
		ID:             row.PhaseID,
		InviteID:       row.InviteID,
		Order:          int(row.Order),
		StrategyKind:   row.StrategyKind,
		StrategyConfig: row.StrategyConfig,
	}

	state := &models.PhaseState{
		PhaseID: row.PhaseID,
		Status:  row.PhaseStatus,
		Data:    row.PhaseData,
	}
	if row.NextCheckAt.Valid {
		state.NextCheckAt = sql.NullTime{Time: row.NextCheckAt.Time, Valid: true}
	}

	s, err := strategy.LoadStrategy(app.Queries, app, phase)
	if err != nil {
		return "", err
	}

	return s.Progress(state), nil
}

func (app *App) AdvanceToNextPhase(ctx context.Context, invite models.Invite, currentPhase models.Phase) error {
	slog.Info("Advancing to next phase", slog.String("invite_id", invite.ID.String()), slog.Int("current_order", currentPhase.Order))

	// 1. Find next phase
	next, err := app.Queries.GetNextInvitePhase(ctx, db.GetNextInvitePhaseParams{
		InviteID: invite.ID,
		Order:    int32(currentPhase.Order),
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Info("No more phases, completing invite", slog.String("invite_id", invite.ID.String()))
			// No more phases, mark invite as completed
			_, err = app.Queries.UpdateInvite(ctx, db.UpdateInviteParams{
				ID:     invite.ID,
				Status: sql.NullString{String: "completed", Valid: true},
			})
			return err
		}
		return fmt.Errorf("failed to get next phase: %w", err)
	}

	slog.Info("Found next phase", slog.String("phase_id", next.ID.String()), slog.Int("order", int(next.Order)))

	// 2. Prepare next phase models
	nextPhaseModel := models.Phase{
		ID:             next.ID,
		InviteID:       next.InviteID,
		Order:          int(next.Order),
		StrategyKind:   next.StrategyKind,
		StrategyConfig: next.StrategyConfig,
	}

	// 3. Load and Execute next strategy
	s, err := strategy.LoadStrategy(app.Queries, app, nextPhaseModel)
	if err != nil {
		return fmt.Errorf("failed to load next strategy: %w", err)
	}

	if err := s.Execute(ctx, invite, nextPhaseModel); err != nil {
		return fmt.Errorf("failed to execute next phase: %w", err)
	}

	// 4. Check if next phase completed immediately too
	_, err = app.Queries.GetActivePhaseForInvite(ctx, invite.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Info("Next phase completed immediately, advancing again", slog.String("invite_id", invite.ID.String()))
			return app.AdvanceToNextPhase(ctx, invite, nextPhaseModel)
		}
	}

	return nil
}

func (app *App) HandleInviteeResponse(ctx context.Context, token uuid.UUID, newState string) error {
	slog.Info("Handling invitee response", slog.String("token", token.String()), slog.String("state", newState))
	// 1. Get Invitee and Invite
	i, err := app.Queries.GetInviteeByToken(ctx, token)
	if err != nil {
		return err
	}

	// 2. Update state
	err = app.Queries.RespondToInvite(ctx, db.RespondToInviteParams{
		MagicToken: token,
		State:      newState,
	})
	if err != nil {
		return err
	}

	// 3. Trigger Strategy Event if active
	if newState == "accepted" || newState == "declined" {
		activeRow, err := app.Queries.GetActivePhaseForInvite(ctx, i.InviteID)
		if err == nil {
			// Trigger event
			event := models.Event{
				Kind:      "invitee_" + newState,
				InviteeID: i.ID,
			}

			// Get full invite.
			fullInvite, err := app.Queries.GetInvite(ctx, i.InviteID)
			if err != nil {
				return err
			}
			invite := models.Invite{
				ID:           fullInvite.ID,
				Title:        fullInvite.Title,
				Description:  fullInvite.Description.String,
				From:         fullInvite.From,
				To:           fullInvite.To.Time,
				Duration:     time.Duration(fullInvite.Duration.Int64),
				CreatedAt:    fullInvite.CreatedAt,
				Status:       fullInvite.Status,
				FromPersonID: fullInvite.FromPersonID.UUID,
			}

			phase := models.Phase{
				ID:             activeRow.PhaseID,
				InviteID:       activeRow.InviteID,
				Order:          int(activeRow.Order),
				StrategyKind:   activeRow.StrategyKind,
				StrategyConfig: activeRow.StrategyConfig,
			}

			state := &models.PhaseState{
				PhaseID:     activeRow.PhaseID,
				Status:      activeRow.PhaseStatus,
				NextCheckAt: activeRow.NextCheckAt,
				Data:        activeRow.PhaseData,
			}

			s, err := strategy.LoadStrategy(app.Queries, app, phase)
			if err == nil {
				slog.Info("Triggering strategy event", slog.String("kind", event.Kind), slog.String("phase_id", phase.ID.String()))
				if err := s.HandleEvent(ctx, invite, phase, state, event); err == nil {
					// Update state in DB
					updateParams := db.UpdatePhaseStateParams{
						PhaseID:     state.PhaseID,
						Status:      state.Status,
						NextCheckAt: state.NextCheckAt,
						Data:        state.Data,
					}
					if err := app.Queries.UpdatePhaseState(ctx, updateParams); err != nil {
						slog.Error("Failed to update phase state after event", slog.Any("error", err))
					}

					// If completed, trigger next phase
					if state.Status == "completed" {
						// Logic change: if it was an acceptance, we mark the WHOLE invite as completed and DON'T advance.
						if event.Kind == "invitee_accepted" {
							slog.Info("Invite accepted, marking entire process as completed", slog.String("invite_id", invite.ID.String()))
							_, err = app.Queries.UpdateInvite(ctx, db.UpdateInviteParams{
								ID:     invite.ID,
								Status: sql.NullString{String: "completed", Valid: true},
							})
							if err != nil {
								slog.Error("Failed to mark invite as completed after acceptance", slog.Any("error", err))
							}
							return nil // Stop here
						}

						slog.Info("Phase completed after event, advancing", slog.String("phase_id", phase.ID.String()))
						if err := app.AdvanceToNextPhase(ctx, invite, phase); err != nil {
							slog.Error("Failed to advance phase after event", slog.Any("error", err))
						}
					}
				} else {
					slog.Error("Strategy failed to handle event", slog.Any("error", err))
				}
			}
		}
	}

	return nil
}

func (app *App) InvalidateInvite(ctx context.Context, inviteID uuid.UUID) error {
	slog.Info("Invalidating invite", slog.String("invite_id", inviteID.String()))

	// 1. Mark invite as cancelled
	_, err := app.Queries.UpdateInvite(ctx, db.UpdateInviteParams{
		ID:     inviteID,
		Status: sql.NullString{String: "cancelled", Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to cancel invite: %w", err)
	}

	// 2. Delete all phase states for this invite
	err = app.Queries.DeleteInvitePhaseStates(ctx, inviteID)
	if err != nil {
		return fmt.Errorf("failed to delete phase states: %w", err)
	}

	return nil
}

func (app *App) InvalidatePhase(ctx context.Context, inviteID uuid.UUID, phaseID uuid.UUID) error {
	slog.Info("Invalidating phase", slog.String("invite_id", inviteID.String()), slog.String("phase_id", phaseID.String()))

	// 1. Get phase info before deletion
	p, err := app.Queries.GetInvitePhase(ctx, phaseID)
	if err != nil {
		return fmt.Errorf("failed to get phase: %w", err)
	}

	// 2. Delete the phase state (if it exists)
	// We need a specific query for this or just use the general one if we don't care about others.
	// Actually, let's add DeletePhaseState query.
	err = app.Queries.DeletePhaseState(ctx, phaseID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to delete phase state: %w", err)
	}

	// 3. Delete the phase itself
	err = app.Queries.DeleteInvitePhase(ctx, db.DeleteInvitePhaseParams{
		ID:       phaseID,
		InviteID: inviteID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete phase: %w", err)
	}

	// 4. If the invite is active, we might need to advance to the next phase
	i, err := app.Queries.GetInvite(ctx, inviteID)
	if err == nil && i.Status == "active" {
		// Check if any other phase is active
		_, err := app.Queries.GetActivePhaseForInvite(ctx, inviteID)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			slog.Info("Active phase deleted, advancing to next available phase", slog.String("invite_id", inviteID.String()))
			
			// Re-map current phase enough to find the next one
			currentPhase := models.Phase{
				ID:       p.ID,
				InviteID: p.InviteID,
				Order:    int(p.Order),
			}
			
			// Map full invite for AdvanceToNextPhase
			inviteModel := models.Invite{
				ID:           i.ID,
				Title:        i.Title,
				Description:  i.Description.String,
				From:         i.From,
				To:           i.To.Time,
				Duration:     time.Duration(i.Duration.Int64),
				CreatedAt:    i.CreatedAt,
				Status:       i.Status,
				FromPersonID: i.FromPersonID.UUID,
			}

			return app.AdvanceToNextPhase(ctx, inviteModel, currentPhase)
		}
	}

	return nil
}
