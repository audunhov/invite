package app

import (
	"context"
	"encoding/json"
	"fmt"

	"invite/models"
)

func (app *App) GetDashboardStats(ctx context.Context) (*models.DashboardStats, error) {
	// 1. Fetch Global Stats
	dbStats, err := app.Queries.GetGlobalStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch global stats: %w", err)
	}

	// 2. Fetch Recent Activity
	dbActivity, err := app.Queries.GetRecentActivity(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent activity: %w", err)
	}

	activity := make([]models.RecentActivity, len(dbActivity))
	for i, a := range dbActivity {
		activity[i] = models.RecentActivity{
			Timestamp: a.Timestamp,
			Type:      a.Type,
			Message:   a.Message.(string),
		}
	}

	// 3. Fetch Active Phases for Bottlenecks
	activePhases, err := app.Queries.GetActivePhases(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active phases: %w", err)
	}

	bottlenecks := []models.Bottleneck{}
	for _, row := range activePhases {
		waitingFor := "Unknown"

		if row.StrategyKind == string(models.StrategyKindLadder) {
			var data struct{ Index int }
			if err := json.Unmarshal(row.PhaseData, &data); err == nil {
				var config struct {
					List []models.Person `json:"List"`
				}
				if err := json.Unmarshal(row.StrategyConfig, &config); err == nil {
					if data.Index >= 0 && data.Index < len(config.List) {
						waitingFor = config.List[data.Index].Name
					}
				}
			}
		} else if row.StrategyKind == string(models.StrategyKindSprint) {
			waitingFor = "Recipients"
		}

		bottlenecks = append(bottlenecks, models.Bottleneck{
			InviteID:     row.InviteID,
			Title:        row.Title,
			PhaseOrder:   int(row.Order),
			StrategyKind: row.StrategyKind,
			WaitingFor:   waitingFor,
			ActiveSince:  row.CreatedAt, // Note: This is invite created_at, not phase active_since. Close enough for now.
		})
	}

	return &models.DashboardStats{
		Stats: models.GlobalStats{
			ActiveInvites: int(dbStats.ActiveInvites),
			FailedEmails:  int(dbStats.FailedEmails),
			SuccessRate:   dbStats.SuccessRate,
		},
		Bottlenecks: bottlenecks,
		Activity:    activity,
	}, nil
}
