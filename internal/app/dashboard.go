package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
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
			Tags:         app.getTagsForInvite(ctx, row.InviteID),
		})
	}

	// 4. Fetch Timeline Data
	timelineRows, err := app.Queries.GetTimelineData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timeline data: %w", err)
	}

	timelineMap := make(map[uuid.UUID]*models.TimelineInvite)
	timelineOrder := []uuid.UUID{}

	for _, row := range timelineRows {
		if _, exists := timelineMap[row.InviteID]; !exists {
			invite := &models.TimelineInvite{
				ID:     row.InviteID,
				Title:  row.Title,
				Status: row.InviteStatus,
				Phases: []models.TimelinePhase{},
			}
			timelineMap[row.InviteID] = invite
			timelineOrder = append(timelineOrder, row.InviteID)
		}

		timelineMap[row.InviteID].Phases = append(timelineMap[row.InviteID].Phases, models.TimelinePhase{
			Order:         int(row.PhaseOrder),
			Status:        row.PhaseStatus,
			AcceptedCount: int(row.AcceptedCount),
			DeclinedCount: int(row.DeclinedCount),
			TotalInvitees: int(row.TotalInvitees),
		})
	}

	timeline := make([]models.TimelineInvite, len(timelineOrder))
	for i, id := range timelineOrder {
		timeline[i] = *timelineMap[id]
	}

	return &models.DashboardStats{
		Stats: models.GlobalStats{
			ActiveInvites: int(dbStats.ActiveInvites),
			FailedEmails:  int(dbStats.FailedEmails),
			SuccessRate:   dbStats.SuccessRate,
		},
		Timeline:    timeline,
		Bottlenecks: bottlenecks,
		Activity:    activity,
	}, nil
}

func (app *App) getTagsForInvite(ctx context.Context, inviteID uuid.UUID) []models.Tag {
	dbTags, err := app.Queries.GetTagsByInvite(ctx, inviteID)
	if err != nil {
		return []models.Tag{}
	}

	tags := make([]models.Tag, len(dbTags))
	for i, t := range dbTags {
		tags[i] = models.Tag{
			ID:    t.ID,
			Name:  t.Name,
			Color: t.Color,
		}
	}
	return tags
}
