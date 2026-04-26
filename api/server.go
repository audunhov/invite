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
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/oapi-codegen/runtime/types"
	"invite/db"
	"invite/email"
	"invite/internal/auth"
	"invite/internal/limiter"
	"invite/models"
)

type Server struct {
	Queries                  *db.Queries
	StartInviteFunc          func(ctx context.Context, inviteID uuid.UUID) error
	GetProgressFunc          func(ctx context.Context, row db.GetActivePhaseForInviteRow) (string, error)
	HandleInviteeResponseFunc func(ctx context.Context, token uuid.UUID, state string) error
	InvalidateInviteFunc      func(ctx context.Context, inviteID uuid.UUID) error
	InvalidatePhaseFunc       func(ctx context.Context, inviteID uuid.UUID, phaseID uuid.UUID) error
	GetDashboardStatsFunc    func(ctx context.Context) (*models.DashboardStats, error)
	Limiter                   *limiter.IPRateLimiter
	EmailService              *email.Service
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
		if err := s.EmailService.SendResetPasswordEmail(context.Background(), s.Queries, p.Email, token); err != nil {
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

func (s *Server) RetryEmail(ctx context.Context, request RetryEmailRequestObject) (RetryEmailResponseObject, error) {
	log, err := s.Queries.GetEmailLog(ctx, request.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RetryEmail404Response{}, nil
		}
		return nil, err
	}

	// Only allow retrying if it failed
	if log.Status == "sent" {
		return RetryEmail400Response{}, nil
	}

	err = s.EmailService.SendRaw(log.RecipientEmail, log.Subject, log.Body)

	status := "sent"
	var errMsg sql.NullString
	if err != nil {
		status = "failed"
		errMsg = sql.NullString{String: err.Error(), Valid: true}
	}

	updateErr := s.Queries.UpdateEmailLogStatus(ctx, db.UpdateEmailLogStatusParams{
		ID:           log.ID,
		Status:       status,
		ErrorMessage: errMsg,
	})
	if updateErr != nil {
		return nil, fmt.Errorf("failed to update email log status: %w", updateErr)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	return RetryEmail204Response{}, nil
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

func toFloat32Ptr(f float64) *float32 {
	val := float32(f)
	return &val
}

// Dashboard Handlers
func (s *Server) GetDashboardStats(ctx context.Context, request GetDashboardStatsRequestObject) (GetDashboardStatsResponseObject, error) {
	if s.GetDashboardStatsFunc == nil {
		return nil, errors.New("GetDashboardStatsFunc not initialized")
	}

	stats, err := s.GetDashboardStatsFunc(ctx)
	if err != nil {
		return nil, err
	}

	resp := DashboardStats{}
	
	// Map Stats
	resp.Stats.ActiveInvites = &stats.Stats.ActiveInvites
	resp.Stats.FailedEmails = &stats.Stats.FailedEmails
	resp.Stats.SuccessRate = toFloat32Ptr(stats.Stats.SuccessRate)

	// Map Timeline
	timeline := make([]struct {
		Id     openapi_types.UUID `json:"id"`
		Phases []struct {
			AcceptedCount int    `json:"accepted_count"`
			DeclinedCount int    `json:"declined_count"`
			Order         int    `json:"order"`
			Status        string `json:"status"`
			TotalInvitees int    `json:"total_invitees"`
		} `json:"phases"`
		Status string `json:"status"`
		Title  string `json:"title"`
	}, len(stats.Timeline))

	for i, t := range stats.Timeline {
		timeline[i].Id = t.ID
		timeline[i].Title = t.Title
		timeline[i].Status = t.Status
		timeline[i].Phases = make([]struct {
			AcceptedCount int    `json:"accepted_count"`
			DeclinedCount int    `json:"declined_count"`
			Order         int    `json:"order"`
			Status        string `json:"status"`
			TotalInvitees int    `json:"total_invitees"`
		}, len(t.Phases))
		for j, p := range t.Phases {
			timeline[i].Phases[j].Order = p.Order
			timeline[i].Phases[j].Status = p.Status
			timeline[i].Phases[j].AcceptedCount = p.AcceptedCount
			timeline[i].Phases[j].DeclinedCount = p.DeclinedCount
			timeline[i].Phases[j].TotalInvitees = p.TotalInvitees
		}
	}
	resp.Timeline = &timeline

	// Map Bottlenecks
	resp.Bottlenecks = make([]struct {
		ActiveSince  *time.Time          "json:\"active_since,omitempty\""
		InviteId     *openapi_types.UUID "json:\"invite_id,omitempty\""
		PhaseOrder   *int                "json:\"phase_order,omitempty\""
		StrategyKind *string             "json:\"strategy_kind,omitempty\""
		Tags         *[]Tag              "json:\"tags,omitempty\""
		Title        *string             "json:\"title,omitempty\""
		WaitingFor   *string             "json:\"waiting_for,omitempty\""
	}, len(stats.Bottlenecks))

	for i, b := range stats.Bottlenecks {
		b := b
		resp.Bottlenecks[i].InviteId = &b.InviteID
		resp.Bottlenecks[i].Title = &b.Title
		resp.Bottlenecks[i].PhaseOrder = &b.PhaseOrder
		resp.Bottlenecks[i].StrategyKind = &b.StrategyKind
		resp.Bottlenecks[i].WaitingFor = &b.WaitingFor
		resp.Bottlenecks[i].ActiveSince = &b.ActiveSince

		// Map Tags
		tags := make([]Tag, len(b.Tags))
		for j, t := range b.Tags {
			tags[j] = Tag{
				Id:    t.ID,
				Name:  t.Name,
				Color: t.Color,
			}
		}
		resp.Bottlenecks[i].Tags = &tags
	}

	// Map Activity
	resp.Activity = make([]struct {
		Message   *string    "json:\"message,omitempty\""
		Timestamp *time.Time "json:\"timestamp,omitempty\""
		Type      *string    "json:\"type,omitempty\""
	}, len(stats.Activity))

	for i, a := range stats.Activity {
		a := a
		resp.Activity[i].Timestamp = &a.Timestamp
		resp.Activity[i].Type = &a.Type
		resp.Activity[i].Message = &a.Message
	}

	return GetDashboardStats200JSONResponse(resp), nil
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

	res := []Person{}
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

// Tag Handlers
func (s *Server) ListTags(ctx context.Context, request ListTagsRequestObject) (ListTagsResponseObject, error) {
	tags, err := s.Queries.ListTags(ctx)
	if err != nil {
		return nil, err
	}

	res := []Tag{}
	for _, t := range tags {
		res = append(res, Tag{
			Id:    t.ID,
			Name:  t.Name,
			Color: t.Color,
		})
	}

	return ListTags200JSONResponse(res), nil
}

func (s *Server) CreateTag(ctx context.Context, request CreateTagRequestObject) (CreateTagResponseObject, error) {
	t, err := s.Queries.CreateTag(ctx, db.CreateTagParams{
		ID:    uuid.New(),
		Name:  request.Body.Name,
		Color: request.Body.Color,
	})
	if err != nil {
		return nil, err
	}

	return CreateTag201JSONResponse(Tag{
		Id:    t.ID,
		Name:  t.Name,
		Color: t.Color,
	}), nil
}

func (s *Server) GetTagUsage(ctx context.Context, request GetTagUsageRequestObject) (GetTagUsageResponseObject, error) {
	_, err := s.Queries.GetTag(ctx, request.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GetTagUsage404Response{}, nil
		}
		return nil, err
	}

	count, err := s.Queries.GetTagUsageCount(ctx, request.Id)
	if err != nil {
		return nil, err
	}

	c := int(count)
	return GetTagUsage200JSONResponse{Count: &c}, nil
}

func (s *Server) UpdateTag(ctx context.Context, request UpdateTagRequestObject) (UpdateTagResponseObject, error) {
	params := db.UpdateTagParams{
		ID: request.Id,
	}
	
	// Get current tag to handle partial updates
	current, err := s.Queries.GetTag(ctx, request.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UpdateTag404Response{}, nil
		}
		return nil, err
	}

	params.Name = current.Name
	if request.Body.Name != nil {
		params.Name = *request.Body.Name
	}
	
	params.Color = current.Color
	if request.Body.Color != nil {
		params.Color = *request.Body.Color
	}

	t, err := s.Queries.UpdateTag(ctx, params)
	if err != nil {
		return nil, err
	}

	return UpdateTag200JSONResponse(Tag{
		Id:    t.ID,
		Name:  t.Name,
		Color: t.Color,
	}), nil
}

func (s *Server) DeleteTag(ctx context.Context, request DeleteTagRequestObject) (DeleteTagResponseObject, error) {
	_, err := s.Queries.GetTag(ctx, request.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeleteTag404Response{}, nil
		}
		return nil, err
	}

	err = s.Queries.DeleteTag(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	return DeleteTag204Response{}, nil
}

// Invite Handlers
func (s *Server) ListInvites(ctx context.Context, request ListInvitesRequestObject) (ListInvitesResponseObject, error) {
	invites, err := s.Queries.ListInvites(ctx)
	if err != nil {
		return nil, err
	}

	res := []Invite{}
	for _, i := range invites {
		tags, _ := s.Queries.GetTagsByInvite(ctx, i.ID)
		resTags := []Tag{}
		for _, t := range tags {
			resTags = append(resTags, Tag{
				Id:    t.ID,
				Name:  t.Name,
				Color: t.Color,
			})
		}

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
			Tags:         &resTags,
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

	// Handle tags
	if request.Body.TagIds != nil && len(*request.Body.TagIds) > 0 {
		var tagUUIDs []uuid.UUID
		for _, idStr := range *request.Body.TagIds {
			u, err := uuid.Parse(idStr.String())
			if err == nil {
				tagUUIDs = append(tagUUIDs, u)
			}
		}
		if len(tagUUIDs) > 0 {
			err = s.Queries.AddInviteTags(ctx, db.AddInviteTagsParams{
				InviteID: i.ID,
				Column2:  tagUUIDs,
			})
			if err != nil {
				return nil, err
			}
		}
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

	tags, _ := s.Queries.GetTagsByInvite(ctx, i.ID)
	resTags := []Tag{}
	for _, t := range tags {
		resTags = append(resTags, Tag{
			Id:    t.ID,
			Name:  t.Name,
			Color: t.Color,
		})
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
		Tags:         &resTags,
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

	// Handle tags
	if request.Body.TagIds != nil {
		err = s.Queries.ClearInviteTags(ctx, i.ID)
		if err != nil {
			return nil, err
		}

		if len(*request.Body.TagIds) > 0 {
			var tagUUIDs []uuid.UUID
			for _, idStr := range *request.Body.TagIds {
				u, err := uuid.Parse(idStr.String())
				if err == nil {
					tagUUIDs = append(tagUUIDs, u)
				}
			}
			if len(tagUUIDs) > 0 {
				err = s.Queries.AddInviteTags(ctx, db.AddInviteTagsParams{
					InviteID: i.ID,
					Column2:  tagUUIDs,
				})
				if err != nil {
					return nil, err
				}
			}
		}
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
	if s.InvalidateInviteFunc != nil {
		if err := s.InvalidateInviteFunc(ctx, request.Id); err != nil {
			return nil, err
		}
	}

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
	if s.InvalidatePhaseFunc != nil {
		if err := s.InvalidatePhaseFunc(ctx, request.Id, request.PhaseId); err != nil {
			return nil, err
		}
	} else {
		err := s.Queries.DeleteInvitePhase(ctx, db.DeleteInvitePhaseParams{
			ID:       request.PhaseId,
			InviteID: request.Id,
		})
		if err != nil {
			return nil, err
		}
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

	tags, _ := s.Queries.GetTagsByInvite(ctx, i.ID)
	resTags := []Tag{}
	for _, t := range tags {
		resTags = append(resTags, Tag{
			Id:    t.ID,
			Name:  t.Name,
			Color: t.Color,
		})
	}

	resp := InviteStatusReport{
		InviteId:      i.ID,
		OverallStatus: i.Status,
		Tags:          &resTags,
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
		is := InviteeStatus{
			Id:         ir.ID,
			Email:      types.Email(ir.Email),
			Name:       ir.Name,
			InvitedAt:  ir.InvitedAt,
			Status:     InviteeStatusStatus(ir.Status),
			MagicToken: &ir.MagicToken,
		}

		if ir.PhaseOrder.Valid {
			order := int(ir.PhaseOrder.Int32)
			is.PhaseOrder = &order
		}

		// Fetch latest email log for this invitee
		emailLog, err := s.Queries.GetEmailLogByInvitee(ctx, uuid.NullUUID{UUID: ir.ID, Valid: true})
		if err == nil {
			is.EmailId = &emailLog.ID
			is.EmailStatus = &emailLog.Status
			is.EmailError = toStringPtr(emailLog.ErrorMessage)
			attempts := int(emailLog.Attempts)
			is.EmailAttempts = &attempts
		}

		resInvitees = append(resInvitees, is)
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

	if s.HandleInviteeResponseFunc != nil {
		err := s.HandleInviteeResponseFunc(ctx, request.Token, state)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return RespondToInvite404Response{}, nil
			}
			return nil, err
		}
	} else {
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
	}

	return RespondToInvite204Response{}, nil
}
