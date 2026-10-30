package admin

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

func (p *adminPlugin) GetUserRoles(ctx context.Context, user *limen.User) ([]access.Role, error) {
	names, err := p.assignedRoleNames(ctx, user.ID)
	if err != nil {
		return nil, err
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

	return access.MergeRolePermissions(roles...), nil
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

	roles := make(map[string]access.Permissions, len(p.config.roles))
	for _, role := range p.config.roles {
		roles[role.Name()] = role.Permissions()
	}
	return roles, nil
}

func (p *adminPlugin) SetRoles(ctx context.Context, actor *limen.User, userID any, roles []string) (*UserWithRoles, error) {
	if err := p.HasPermission(ctx, actor, access.P("user:set-role")); err != nil {
		return nil, err
	}

	user, err := p.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	wanted := make([]string, 0, len(roles))
	for _, name := range roles {
		role := p.findRole(strings.ToLower(strings.TrimSpace(name)))
		if role == nil {
			return nil, newUnknownRoleError(name)
		}
		if !slices.Contains(wanted, role.Name()) {
			wanted = append(wanted, role.Name())
		}
	}

	current, err := p.assignedRoleNames(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	toRemove := slices.DeleteFunc(slices.Clone(current), func(name string) bool { return slices.Contains(wanted, name) })
	toAdd := slices.DeleteFunc(wanted, func(name string) bool { return slices.Contains(current, name) })
	if len(toAdd) == 0 && len(toRemove) == 0 {
		return p.withRoles(ctx, user)
	}

	if err := p.requireGrantable(ctx, actor, slices.Concat(toAdd, toRemove)); err != nil {
		return nil, err
	}

	if slices.Contains(toRemove, p.config.adminRole) {
		if err := p.requireNotLastAdmin(ctx, user.ID); err != nil {
			return nil, err
		}
	}

	err = p.core.WithTransaction(ctx, func(ctx context.Context) error {
		if len(toRemove) > 0 {
			if err := p.core.Delete(ctx, p.userRoleSchema, []limen.Where{
				limen.Eq(p.userRoleSchema.GetUserIDField(), user.ID),
				limen.In(p.userRoleSchema.GetRoleField(), toRemove),
			}); err != nil {
				return err
			}
		}
		for _, name := range toAdd {
			if err := p.core.Create(ctx, p.userRoleSchema, &UserRole{UserID: user.ID, Role: name}, nil); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p.withRoles(ctx, user)
}

func (p *adminPlugin) requireGrantable(ctx context.Context, actor *limen.User, roleNames []string) error {
	granted, err := p.GetPermissions(ctx, actor)
	if err != nil {
		return err
	}

	for _, name := range roleNames {
		role := p.findRole(name)
		if role == nil {
			continue
		}
		if !access.HasRequiredPermissions(granted, role.Permissions()) {
			return ErrRoleNotGrantable
		}
	}
	return nil
}

func (p *adminPlugin) assignedRoleNames(ctx context.Context, userID any) ([]string, error) {
	models, err := p.core.FindMany(ctx, p.userRoleSchema, []limen.Where{
		limen.Eq(p.userRoleSchema.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(models))
	for _, userRole := range limen.MapToSliceOfType[*UserRole](models) {
		names = append(names, userRole.Role)
	}
	return names, nil
}
