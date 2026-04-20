package strategy

import (
	"encoding/json"
	"fmt"

	"invite/db"
	"invite/internal/strategy/ladder"
	"invite/internal/strategy/sprint"
	"invite/models"
)

func LoadStrategy(queries *db.Queries, inviter models.Inviter, p models.Phase) (models.Strategy, error) {
	kind := models.StrategyKind(p.StrategyKind)
	switch kind {
	case models.StrategyKindLadder:
		var cfg ladder.LadderStrategy
		if err := json.Unmarshal(p.StrategyConfig, &cfg); err != nil {
			return nil, err
		}
		cfg.Queries = queries
		cfg.Inviter = inviter
		return &cfg, nil

	case models.StrategyKindSprint:
		var cfg sprint.SprintStrategy
		if err := json.Unmarshal(p.StrategyConfig, &cfg); err != nil {
			return nil, err
		}
		cfg.Queries = queries
		cfg.Inviter = inviter
		return &cfg, nil

	default:
		return nil, fmt.Errorf("unknown strategy: %s", p.StrategyKind)
	}
}
