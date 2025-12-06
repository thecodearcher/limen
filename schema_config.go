package aegis

type SchemaConfig struct {
	// A function to return a map of global fields to be added to all schemas when creating a record. e.g:
	//  func(ctx context.Context) map[string]any {
	// 		return map[string]any{
	//  		"uuid": uuid.New().String(),
	//  		"created_at": time.Now(),
	//  		"updated_at": time.Now(),
	// 		 }
	//	 }
	// this function will be called during the creation of any schema record.
	// You can also set fields on supported schemas itself.
	AdditionalFields AdditionalFieldsFunc
	// User schema configuration
	User *UserSchema
	// Verification schema configuration
	Verification *VerificationSchema
	// Session schema configuration
	Session *SessionSchema
	// Rate limit schema configuration
	RateLimit *RateLimitSchema
}

type SchemaConfigOption func(*SchemaConfig)

// NewDefaultSchemaConfig creates a new SchemaConfig with default values.
func NewDefaultSchemaConfig(opts ...SchemaConfigOption) *SchemaConfig {
	config := &SchemaConfig{
		User:         NewDefaultUserSchema(),
		Verification: NewDefaultVerificationSchema(),
		Session:      NewDefaultSessionSchema(),
		RateLimit:    NewDefaultRateLimitSchema(),
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// WithSchemaAdditionalFields sets the global additional fields function
func WithSchemaAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.AdditionalFields = fn
	}
}

// WithSchemaUser sets the user schema configuration
func WithSchemaUser(opts ...UserSchemaOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.User = NewDefaultUserSchema(opts...)
	}
}

// WithSchemaVerification sets the verification schema configuration
func WithSchemaVerification(opts ...VerificationSchemaOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.Verification = NewDefaultVerificationSchema(opts...)
	}
}

// WithSchemaSession sets the session schema configuration
func WithSchemaSession(opts ...SessionSchemaOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.Session = NewDefaultSessionSchema(opts...)
	}
}

// WithSchemaRateLimit sets the rate limit schema configuration
func WithSchemaRateLimit(opts ...RateLimitSchemaOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.RateLimit = NewDefaultRateLimitSchema(opts...)
	}
}
