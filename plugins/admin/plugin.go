package admin

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type adminPlugin struct {
	config *config

	core *limen.LimenCore

	userRoleSchema *userRoleSchema
	userSchema     *userSchema

	adminUserIDs map[string]bool
	userFilters  map[string]UserFilterFunc
}

func New(opts ...ConfigOption) *adminPlugin {
	config := &config{
		accessControl:          access.New(defaultStatements),
		defaultRoles:           []string{roleNameUser},
		adminRole:              roleNameAdmin,
		bansEnabled:            true,
		impersonationEnabled:   true,
		impersonationDuration:  time.Hour,
		adminSessionCookieName: "limen_admin_session",
	}

	for _, opt := range opts {
		opt(config)
	}

	adminUserIDs := make(map[string]bool, len(config.adminUserIDs))
	for _, id := range config.adminUserIDs {
		adminUserIDs[fmt.Sprint(id)] = true
	}

	return &adminPlugin{config: config, adminUserIDs: adminUserIDs}
}

func (p *adminPlugin) Name() limen.PluginName {
	return limen.PluginAdmin
}

func (p *adminPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	p.userRoleSchema = newUserRoleSchema()
	p.userSchema = &userSchema{UserSchema: schema.User}

	schemas := []limen.SchemaIntrospector{
		buildUserRoleTableDef(schema, p.userRoleSchema),
	}

	if p.config.bansEnabled {
		schemas = append(schemas, buildUserBanExtension(p.userSchema))
	}
	if p.config.impersonationEnabled {
		schemas = append(schemas, buildSessionImpersonationExtension(schema))
	}
	return schemas
}

func (p *adminPlugin) Initialize(core *limen.LimenCore) error {
	p.core = core

	if p.config.accessControl == nil {
		return errors.New("admin: access control must be provided")
	}

	if len(p.config.roles) == 0 {
		roles, err := DefaultRoles(p.config.accessControl)
		if err != nil {
			return err
		}
		p.config.roles = roles
	}

	if err := p.validateRoles(); err != nil {
		return err
	}

	for _, name := range p.config.defaultRoles {
		if p.findRole(name) == nil {
			return fmt.Errorf("admin: default role %q is not among the configured roles", name)
		}
	}

	if p.findRole(p.config.adminRole) == nil {
		return fmt.Errorf("admin: admin role %q is not among the configured roles", p.config.adminRole)
	}

	p.userFilters = p.builtinUserFilters()
	maps.Copy(p.userFilters, p.config.userFilters)

	return nil
}

func (p *adminPlugin) BeforeSessionCreate(ctx context.Context, auth *limen.AuthenticationResult) error {
	if !p.config.bansEnabled {
		return nil
	}

	user, err := p.core.DBAction.FindUserByID(ctx, auth.User.ID)
	if err != nil {
		return err
	}
	if !p.isBanned(user, time.Now()) {
		return nil
	}
	if p.config.bannedMessage == nil {
		return ErrUserBanned
	}
	return limen.NewLimenErrorWithCode(ErrUserBanned.Code(), p.config.bannedMessage(user), http.StatusForbidden, nil)
}

func (p *adminPlugin) validateRoles() error {
	seen := make(map[string]bool, len(p.config.roles))
	expanded := make([]access.Role, 0, len(p.config.roles))
	for _, role := range p.config.roles {
		if seen[role.Name()] {
			return fmt.Errorf("admin: role %q is defined multiple times", role.Name())
		}
		seen[role.Name()] = true

		expandedRole, err := p.config.accessControl.NewRole(role.Name(), p.config.accessControl.ExpandPermissions(role.Permissions()))
		if err != nil {
			return fmt.Errorf("admin: role %q is not valid against the configured access control: %w", role.Name(), err)
		}
		expanded = append(expanded, expandedRole)
	}
	p.config.roles = expanded
	return nil
}

func (p *adminPlugin) findRole(name string) *access.Role {
	i := slices.IndexFunc(p.config.roles, func(role access.Role) bool {
		return role.Name() == name
	})
	if i == -1 {
		return nil
	}
	return &p.config.roles[i]
}
