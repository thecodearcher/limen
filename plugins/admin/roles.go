package admin

import (
	"context"
	"fmt"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (p *adminPlugin) GetRoles(ctx context.Context, user *limen.User) ([]access.Role, error) {
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
	if p.adminUserIDs[fmt.Sprint(user.ID)] {
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
	roles, err := p.GetRoles(ctx, user)
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
