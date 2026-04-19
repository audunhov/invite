package api

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"invite/db"
)

type Server struct {
	Queries *db.Queries
}

var _ StrictServerInterface = (*Server)(nil)

func (s *Server) ListPersons(ctx context.Context, request ListPersonsRequestObject) (ListPersonsResponseObject, error) {
	persons, err := s.Queries.ListPersons(ctx)
	if err != nil {
		return nil, err
	}

	var res []Person
	for _, p := range persons {
		res = append(res, Person{
			Id:    p.ID,
			Email: p.Email,
			Name:  p.Name,
		})
	}

	return ListPersons200JSONResponse(res), nil
}

func (s *Server) CreatePerson(ctx context.Context, request CreatePersonRequestObject) (CreatePersonResponseObject, error) {
	newID := uuid.New()
	p, err := s.Queries.CreatePerson(ctx, db.CreatePersonParams{
		ID:    newID,
		Email: request.Body.Email,
		Name:  request.Body.Name,
	})
	if err != nil {
		return nil, err
	}

	return CreatePerson201JSONResponse(Person{
		Id:    p.ID,
		Email: p.Email,
		Name:  p.Name,
	}), nil
}

func (s *Server) GetPerson(ctx context.Context, request GetPersonRequestObject) (GetPersonResponseObject, error) {
	p, err := s.Queries.GetPerson(ctx, request.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetPerson404Response{}, nil
		}
		return nil, err
	}

	return GetPerson200JSONResponse(Person{
		Id:    p.ID,
		Email: p.Email,
		Name:  p.Name,
	}), nil
}
