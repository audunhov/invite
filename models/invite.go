package models

import (
	"time"

	"github.com/google/uuid"
)

type InviteeState string

const (
	InviteeStateAccepted InviteeState = "accepted"
	InviteeStatePending  InviteeState = "pending"
	InviteeStateDeclined InviteeState = "declined"
	InviteeStateExpired  InviteeState = "expired"
)

type InviteStatus string

const (
	InviteStatusPending   InviteStatus = "pending"
	InviteStatusActive    InviteStatus = "active"
	InviteStatusCompleted InviteStatus = "completed"
	InviteStatusCancelled InviteStatus = "cancelled"
)

type Invite struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	From         time.Time `json:"from"`
	To           time.Time `json:"to"`
	Duration     time.Duration `json:"duration"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
	FromPersonID uuid.UUID `json:"from_person_id"`
}

type Invitee struct {
	ID         uuid.UUID    `json:"id"`
	InviteID   uuid.UUID    `json:"invite_id"`
	ContactID  uuid.UUID    `json:"contact_id"`
	State      InviteeState `json:"state"`
	CreatedAt  time.Time    `json:"created_at"`
	MagicToken uuid.UUID    `json:"magic_token"`
}

type Event struct {
	Kind      string    `json:"kind"`
	InviteeID uuid.UUID `json:"invitee_id"`
}
