package email

import (
	"context"
	"database/sql"
	"fmt"
	"invite/config"
	"invite/db"
	"invite/models"
	"net/smtp"

	"github.com/google/uuid"
)

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) SendRaw(toEmail, subject, body string) error {
	if s.cfg.SMTPHost == "" {
		return nil
	}

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)
	to := []string{toEmail}
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s",
		toEmail, s.cfg.SMTPUser, subject, body))

	return smtp.SendMail(fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort), auth, s.cfg.SMTPUser, to, msg)
}

func (s *Service) LogAndSend(ctx context.Context, queries *db.Queries, inviteeID uuid.NullUUID, toEmail, subject, body string) error {
	logID := uuid.New()
	_, err := queries.CreateEmailLog(ctx, db.CreateEmailLogParams{
		ID:             logID,
		InviteeID:      inviteeID,
		RecipientEmail: toEmail,
		Subject:        subject,
		Body:           body,
		Status:         "pending",
	})
	if err != nil {
		return fmt.Errorf("failed to create email log: %w", err)
	}

	err = s.SendRaw(toEmail, subject, body)

	status := "sent"
	var errMsg sql.NullString
	if err != nil {
		status = "failed"
		errMsg = sql.NullString{String: err.Error(), Valid: true}
	}

	updateErr := queries.UpdateEmailLogStatus(ctx, db.UpdateEmailLogStatusParams{
		ID:           logID,
		Status:       status,
		ErrorMessage: errMsg,
	})
	if updateErr != nil {
		return fmt.Errorf("failed to update email log status: %w (original error: %v)", updateErr, err)
	}

	return err
}

func (s *Service) SendInvite(ctx context.Context, queries *db.Queries, recipient models.Person, sender models.Person, inviteTitle string, inviteDesc string, token string, inviteeID uuid.UUID) error {
	subject := fmt.Sprintf("Invite: %s", inviteTitle)
	body := fmt.Sprintf("Hi %s,\r\n\r\n%s\r\n\r\nRespond here: %s/respond/%s\r\n",
		recipient.Name, inviteDesc, s.cfg.BaseURL, token)

	return s.LogAndSend(ctx, queries, uuid.NullUUID{UUID: inviteeID, Valid: true}, recipient.Email, subject, body)
}

func (s *Service) SendResetPasswordEmail(ctx context.Context, queries *db.Queries, email string, token string) error {
	subject := "Password Reset Request"
	body := fmt.Sprintf("Hi,\r\n\r\nYou requested a password reset. Use the following link to reset it:\r\n\r\n%s/reset-password?token=%s\r\n",
		s.cfg.BaseURL, token)

	return s.LogAndSend(ctx, queries, uuid.NullUUID{}, email, subject, body)
}
