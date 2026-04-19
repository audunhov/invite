package api_test

import (
	"context"
	"testing"
	"time"

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

	successRes, ok := res.(api.CreatePerson201JSONResponse)
	require.True(t, ok, "Expected 201 response")
	require.Equal(t, string(successRes.Email), email)
	require.Equal(t, successRes.Name, name)
	require.NotEmpty(t, successRes.Id)

	p, err := queries.GetPerson(ctx, successRes.Id)
	require.NoError(t, err)
	require.Equal(t, p.Email, email)

	// Test Update
	newName := "Updated Name"
	updateReq := api.UpdatePersonRequestObject{
		Id: successRes.Id,
		Body: &api.UpdatePerson{
			Name: &newName,
		},
	}
	updateRes, err := server.UpdatePerson(ctx, updateReq)
	require.NoError(t, err)
	updatedPerson := updateRes.(api.UpdatePerson200JSONResponse)
	require.Equal(t, updatedPerson.Name, newName)

	// Test Delete
	deleteRes, err := server.DeletePerson(ctx, api.DeletePersonRequestObject{Id: successRes.Id})
	require.NoError(t, err)
	require.IsType(t, api.DeletePerson204Response{}, deleteRes)

	_, err = queries.GetPerson(ctx, successRes.Id)
	require.Error(t, err)
}

func TestInviteAPI(t *testing.T) {
	dbConn := testutil.StartTestDB(t)
	queries := db.New(dbConn)
	server := &api.Server{Queries: queries}
	ctx := context.Background()

	title := "Test Invite"
	from := time.Now().Add(24 * time.Hour).Truncate(time.Microsecond)
	
	req := api.CreateInviteRequestObject{
		Body: &api.NewInvite{
			Title: title,
			From:  from,
		},
	}

	res, err := server.CreateInvite(ctx, req)
	require.NoError(t, err)

	successRes, ok := res.(api.CreateInvite201JSONResponse)
	require.True(t, ok, "Expected 201 response")
	require.Equal(t, successRes.Title, title)
	require.True(t, successRes.From.Equal(from))
	require.Equal(t, string(successRes.Status), "pending")

	// Get Invite
	getRes, err := server.GetInvite(ctx, api.GetInviteRequestObject{Id: successRes.Id})
	require.NoError(t, err)
	
	getSuccessRes, ok := getRes.(api.GetInvite200JSONResponse)
	require.True(t, ok)
	require.Equal(t, getSuccessRes.Id, successRes.Id)

	// Update Invite
	newTitle := "Updated Title"
	newStatus := api.Active
	updateRes, err := server.UpdateInvite(ctx, api.UpdateInviteRequestObject{
		Id: successRes.Id,
		Body: &api.UpdateInvite{
			Title:  &newTitle,
			Status: &newStatus,
		},
	})
	require.NoError(t, err)
	updatedInvite := updateRes.(api.UpdateInvite200JSONResponse)
	require.Equal(t, updatedInvite.Title, newTitle)
	require.Equal(t, string(updatedInvite.Status), string(newStatus))

	// Delete Invite
	_, err = server.DeleteInvite(ctx, api.DeleteInviteRequestObject{Id: successRes.Id})
	require.NoError(t, err)
}

func TestGroupAPI(t *testing.T) {
	dbConn := testutil.StartTestDB(t)
	queries := db.New(dbConn)
	server := &api.Server{Queries: queries}
	ctx := context.Background()

	// 1. Create Group
	name := "Test Group"
	desc := "A group for testing"
	req := api.CreateGroupRequestObject{
		Body: &api.NewGroup{
			Name:        name,
			Description: &desc,
		},
	}

	res, err := server.CreateGroup(ctx, req)
	require.NoError(t, err)

	successRes, ok := res.(api.CreateGroup201JSONResponse)
	require.True(t, ok)
	require.Equal(t, successRes.Name, name)
	require.Equal(t, *successRes.Description, desc)

	// 2. Create Person
	pRes, _ := server.CreatePerson(ctx, api.CreatePersonRequestObject{
		Body: &api.NewPerson{Email: "member@test.com", Name: "Member"},
	})
	person := pRes.(api.CreatePerson201JSONResponse)

	// 3. Add Member
	_, err = server.AddGroupMember(ctx, api.AddGroupMemberRequestObject{
		Id: successRes.Id,
		Body: &api.AddGroupMemberRequest{
			PersonId: person.Id,
		},
	})
	require.NoError(t, err)

	// 4. List Members
	listRes, err := server.ListGroupMembers(ctx, api.ListGroupMembersRequestObject{Id: successRes.Id})
	require.NoError(t, err)
	
	listSuccessRes, ok := listRes.(api.ListGroupMembers200JSONResponse)
	require.True(t, ok)
	require.Len(t, listSuccessRes, 1)
	require.Equal(t, listSuccessRes[0].Id, person.Id)

	// 5. Remove Member
	_, err = server.RemoveGroupMember(ctx, api.RemoveGroupMemberRequestObject{
		Id:       successRes.Id,
		PersonId: person.Id,
	})
	require.NoError(t, err)

	listRes2, _ := server.ListGroupMembers(ctx, api.ListGroupMembersRequestObject{Id: successRes.Id})
	listSuccessRes2 := listRes2.(api.ListGroupMembers200JSONResponse)
	require.Len(t, listSuccessRes2, 0)

	// 6. Delete Group
	_, err = server.DeleteGroup(ctx, api.DeleteGroupRequestObject{Id: successRes.Id})
	require.NoError(t, err)
}
