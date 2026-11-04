package admin

import (
	"context"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type passwordHasher interface {
	ValidatePassword(password string) error
	HashPassword(password string) (string, error)
}

func (p *adminPlugin) passwordHasher() (passwordHasher, error) {
	plugin, ok := p.core.GetPlugin(limen.PluginCredentialPassword)
	if !ok {
		return nil, ErrPasswordsNotEnabled
	}
	hasher, ok := plugin.(passwordHasher)
	if !ok {
		return nil, ErrPasswordsNotEnabled
	}
	return hasher, nil
}

func (p *adminPlugin) SetUserPassword(ctx context.Context, actor *limen.User, userID any, req *SetUserPasswordRequest) error {
	if err := p.HasPermission(ctx, actor, access.P("user:set-password")); err != nil {
		return err
	}

	hasher, err := p.passwordHasher()
	if err != nil {
		return err
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := hasher.ValidatePassword(req.Password); err != nil {
		return err
	}
	hashedPassword, err := hasher.HashPassword(req.Password)
	if err != nil {
		return err
	}

	userSchema := p.core.Schema.User
	if err := p.core.Update(ctx, userSchema, map[limen.SchemaField]any{
		limen.UserSchemaPasswordField: hashedPassword,
	}, []limen.Where{limen.Eq(userSchema.GetIDField(), user.ID)}); err != nil {
		return err
	}
	if !req.RevokeSessions {
		return nil
	}
	return p.core.SessionManager.RevokeAllSessions(ctx, user.ID)
}
