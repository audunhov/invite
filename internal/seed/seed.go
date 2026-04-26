package seed

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"invite/db"
	"invite/internal/auth"
)

func Run(ctx context.Context, dbConn *sql.DB, queries *db.Queries) error {
	slog.Info("Starting database seed...")

	// Clear existing data to ensure idempotency
	tables := []string{
		"email_logs", "invite_phase_state", "invite_phases", "invite_tags", 
		"invitees", "invites", "group_members", "groups", "tags", "sessions", "persons",
	}
	for _, t := range tables {
		_, err := dbConn.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", t))
		if err != nil {
			return fmt.Errorf("truncating %s: %w", t, err)
		}
	}

	// 1. Create Admin User
	adminID := uuid.New()
	hash, _ := auth.HashPassword("password123")
	_, err := queries.CreatePerson(ctx, db.CreatePersonParams{
		ID:           adminID,
		Email:        "admin@example.com",
		Name:         "Admin User",
		PasswordHash: sql.NullString{String: hash, Valid: true},
	})
	if err != nil {
		return err
	}

	// 2. Create Sample Persons
	persons := []struct {
		ID    uuid.UUID
		Email string
		Name  string
	}{
		{uuid.New(), "alice@example.com", "Alice Smith"},
		{uuid.New(), "bob@example.com", "Bob Jones"},
		{uuid.New(), "carol@example.com", "Carol Williams"},
		{uuid.New(), "david@example.com", "David Brown"},
		{uuid.New(), "eve@example.com", "Eve Davis"},
	}

	for _, p := range persons {
		_, err = queries.CreatePerson(ctx, db.CreatePersonParams{
			ID:    p.ID,
			Email: p.Email,
			Name:  p.Name,
		})
		if err != nil {
			return err
		}
	}

	// 3. Create Sample Tags
	tags := []struct {
		ID    uuid.UUID
		Name  string
		Color string
	}{
		{uuid.New(), "Conference", "#3b82f6"}, // blue-500
		{uuid.New(), "Debate", "#ef4444"},     // red-500
		{uuid.New(), "Urgent", "#f59e0b"},     // amber-500
		{uuid.New(), "Follow-up", "#10b981"},  // emerald-500
	}

	for _, t := range tags {
		_, err = queries.CreateTag(ctx, db.CreateTagParams{
			ID:    t.ID,
			Name:  t.Name,
			Color: t.Color,
		})
		if err != nil {
			return err
		}
	}

	// 4. Create Sample Groups
	groupID := uuid.New()
	_, err = queries.CreateGroup(ctx, db.CreateGroupParams{
		ID:          groupID,
		Name:        "Reviewers",
		Description: sql.NullString{String: "Paper reviewers for the next conference", Valid: true},
	})
	if err != nil {
		return err
	}

	for i := 0; i < 3; i++ {
		err = queries.AddGroupMember(ctx, db.AddGroupMemberParams{
			ID:        uuid.New(),
			GroupID:   groupID,
			ContactID: persons[i].ID,
		})
		if err != nil {
			return err
		}
	}

	slog.Info("Database seeded successfully!")
	return nil
}
