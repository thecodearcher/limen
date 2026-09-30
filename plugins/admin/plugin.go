package admin

import "github.com/thecodearcher/limen"

type adminPlugin struct {
	config *config

	core *limen.LimenCore

	userRoleSchema *userRoleSchema
	userSchema     *userSchema
}

func New(opts ...ConfigOption) *adminPlugin {
	config := &config{
		bansEnabled:          true,
		impersonationEnabled: true,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &adminPlugin{config: config}
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
	return nil
}

func (p *adminPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/admin",
	}
}

func (p *adminPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
}
