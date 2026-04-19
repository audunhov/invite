package api_test

import (
	"context"
	"testing"

	"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"invite/api"
	"invite/db"
	"invite/testutil"
)

func TestCreatePersonAPI(t *testing.T) {
	dbConn := testutil.StartTestDB(t)
	queries := db.New(dbConn)
	server := &api.Server{Queries: queries}
	ctx := context.Background()

	// Call the handler directly (since it's a strict server interface)
	email := "new@test.com"
	name := "New User"
	req := api.CreatePersonRequestObject{
		Body: &api.NewPerson{
			Email: types.Email(email),
			Name:  name,
		},
	}

	res, err := server.CreatePerson(ctx, req)
	require.NoError(t, err)

	// Verify the response type
	successRes, ok := res.(api.CreatePerson201JSONResponse)
	require.True(t, ok, "Expected 201 response")
	require.Equal(t, string(successRes.Email), email)
	require.Equal(t, successRes.Name, name)
	require.NotEmpty(t, successRes.Id)

	// Verify the database record
	p, err := queries.GetPerson(ctx, successRes.Id)
	require.NoError(t, err)
	require.Equal(t, p.Email, email)
}
