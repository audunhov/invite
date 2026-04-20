package models

import "github.com/google/uuid"

type Group struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type GroupMember struct {
	ID        uuid.UUID `json:"id"`
	ContactID uuid.UUID `json:"contact_id"`
	GroupID   uuid.UUID `json:"group_id"`
}
