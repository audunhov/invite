package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/lib/pq"
	"invite/db"
)

// StartTestDB spins up a Postgres container, runs migrations, and returns the DB connection.
func StartTestDB(t *testing.T) *sql.DB {
	ctx := context.Background()

	dbName := "testdb"
	dbUser := "user"
	dbPassword := "password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)

	// Clean up the container when the test is done
	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	dbConn, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	// Ensure DB is responsive
	require.NoError(t, dbConn.Ping())

	// Run migrations
	require.NoError(t, db.Migrate(ctx, dbConn))

	return dbConn
}
