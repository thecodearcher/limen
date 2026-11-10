package admin

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (p *adminPlugin) ListUserSessions(ctx context.Context, actor *limen.User, userID any) ([]UserSession, error) {
	if err := p.HasPermission(ctx, actor, access.P("session:list")); err != nil {
		return nil, err
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessions, err := p.core.SessionManager.ListSessions(ctx, user.ID)
	if err != nil && !errors.Is(err, limen.ErrRecordNotFound) {
		return nil, err
	}

	if len(sessions) == 0 {
		return nil, nil
	}

	impersonators, err := p.findImpersonators(ctx, sessions)
	if err != nil {
		return nil, err
	}

	result := make([]UserSession, 0, len(sessions))
	for _, session := range sessions {
		ipAddress, _ := session.Metadata["ip_address"].(string)
		userAgent, _ := session.Metadata["user_agent"].(string)
		result = append(result, UserSession{
			ID:             session.Token,
			CreatedAt:      session.CreatedAt,
			ExpiresAt:      session.ExpiresAt,
			LastAccess:     session.LastAccess,
			IPAddress:      ipAddress,
			UserAgent:      userAgent,
			ImpersonatedBy: impersonators[p.impersonatorID(&session)],
		})
	}
	return result, nil
}

func (p *adminPlugin) impersonatorID(session *limen.Session) any {
	column := p.core.Schema.Session.GetField(SessionSchemaImpersonatedByField)
	return p.core.Schema.NormalizeIDValue(session.Raw()[column])
}

func (p *adminPlugin) findImpersonators(ctx context.Context, sessions []limen.Session) (map[any]*limen.User, error) {
	var ids []any
	for _, session := range sessions {
		if id := p.impersonatorID(&session); id != nil && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	users, err := p.core.FindMany(ctx, p.core.Schema.User, []limen.Where{
		limen.In(p.core.Schema.User.GetIDField(), ids),
	})
	if err != nil {
		return nil, err
	}

	usersByID := make(map[any]*limen.User, len(users))
	for _, user := range limen.MapToSliceOfType[*limen.User](users) {
		usersByID[user.ID] = user
		if encoded, ok := p.core.EncodePublicID(p.core.Schema.User, user); ok {
			usersByID[encoded] = user
		}
	}
	return usersByID, nil
}

func (p *adminPlugin) serializeUserSessions(sessions []UserSession) []userSessionResponse {
	serialized := make([]userSessionResponse, 0, len(sessions))
	for _, session := range sessions {
		response := userSessionResponse{UserSession: session}
		if session.ImpersonatedBy != nil {
			response.ImpersonatedBy = p.serializeUser(session.ImpersonatedBy)
		}
		serialized = append(serialized, response)
	}
	return serialized
}

func (p *adminPlugin) RevokeSession(ctx context.Context, actor *limen.User, token string) error {
	if err := p.HasPermission(ctx, actor, access.P("session:revoke")); err != nil {
		return err
	}
	return p.core.SessionManager.RevokeSession(ctx, token)
}

func (p *adminPlugin) RevokeUserSessions(ctx context.Context, actor *limen.User, userID any) error {
	if err := p.HasPermission(ctx, actor, access.P("session:revoke")); err != nil {
		return err
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	return p.core.SessionManager.RevokeAllSessions(ctx, user.ID)
}

func (p *adminPlugin) ImpersonateUser(ctx context.Context, r *http.Request, session *limen.ValidatedSession, userID any) (*limen.User, *limen.SessionResult, error) {
	actor := session.User
	if err := p.HasPermission(ctx, actor, access.P("user:impersonate")); err != nil {
		return nil, nil, err
	}
	if !p.config.impersonationEnabled {
		return nil, nil, ErrImpersonationNotEnabled
	}

	impersonatorID, err := p.core.SessionManager.GetSessionData(ctx, session.Session, SessionSchemaImpersonatedByField)
	if err != nil {
		return nil, nil, err
	}
	if impersonatorID != nil {
		return nil, nil, ErrAlreadyImpersonating
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if user.ID == actor.ID {
		return nil, nil, ErrCannotImpersonateSelf
	}

	if err := p.requireAdminTargetPermission(ctx, actor, user, "user:impersonate-admins"); err != nil {
		return nil, nil, err
	}

	impersonatedBy := limen.SessionValue{Internal: actor.ID, Client: actor.ID}
	if encoded, ok := p.core.EncodePublicID(p.core.Schema.User, actor); ok {
		impersonatedBy.Client = encoded
	}

	sessionResult, err := p.core.CreateSession(ctx, r, nil, &limen.AuthenticationResult{User: user},
		limen.WithSessionExpiresIn(p.config.impersonationDuration),
		limen.WithSessionData(map[limen.SchemaField]any{SessionSchemaImpersonatedByField: impersonatedBy}),
	)
	if err != nil {
		return nil, nil, err
	}
	return user, sessionResult, nil
}

func (p *adminPlugin) StopImpersonating(ctx context.Context, session *limen.ValidatedSession) (*limen.User, error) {
	if !p.config.impersonationEnabled {
		return nil, ErrImpersonationNotEnabled
	}

	impersonatorID, err := p.core.SessionManager.GetSessionData(ctx, session.Session, SessionSchemaImpersonatedByField)
	if err != nil {
		return nil, err
	}
	if impersonatorID == nil {
		return nil, ErrNotImpersonating
	}

	if err := p.core.SessionManager.RevokeSession(ctx, session.Session.Token); err != nil {
		return nil, err
	}
	return p.core.DBAction.FindUserByID(ctx, p.core.Schema.NormalizeIDValue(impersonatorID))
}
