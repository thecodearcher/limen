package admin

type config struct {
	bansEnabled          bool
	impersonationEnabled bool
}

type ConfigOption func(*config)

// WithBans enables banning users. When disabled the ban columns are not added to the users table.
func WithBans(enabled bool) ConfigOption {
	return func(c *config) {
		c.bansEnabled = enabled
	}
}

// WithImpersonation enables impersonating users. When disabled the impersonated_by column
// is not added to the sessions table.
func WithImpersonation(enabled bool) ConfigOption {
	return func(c *config) {
		c.impersonationEnabled = enabled
	}
}
