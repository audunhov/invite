package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"invite/db"
	"invite/internal/app"
	"invite/internal/strategy/sprint"
	"invite/models"
	"invite/testutil"
)

func TestSprintStrategy_Execute(t *testing.T) {
	dbConn := testutil.StartTestDB(t)
	queries := db.New(dbConn)
	application := app.New(dbConn, queries, nil)
	ctx := context.Background()

	// 1. Setup test data (Persons)
	person1ID := uuid.New()
	person2ID := uuid.New()
	
	_, err := queries.CreatePerson(ctx, db.CreatePersonParams{ID: person1ID, Email: "p1@test.com", Name: "Person 1"})
	require.NoError(t, err)
	_, err = queries.CreatePerson(ctx, db.CreatePersonParams{ID: person2ID, Email: "p2@test.com", Name: "Person 2"})
	require.NoError(t, err)

	// 2. Setup Invite and Phase
	inviteID := uuid.New()
	_, err = dbConn.ExecContext(ctx, `INSERT INTO invites (id, title, "from", created_at, status) VALUES ($1, 'Test Invite', NOW(), NOW(), 'active')`, inviteID)
	require.NoError(t, err)

	phaseID := uuid.New()
	cfgBytes, _ := json.Marshal(map[string]interface{}{})
	_, err = dbConn.ExecContext(ctx, `INSERT INTO invite_phases (id, invite_id, "order", strategy_kind, strategy_config) VALUES ($1, $2, 1, 'sprint', $3)`, phaseID, inviteID, cfgBytes)
	require.NoError(t, err)

	invite := models.Invite{ID: inviteID, Status: "active"}
	phase := models.Phase{ID: phaseID}

	// 3. Initialize SprintStrategy
	strategy := &sprint.SprintStrategy{
		Queries:    queries,
		Inviter:    application,
		Recipients: []uuid.UUID{person1ID, person2ID},
		Deadline:   time.Now().Add(1 * time.Hour),
	}

	// 4. Execute Strategy
	err = strategy.Execute(ctx, invite, phase)
	require.NoError(t, err)

	// 5. Verify Results (Invitees created)
	var count int
	err = dbConn.QueryRowContext(ctx, "SELECT COUNT(*) FROM invitees WHERE invite_id = $1", inviteID).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 2, count, "Should have created 2 invitees")

	// 6. Verify PhaseState created
	var phaseStatus string
	err = dbConn.QueryRowContext(ctx, "SELECT status FROM invite_phase_state WHERE phase_id = $1", phaseID).Scan(&phaseStatus)
	require.NoError(t, err)
	require.Equal(t, "active", phaseStatus)
}
