package models

import (
	"time"

	"github.com/google/uuid"
)

type DashboardStats struct {
	Stats       GlobalStats      `json:"stats"`
	Bottlenecks []Bottleneck     `json:"bottlenecks"`
	Activity    []RecentActivity `json:"activity"`
}

type GlobalStats struct {
	ActiveInvites int     `json:"active_invites"`
	FailedEmails  int     `json:"failed_emails"`
	SuccessRate   float64 `json:"success_rate"`
}

type Bottleneck struct {
	InviteID     uuid.UUID `json:"invite_id"`
	Title        string    `json:"title"`
	PhaseOrder   int       `json:"phase_order"`
	StrategyKind string    `json:"strategy_kind"`
	WaitingFor   string    `json:"waiting_for"`
	ActiveSince  time.Time `json:"active_since"`
	Tags         []Tag     `json:"tags"`
}

type RecentActivity struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
}
