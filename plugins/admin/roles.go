package admin

import (
	"context"
	"fmt"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (p *adminPlugin) GetUserRoles(ctx context.Context, user *limen.User) ([]access.Role, error) {
	models, err := p.core.FindMany(ctx, p.userRoleSchema, []limen.Where{
		limen.Eq(p.userRoleSchema.GetUserIDField(), user.ID),
	})
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(models))
	for _, userRole := range limen.MapToSliceOfType[*UserRole](models) {
		names = append(names, userRole.Role)
	}
	if len(names) == 0 {
		names = append(names, p.config.defaultRoles...)
	}
	if p.isConfiguredAdmin(user) {
		names = append(names, p.config.adminRole)
	}

	roles := make([]access.Role, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		role := p.findRole(name)
		if role == nil || seen[role.Name()] {
			continue
		}
		seen[role.Name()] = true
		roles = append(roles, *role)
	}
	return roles, nil
}

func (p *adminPlugin) GetPermissions(ctx context.Context, user *limen.User) (access.Permissions, error) {
	roles, err := p.GetUserRoles(ctx, user)
	if err != nil {
		return nil, err
	}

	permissions := make(access.Permissions)
	for _, role := range roles {
		permissions = access.MergePermissions(permissions, role.Permissions())
	}
	return permissions, nil
}

func (p *adminPlugin) HasPermission(ctx context.Context, user *limen.User, permissions access.Permissions) error {
	granted, err := p.GetPermissions(ctx, user)
	if err != nil {
		return err
	}
	if !access.HasRequiredPermissions(granted, permissions) {
		return ErrInsufficientPermission
	}
	return nil
}

func (p *adminPlugin) isConfiguredAdmin(user *limen.User) bool {
	if p.adminUserIDs[fmt.Sprint(user.ID)] {
		return true
	}
	publicID, ok := p.core.EncodePublicID(p.core.Schema.User, user)
	return ok && p.adminUserIDs[publicID]
}

func (p *adminPlugin) ListRoles(ctx context.Context, actor *limen.User) (map[string]access.Permissions, error) {
	if err := p.HasPermission(ctx, actor, access.P("user:set-role")); err != nil {
		return nil, err
	}

	vocabulary := access.Permissions(p.config.accessControl.Statements())
	roles := make(map[string]access.Permissions, len(p.config.roles))
	for _, role := range p.config.roles {
		roles[role.Name()] = access.ClampPermissions(vocabulary, role.Permissions())
	}
	return roles, nil
}
