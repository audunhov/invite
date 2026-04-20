package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"invite/db"
	"invite/email"
	"invite/internal/auth"
	"invite/internal/limiter"
	"invite/models"
)

type Server struct {
	Queries         *db.Queries
	StartInviteFunc func(ctx context.Context, inviteID uuid.UUID) error
	GetProgressFunc func(ctx context.Context, row db.GetActivePhaseForInviteRow) (string, error)
	Limiter         *limiter.IPRateLimiter
	EmailService    *email.Service
}

type contextKey string

const (
	personContextKey    contextKey = "person"
	sessionIDContextKey contextKey = "session_id"
)

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		sessionID, err := uuid.Parse(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		session, err := s.Queries.GetSession(r.Context(), sessionID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), personContextKey, models.Person{
			ID:    session.PersonID,
			Email: session.Email,
			Name:  session.Name,
		})
		ctx = context.WithValue(ctx, sessionIDContextKey, sessionID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type LoginSuccessResponse struct {
	SessionID uuid.UUID
}

func (r LoginSuccessResponse) VisitLoginResponse(w http.ResponseWriter) error {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    r.SessionID.String(),
		Path:     "/",
		Expires:  time.Now().Add(24 * 7 * time.Hour), // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(200)
	return nil
}

type LogoutSuccessResponse struct{}

func (r LogoutSuccessResponse) VisitLogoutResponse(w http.ResponseWriter) error {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(204)
	return nil
}

// Auth Handlers
func (s *Server) Login(ctx context.Context, request LoginRequestObject) (LoginResponseObject, error) {
	p, err := s.Queries.GetPersonByEmail(ctx, string(request.Body.Email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Login401Response{}, nil
		}
		return nil, err
	}

	if !p.PasswordHash.Valid || !auth.CheckPassword(request.Body.Password, p.PasswordHash.String) {
		return Login401Response{}, nil
	}

	sessionID := uuid.New()
	_, err = s.Queries.CreateSession(ctx, db.CreateSessionParams{
		ID:        sessionID,
		PersonID:  p.ID,
		ExpiresAt: time.Now().Add(24 * 7 * time.Hour),
	})
	if err != nil {
		return nil, err
	}

	return LoginSuccessResponse{SessionID: sessionID}, nil
}

func (s *Server) Logout(ctx context.Context, request LogoutRequestObject) (LogoutResponseObject, error) {
	sessionID, ok := ctx.Value(sessionIDContextKey).(uuid.UUID)
	if ok {
		s.Queries.DeleteSession(ctx, sessionID)
	}
	return LogoutSuccessResponse{}, nil
}

func (s *Server) ForgotPassword(ctx context.Context, request ForgotPasswordRequestObject) (ForgotPasswordResponseObject, error) {
	p, err := s.Queries.GetPersonByEmail(ctx, string(request.Body.Email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ForgotPassword204Response{}, nil
		}
		return nil, err
	}

	token := auth.GenerateSecureToken()
	_, err = s.Queries.UpdatePersonAuth(ctx, db.UpdatePersonAuthParams{
		ID:                     p.ID,
		PasswordResetToken:     sql.NullString{String: token, Valid: true},
		PasswordResetExpiresAt: sql.NullTime{Time: time.Now().Add(1 * time.Hour), Valid: true},
	})
	if err != nil {
		return nil, err
	}

	go func() {
		if err := s.EmailService.SendResetPasswordEmail(p.Email, token); err != nil {
			slog.Error("Failed to send reset password email", slog.Any("error", err), slog.String("email", p.Email))
		}
	}()

	return ForgotPassword204Response{}, nil
}

func (s *Server) ResetPassword(ctx context.Context, request ResetPasswordRequestObject) (ResetPasswordResponseObject, error) {
	p, err := s.Queries.GetPersonByResetToken(ctx, sql.NullString{String: request.Body.Token, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ResetPassword400Response{}, nil
		}
		return nil, err
	}

	hash, err := auth.HashPassword(request.Body.Password)
	if err != nil {
		return nil, err
	}

	_, err = s.Queries.UpdatePersonAuth(ctx, db.UpdatePersonAuthParams{
		ID:                     p.ID,
		PasswordHash:           sql.NullString{String: hash, Valid: true},
		PasswordResetToken:     sql.NullString{Valid: false},
		PasswordResetExpiresAt: sql.NullTime{Valid: false},
	})
	if err != nil {
		return nil, err
	}

	return ResetPassword204Response{}, nil
}

func (s *Server) GetMe(ctx context.Context, request GetMeRequestObject) (GetMeResponseObject, error) {
	p, ok := ctx.Value(personContextKey).(models.Person)
	if !ok {
		return GetMe401Response{}, nil
	}

	return GetMe200JSONResponse(Person{
		Id:          p.ID,
		Email:       types.Email(p.Email),
		Name:        p.Name,
		HasPassword: true, // If they are logged in, they have a password
	}), nil
}

var _ StrictServerInterface = (*Server)(nil)

// Helper functions for conversions
func toStringPtr(s sql.NullString) *string {
	if s.Valid {
		return &s.String
	}
	return nil
}

func fromStringPtr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func toTimePtr(t sql.NullTime) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

func fromTimePtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func toIntPtr(i sql.NullInt64) *int {
	if i.Valid {
		val := int(i.Int64)
		return &val
	}
	return nil
}

func fromIntPtr(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

// Person Handlers
func (s *Server) ListPersons(ctx context.Context, request ListPersonsRequestObject) (ListPersonsResponseObject, error) {
	persons, err := s.Queries.ListPersons(ctx)
	if err != nil {
		return nil, err
	}

	res := []Person{}
	for _, p := range persons {
		res = append(res, Person{
			Id:          p.ID,
			Email:       types.Email(p.Email),
			Name:        p.Name,
			HasPassword: p.PasswordHash.Valid,
		})
	}

	return ListPersons200JSONResponse(res), nil
}

func (s *Server) CreatePerson(ctx context.Context, request CreatePersonRequestObject) (CreatePersonResponseObject, error) {
	newID := uuid.New()
	params := db.CreatePersonParams{
		ID:    newID,
		Email: string(request.Body.Email),
		Name:  request.Body.Name,
	}

	if request.Body.Password != nil {
		hash, err := auth.HashPassword(*request.Body.Password)
		if err != nil {
			return nil, err
		}
		params.PasswordHash = sql.NullString{String: hash, Valid: true}
	}

	p, err := s.Queries.CreatePerson(ctx, params)
	if err != nil {
		return nil, err
	}

	return CreatePerson201JSONResponse(Person{
		Id:          p.ID,
		Email:       types.Email(p.Email),
		Name:        p.Name,
		HasPassword: p.PasswordHash.Valid,
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
		Id:          p.ID,
		Email:       types.Email(p.Email),
		Name:        p.Name,
		HasPassword: p.PasswordHash.Valid,
	}), nil
}

func (s *Server) UpdatePerson(ctx context.Context, request UpdatePersonRequestObject) (UpdatePersonResponseObject, error) {
	params := db.UpdatePersonParams{
		ID: request.Id,
	}
	if request.Body.Email != nil {
		params.Email = sql.NullString{String: string(*request.Body.Email), Valid: true}
	}
	if request.Body.Name != nil {
		params.Name = sql.NullString{String: *request.Body.Name, Valid: true}
	}
	if request.Body.Password != nil {
		hash, err := auth.HashPassword(*request.Body.Password)
		if err != nil {
			return nil, err
		}
		params.PasswordHash = sql.NullString{String: hash, Valid: true}
	}

	p, err := s.Queries.UpdatePerson(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UpdatePerson404Response{}, nil
		}
		return nil, err
	}

	return UpdatePerson200JSONResponse(Person{
		Id:          p.ID,
		Email:       types.Email(p.Email),
		Name:        p.Name,
		HasPassword: p.PasswordHash.Valid,
	}), nil
}

func (s *Server) DeletePerson(ctx context.Context, request DeletePersonRequestObject) (DeletePersonResponseObject, error) {
	// Security: check if it's the last admin
	p, err := s.Queries.GetPerson(ctx, request.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeletePerson404Response{}, nil
		}
		return nil, err
	}

	if p.PasswordHash.Valid {
		count, err := s.Queries.CountAdmins(ctx)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, errors.New("cannot delete the last administrative user")
		}
	}

	err = s.Queries.DeletePerson(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	return DeletePerson204Response{}, nil
}

// Group Handlers
func (s *Server) ListGroups(ctx context.Context, request ListGroupsRequestObject) (ListGroupsResponseObject, error) {
	groups, err := s.Queries.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	res := []Group{}
	for _, g := range groups {
		res = append(res, Group{
			Id:          g.ID,
			Name:        g.Name,
			Description: toStringPtr(g.Description),
		})
	}

	return ListGroups200JSONResponse(res), nil
}

func (s *Server) CreateGroup(ctx context.Context, request CreateGroupRequestObject) (CreateGroupResponseObject, error) {
	g, err := s.Queries.CreateGroup(ctx, db.CreateGroupParams{
		ID:          uuid.New(),
		Name:        request.Body.Name,
		Description: fromStringPtr(request.Body.Description),
	})
	if err != nil {
		return nil, err
	}

	return CreateGroup201JSONResponse(Group{
		Id:          g.ID,
		Name:        g.Name,
		Description: toStringPtr(g.Description),
	}), nil
}

func (s *Server) GetGroup(ctx context.Context, request GetGroupRequestObject) (GetGroupResponseObject, error) {
	g, err := s.Queries.GetGroup(ctx, request.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetGroup404Response{}, nil
		}
		return nil, err
	}

	return GetGroup200JSONResponse(Group{
		Id:          g.ID,
		Name:        g.Name,
		Description: toStringPtr(g.Description),
	}), nil
}

func (s *Server) UpdateGroup(ctx context.Context, request UpdateGroupRequestObject) (UpdateGroupResponseObject, error) {
	params := db.UpdateGroupParams{
		ID: request.Id,
	}
	if request.Body.Name != nil {
		params.Name = sql.NullString{String: *request.Body.Name, Valid: true}
	}
	if request.Body.Description != nil {
		params.Description = sql.NullString{String: *request.Body.Description, Valid: true}
	}

	g, err := s.Queries.UpdateGroup(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UpdateGroup404Response{}, nil
		}
		return nil, err
	}

	return UpdateGroup200JSONResponse(Group{
		Id:          g.ID,
		Name:        g.Name,
		Description: toStringPtr(g.Description),
	}), nil
}

func (s *Server) DeleteGroup(ctx context.Context, request DeleteGroupRequestObject) (DeleteGroupResponseObject, error) {
	err := s.Queries.DeleteGroup(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	return DeleteGroup204Response{}, nil
}

func (s *Server) ListGroupMembers(ctx context.Context, request ListGroupMembersRequestObject) (ListGroupMembersResponseObject, error) {
	persons, err := s.Queries.ListGroupMembers(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	var res []Person
	for _, p := range persons {
		res = append(res, Person{
			Id:          p.ID,
			Email:       types.Email(p.Email),
			Name:        p.Name,
			HasPassword: p.PasswordHash.Valid,
		})
	}

	return ListGroupMembers200JSONResponse(res), nil
}

func (s *Server) AddGroupMember(ctx context.Context, request AddGroupMemberRequestObject) (AddGroupMemberResponseObject, error) {
	err := s.Queries.AddGroupMember(ctx, db.AddGroupMemberParams{
		ID:        uuid.New(),
		GroupID:   request.Id,
		ContactID: request.Body.PersonId,
	})
	if err != nil {
		return nil, err
	}

	return AddGroupMember204Response{}, nil
}

func (s *Server) RemoveGroupMember(ctx context.Context, request RemoveGroupMemberRequestObject) (RemoveGroupMemberResponseObject, error) {
	err := s.Queries.RemoveGroupMember(ctx, db.RemoveGroupMemberParams{
		GroupID:   request.Id,
		ContactID: request.PersonId,
	})
	if err != nil {
		return nil, err
	}

	return RemoveGroupMember204Response{}, nil
}

// Invite Handlers
func (s *Server) ListInvites(ctx context.Context, request ListInvitesRequestObject) (ListInvitesResponseObject, error) {
	invites, err := s.Queries.ListInvites(ctx)
	if err != nil {
		return nil, err
	}

	res := []Invite{}
	for _, i := range invites {
		res = append(res, Invite{
			Id:           i.ID,
			Title:        i.Title,
			Description:  toStringPtr(i.Description),
			From:         i.From,
			To:           toTimePtr(i.To),
			DurationNs:   toIntPtr(i.Duration),
			CreatedAt:    i.CreatedAt,
			Status:       InviteStatus(i.Status),
			FromPersonId: i.FromPersonID.UUID,
		})
	}

	return ListInvites200JSONResponse(res), nil
}

func (s *Server) CreateInvite(ctx context.Context, request CreateInviteRequestObject) (CreateInviteResponseObject, error) {
	i, err := s.Queries.CreateInvite(ctx, db.CreateInviteParams{
		ID:           uuid.New(),
		Title:        request.Body.Title,
		Description:  fromStringPtr(request.Body.Description),
		From:         request.Body.From,
		To:           fromTimePtr(request.Body.To),
		Duration:     fromIntPtr(request.Body.DurationNs),
		CreatedAt:    time.Now(),
		Status:       "pending",
		FromPersonID: uuid.NullUUID{UUID: request.Body.FromPersonId, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	return CreateInvite201JSONResponse(Invite{
		Id:           i.ID,
		Title:        i.Title,
		Description:  toStringPtr(i.Description),
		From:         i.From,
		To:           toTimePtr(i.To),
		DurationNs:   toIntPtr(i.Duration),
		CreatedAt:    i.CreatedAt,
		Status:       InviteStatus(i.Status),
		FromPersonId: i.FromPersonID.UUID,
	}), nil
}

func (s *Server) GetInvite(ctx context.Context, request GetInviteRequestObject) (GetInviteResponseObject, error) {
	i, err := s.Queries.GetInvite(ctx, request.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetInvite404Response{}, nil
		}
		return nil, err
	}

	return GetInvite200JSONResponse(Invite{
		Id:           i.ID,
		Title:        i.Title,
		Description:  toStringPtr(i.Description),
		From:         i.From,
		To:           toTimePtr(i.To),
		DurationNs:   toIntPtr(i.Duration),
		CreatedAt:    i.CreatedAt,
		Status:       InviteStatus(i.Status),
		FromPersonId: i.FromPersonID.UUID,
	}), nil
}

func (s *Server) UpdateInvite(ctx context.Context, request UpdateInviteRequestObject) (UpdateInviteResponseObject, error) {
	params := db.UpdateInviteParams{
		ID: request.Id,
	}
	if request.Body.Title != nil {
		params.Title = sql.NullString{String: *request.Body.Title, Valid: true}
	}
	if request.Body.Description != nil {
		params.Description = sql.NullString{String: *request.Body.Description, Valid: true}
	}
	if request.Body.From != nil {
		params.From = sql.NullTime{Time: *request.Body.From, Valid: true}
	}
	if request.Body.To != nil {
		params.To = sql.NullTime{Time: *request.Body.To, Valid: true}
	}
	if request.Body.DurationNs != nil {
		params.Duration = sql.NullInt64{Int64: int64(*request.Body.DurationNs), Valid: true}
	}
	if request.Body.Status != nil {
		params.Status = sql.NullString{String: string(*request.Body.Status), Valid: true}
	}
	if request.Body.FromPersonId != nil {
		params.FromPersonID = uuid.NullUUID{UUID: *request.Body.FromPersonId, Valid: true}
	}

	i, err := s.Queries.UpdateInvite(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UpdateInvite404Response{}, nil
		}
		return nil, err
	}

	return UpdateInvite200JSONResponse(Invite{
		Id:           i.ID,
		Title:        i.Title,
		Description:  toStringPtr(i.Description),
		From:         i.From,
		To:           toTimePtr(i.To),
		DurationNs:   toIntPtr(i.Duration),
		CreatedAt:    i.CreatedAt,
		Status:       InviteStatus(i.Status),
		FromPersonId: i.FromPersonID.UUID,
	}), nil
}

func (s *Server) DeleteInvite(ctx context.Context, request DeleteInviteRequestObject) (DeleteInviteResponseObject, error) {
	err := s.Queries.DeleteInvite(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	return DeleteInvite204Response{}, nil
}

// Phase Handlers
func (s *Server) ListInvitePhases(ctx context.Context, request ListInvitePhasesRequestObject) (ListInvitePhasesResponseObject, error) {
	phases, err := s.Queries.ListInvitePhases(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	res := []InvitePhase{}
	for _, p := range phases {
		var cfg map[string]interface{}
		json.Unmarshal(p.StrategyConfig, &cfg)

		res = append(res, InvitePhase{
			Id:             p.ID,
			InviteId:       p.InviteID,
			Order:          int(p.Order),
			StrategyKind:   InvitePhaseStrategyKind(p.StrategyKind),
			StrategyConfig: cfg,
		})
	}

	return ListInvitePhases200JSONResponse(res), nil
}

func (s *Server) CreateInvitePhase(ctx context.Context, request CreateInvitePhaseRequestObject) (CreateInvitePhaseResponseObject, error) {
	cfgBytes, err := json.Marshal(request.Body.StrategyConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid strategy config: %w", err)
	}

	p, err := s.Queries.CreateInvitePhase(ctx, db.CreateInvitePhaseParams{
		ID:             uuid.New(),
		InviteID:       request.Id,
		Order:          int32(request.Body.Order),
		StrategyKind:   string(request.Body.StrategyKind),
		StrategyConfig: cfgBytes,
	})
	if err != nil {
		return nil, err
	}

	var cfgMap map[string]interface{}
	json.Unmarshal(p.StrategyConfig, &cfgMap)

	return CreateInvitePhase201JSONResponse(InvitePhase{
		Id:             p.ID,
		InviteId:       p.InviteID,
		Order:          int(p.Order),
		StrategyKind:   InvitePhaseStrategyKind(p.StrategyKind),
		StrategyConfig: cfgMap,
	}), nil
}

func (s *Server) DeleteInvitePhase(ctx context.Context, request DeleteInvitePhaseRequestObject) (DeleteInvitePhaseResponseObject, error) {
	err := s.Queries.DeleteInvitePhase(ctx, db.DeleteInvitePhaseParams{
		ID:       request.PhaseId,
		InviteID: request.Id,
	})
	if err != nil {
		return nil, err
	}
	return DeleteInvitePhase204Response{}, nil
}

func (s *Server) StartInvite(ctx context.Context, request StartInviteRequestObject) (StartInviteResponseObject, error) {
	if s.StartInviteFunc == nil {
		return nil, errors.New("StartInviteFunc not configured")
	}

	err := s.StartInviteFunc(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	return StartInvite200Response{}, nil
}

func (s *Server) GetInviteStatus(ctx context.Context, request GetInviteStatusRequestObject) (GetInviteStatusResponseObject, error) {
	// 1. Get Overall Invite
	i, err := s.Queries.GetInvite(ctx, request.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetInviteStatus404Response{}, nil
		}
		return nil, err
	}

	resp := InviteStatusReport{
		InviteId:      i.ID,
		OverallStatus: i.Status,
	}

	// 2. Get Active Phase details (if active)
	if i.Status == "active" {
		activeRow, err := s.Queries.GetActivePhaseForInvite(ctx, i.ID)
		if err == nil {
			progressMsg := "Processing"
			if s.GetProgressFunc != nil {
				msg, pErr := s.GetProgressFunc(ctx, activeRow)
				if pErr == nil {
					progressMsg = msg
				}
			}

			resp.ActivePhase = &struct {
				Id              *uuid.UUID "json:\"id,omitempty\""
				NextCheckAt     *time.Time "json:\"next_check_at,omitempty\""
				Order           *int       "json:\"order,omitempty\""
				ProgressMessage *string    "json:\"progress_message,omitempty\""
				StrategyKind    *string    "json:\"strategy_kind,omitempty\""
			}{
				Id:              &activeRow.PhaseID,
				NextCheckAt:     toTimePtr(activeRow.NextCheckAt),
				Order:           toIntPtr(sql.NullInt64{Int64: int64(activeRow.Order), Valid: true}),
				StrategyKind:    &activeRow.StrategyKind,
				ProgressMessage: &progressMsg,
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	// 3. Get All Invitees Status
	inviteeRows, err := s.Queries.GetInviteesStatus(ctx, i.ID)
	if err != nil {
		return nil, err
	}

	resInvitees := []InviteeStatus{}
	for _, ir := range inviteeRows {
		resInvitees = append(resInvitees, InviteeStatus{
			Id:         ir.ID,
			Email:      types.Email(ir.Email),
			Name:       ir.Name,
			InvitedAt:  ir.InvitedAt,
			Status:     InviteeStatusStatus(ir.Status),
			MagicToken: &ir.MagicToken,
		})
	}
	resp.Invitees = &resInvitees

	return GetInviteStatus200JSONResponse(resp), nil
}

func (s *Server) GetInviteForResponse(ctx context.Context, request GetInviteForResponseRequestObject) (GetInviteForResponseResponseObject, error) {
	invitee, err := s.Queries.GetInviteeByToken(ctx, request.Token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetInviteForResponse404Response{}, nil
		}
		return nil, err
	}

	return GetInviteForResponse200JSONResponse(PublicInviteDetails{
		InviteId:     invitee.InviteID,
		Title:        invitee.Title,
		Description:  toStringPtr(invitee.InviteDescription),
		From:         invitee.From,
		To:           toTimePtr(invitee.To),
		CurrentState: PublicInviteDetailsCurrentState(invitee.State),
	}), nil
}

func (s *Server) RespondToInvite(ctx context.Context, request RespondToInviteRequestObject) (RespondToInviteResponseObject, error) {
	state := "pending"
	switch request.Body.Action {
	case Accept:
		state = "accepted"
	case Decline:
		state = "declined"
	}

	err := s.Queries.RespondToInvite(ctx, db.RespondToInviteParams{
		MagicToken: request.Token,
		State:      state,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RespondToInvite404Response{}, nil
		}
		return nil, err
	}

	return RespondToInvite204Response{}, nil
}
