package admin

import (
	"errors"
	"fmt"
	"slices"

	"github.com/thecodearcher/limen"
	"github.com/thecodearcher/limen/access"
)

type adminPlugin struct {
	config *config

	core *limen.LimenCore

	userRoleSchema *userRoleSchema
	userSchema     *userSchema

	adminUserIDs map[string]bool
}

func New(opts ...ConfigOption) *adminPlugin {
	config := &config{
		accessControl:        access.New(defaultStatements),
		defaultRoles:         []string{roleNameUser},
		adminRole:            roleNameAdmin,
		bansEnabled:          true,
		impersonationEnabled: true,
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

	return nil
}

func (p *adminPlugin) validateRoles() error {
	seen := make(map[string]bool, len(p.config.roles))
	for _, role := range p.config.roles {
		if seen[role.Name()] {
			return fmt.Errorf("admin: role %q is defined multiple times", role.Name())
		}
		seen[role.Name()] = true

		if _, err := p.config.accessControl.NewRole(role.Name(), role.Permissions()); err != nil {
			return fmt.Errorf("admin: role %q is not valid against the configured access control: %w", role.Name(), err)
		}
	}
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
