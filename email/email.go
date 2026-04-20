package email

import (
	"fmt"
	"net/smtp"
	"invite/config"
	"invite/models"
)

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) SendInvite(recipient models.Person, sender models.Person, inviteTitle string, inviteDesc string, token string) error {
	if s.cfg.SMTPHost == "" {
		return nil
	}

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)
	to := []string{recipient.Email}
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s <%s>\r\n"+
		"Subject: Invite: %s\r\n"+
		"\r\n"+
		"Hi %s,\r\n\r\n%s\r\n\r\nRespond here: %s/respond/%s\r\n",
		recipient.Email, sender.Name, s.cfg.SMTPUser, inviteTitle, recipient.Name, inviteDesc, s.cfg.BaseURL, token))

	return smtp.SendMail(fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort), auth, s.cfg.SMTPUser, to, msg)
}

func (s *Service) SendResetPasswordEmail(email string, token string) error {
	if s.cfg.SMTPHost == "" {
		return nil
	}

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)
	to := []string{email}
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: Password Reset Request\r\n"+
		"\r\n"+
		"Hi,\r\n\r\nYou requested a password reset. Use the following link to reset it:\r\n\r\n%s/reset-password?token=%s\r\n",
		email, s.cfg.SMTPUser, s.cfg.BaseURL, token))

	return smtp.SendMail(fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort), auth, s.cfg.SMTPUser, to, msg)
}
