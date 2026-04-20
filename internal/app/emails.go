package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"invite/db"
)

func (app *App) ProcessFailedEmails(ctx context.Context) error {
	failed, err := app.Queries.GetFailedEmails(ctx)
	if err != nil {
		return fmt.Errorf("failed to get failed emails: %w", err)
	}

	for _, log := range failed {
		slog.Info("Retrying failed email", slog.String("log_id", log.ID.String()), slog.String("recipient", log.RecipientEmail))
		err := app.EmailService.SendRaw(log.RecipientEmail, log.Subject, log.Body)

		status := "sent"
		var errMsg sql.NullString
		if err != nil {
			status = "failed"
			errMsg = sql.NullString{String: err.Error(), Valid: true}
			slog.Error("Retry failed", slog.String("log_id", log.ID.String()), slog.Any("error", err))
		} else {
			slog.Info("Retry successful", slog.String("log_id", log.ID.String()))
		}

		err = app.Queries.UpdateEmailLogStatus(ctx, db.UpdateEmailLogStatusParams{
			ID:           log.ID,
			Status:       status,
			ErrorMessage: errMsg,
		})
		if err != nil {
			slog.Error("Failed to update email log status after retry", slog.String("log_id", log.ID.String()), slog.Any("error", err))
		}
	}
	return nil
}
