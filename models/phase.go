package models

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

type StrategyKind string

const (
	StrategyKindLadder StrategyKind = "ladder"
	StrategyKindSprint StrategyKind = "sprint"
)

type Phase struct {
	ID             uuid.UUID       `json:"id"`
	InviteID       uuid.UUID       `json:"invite_id"`
	Order          int             `json:"order"`
	StrategyKind   string          `json:"strategy_kind"`
	StrategyConfig json.RawMessage `json:"strategy_config"`
}

type PhaseState struct {
	PhaseID     uuid.UUID       `json:"phase_id"`
	Status      string          `json:"status"`
	NextCheckAt sql.NullTime    `json:"next_check_at"`
	Data        json.RawMessage `json:"data"`
}
