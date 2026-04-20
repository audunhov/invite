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

func (app *App) InvitePerson(ctx context.Context, i models.Invite, p models.Person) (*models.Invitee, error) {
	inviteeID := uuid.New()
	magicToken := uuid.New()
	err := app.Queries.CreateInvitee(ctx, db.CreateInviteeParams{
		ID:         inviteeID,
		InviteID:   i.ID,
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
			if err := app.EmailService.SendInvite(p, sender, i.Title, i.Description, magicToken.String()); err != nil {
				slog.Error("Failed to send invite email", slog.Any("error", err), slog.String("invite_id", i.ID.String()), slog.String("recipient", p.Email))
			}
		} else {
			slog.Error("Failed to fetch sender for email", slog.Any("error", err), slog.String("person_id", i.FromPersonID.String()))
		}
	}()

	return &models.Invitee{
		ID:        inviteeID,
		InviteID:  i.ID,
		ContactID: p.ID,
		State:     models.InviteeStatePending,
		CreatedAt: time.Now(),
		MagicToken: magicToken,
	}, nil
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
