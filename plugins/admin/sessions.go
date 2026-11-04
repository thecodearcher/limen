package admin

import (
	"context"
	"errors"
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

	impersonatedByColumn := p.core.Schema.Session.GetField(SessionSchemaImpersonatedByField)
	impersonators, err := p.findImpersonators(ctx, sessions, impersonatedByColumn)
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
			ImpersonatedBy: impersonators[session.Raw()[impersonatedByColumn]],
		})
	}
	return result, nil
}

func (p *adminPlugin) findImpersonators(ctx context.Context, sessions []limen.Session, impersonatedByColumn string) (map[any]*limen.User, error) {
	var ids []any
	for _, session := range sessions {
		if id := session.Raw()[impersonatedByColumn]; id != nil && !slices.Contains(ids, id) {
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
