package models

import (
	"database/sql"

	"github.com/google/uuid"
)

type Person struct {
	ID                     uuid.UUID      `json:"id"`
	Email                  string         `json:"email"`
	Name                   string         `json:"name"`
	PasswordHash           sql.NullString `json:"-"`
	PasswordResetToken     sql.NullString `json:"-"`
	PasswordResetExpiresAt sql.NullTime   `json:"-"`
}
