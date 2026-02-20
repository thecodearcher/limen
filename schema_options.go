package aegis

// --- Schema config (top-level) ---

// WithSchemaAdditionalFields sets the global additional fields function
func WithSchemaAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.AdditionalFields = fn
	}
}

// WithSchemaIDGenerator sets the global ID generator
func WithSchemaIDGenerator(generator IDGenerator) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.IDGenerator = generator
	}
}

// WithSchemaAccount sets the account schema configuration
func WithSchemaAccount(opts ...SchemaConfigAccountOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.Account = newDefaultAccountSchema(c, opts...)
	}
}

// WithSchemaUser sets the user schema configuration
func WithSchemaUser(opts ...SchemaConfigUserOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.User = newDefaultUserSchema(c, opts...)
	}
}

// WithSchemaVerification sets the verification schema configuration
func WithSchemaVerification(opts ...SchemaConfigVerificationOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.Verification = newDefaultVerificationSchema(c, opts...)
	}
}

// WithSchemaSession sets the session schema configuration
func WithSchemaSession(opts ...SchemaConfigSessionOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.Session = newDefaultSessionSchema(c, opts...)
	}
}

// WithSchemaRateLimit sets the rate limit schema configuration
func WithSchemaRateLimit(opts ...SchemaConfigRateLimitOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		c.RateLimit = newDefaultRateLimitSchema(c, opts...)
	}
}

// WithPluginTableName sets the table name for a plugin schema
func WithPluginTableName(tableName SchemaTableName) PluginSchemaConfigOption {
	return func(c *PluginSchemaConfig) {
		c.TableName = tableName
	}
}

// WithPluginFieldName sets a field name mapping for a plugin schema
func WithPluginFieldName(logicalField SchemaField, columnName string) PluginSchemaConfigOption {
	return func(c *PluginSchemaConfig) {
		if c.Fields == nil {
			c.Fields = make(map[SchemaField]string)
		}
		c.Fields[logicalField] = columnName
	}
}

// WithPluginSchema sets the configuration for a plugin schema
func WithPluginSchema(featureName FeatureName, schemaName SchemaName, opts ...PluginSchemaConfigOption) SchemaConfigOption {
	return func(c *SchemaConfig) {
		if c.pluginSchemas[featureName] == nil {
			c.pluginSchemas[featureName] = make(map[SchemaName]PluginSchemaConfig)
		}
		config := PluginSchemaConfig{
			Fields: make(map[SchemaField]string),
		}
		for _, opt := range opts {
			opt(&config)
		}
		c.pluginSchemas[featureName][schemaName] = config
	}
}

// --- User schema options ---

func WithUserTableName(tableName SchemaTableName) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaTableName(CoreSchemaUsers, tableName)
	}
}

func WithUserAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigUserOption {
	return func(c *SchemaConfig, u *UserSchema) {
		u.additionalFields = fn
	}
}

// WithUserSerializer overrides the default user response serializer.
func WithUserSerializer(serializer func(data *User) map[string]any) SchemaConfigUserOption {
	return func(c *SchemaConfig, u *UserSchema) {
		u.BaseSchema.Serializer = func(data Model) map[string]any {
			return serializer(data.(*User))
		}
	}
}

func WithUserFieldID(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, SchemaIDField, fieldName)
	}
}

func WithUserFieldEmail(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaEmailField, fieldName)
	}
}

func WithUserFieldPassword(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaPasswordField, fieldName)
	}
}

func WithUserFieldEmailVerifiedAt(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaEmailVerifiedAtField, fieldName)
	}
}

func WithUserFieldSoftDelete(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, SchemaSoftDeleteField, fieldName)
	}
}

func WithUserIncludeNameFields(include bool) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		u.includeNameFields = include
	}
}

func WithUserFirstNameField(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaFirstNameField, fieldName)
	}
}

func WithUserLastNameField(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, UserSchemaLastNameField, fieldName)
	}
}

func WithUserFieldCreatedAt(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, SchemaCreatedAtField, fieldName)
	}
}

func WithUserFieldUpdatedAt(fieldName string) SchemaConfigUserOption {
	return func(s *SchemaConfig, u *UserSchema) {
		s.setCoreSchemaField(CoreSchemaUsers, SchemaUpdatedAtField, fieldName)
	}
}

// --- Session schema options ---

func WithSessionTableName(tableName SchemaTableName) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaTableName(CoreSchemaSessions, tableName)
	}
}

func WithSessionAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigSessionOption {
	return func(c *SchemaConfig, sess *SessionSchema) {
		sess.additionalFields = fn
	}
}

func WithSessionFieldID(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, SchemaIDField, fieldName)
	}
}

func WithSessionFieldToken(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, SessionSchemaTokenField, fieldName)
	}
}

func WithSessionFieldUserID(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, SessionSchemaUserIDField, fieldName)
	}
}

func WithSessionFieldCreatedAt(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, SessionSchemaCreatedAtField, fieldName)
	}
}

func WithSessionFieldExpiresAt(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, SessionSchemaExpiresAtField, fieldName)
	}
}

func WithSessionFieldLastAccess(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, SessionSchemaLastAccessField, fieldName)
	}
}

func WithSessionFieldMetadata(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, SessionSchemaMetadataField, fieldName)
	}
}

// --- Verification schema options ---

func WithVerificationTableName(tableName SchemaTableName) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaTableName(CoreSchemaVerifications, tableName)
	}
}

func WithVerificationAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigVerificationOption {
	return func(c *SchemaConfig, v *VerificationSchema) {
		v.additionalFields = fn
	}
}

func WithVerificationFieldID(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, SchemaIDField, fieldName)
	}
}

func WithVerificationFieldSubject(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, VerificationSchemaSubjectField, fieldName)
	}
}

func WithVerificationFieldValue(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, VerificationSchemaValueField, fieldName)
	}
}

func WithVerificationFieldExpiresAt(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, VerificationSchemaExpiresAtField, fieldName)
	}
}

func WithVerificationFieldCreatedAt(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, SchemaCreatedAtField, fieldName)
	}
}

func WithVerificationFieldUpdatedAt(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, SchemaUpdatedAtField, fieldName)
	}
}

func WithVerificationFieldSoftDelete(fieldName string) SchemaConfigVerificationOption {
	return func(s *SchemaConfig, v *VerificationSchema) {
		s.setCoreSchemaField(CoreSchemaVerifications, SchemaSoftDeleteField, fieldName)
	}
}

// --- Rate limit schema options ---

func WithRateLimitTableName(tableName SchemaTableName) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaTableName(CoreSchemaRateLimits, tableName)
	}
}

func WithRateLimitFieldID(fieldName string) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaField(CoreSchemaRateLimits, SchemaIDField, fieldName)
	}
}

func WithRateLimitFieldKey(fieldName string) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaField(CoreSchemaRateLimits, RateLimitSchemaKeyField, fieldName)
	}
}

func WithRateLimitFieldCount(fieldName string) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaField(CoreSchemaRateLimits, RateLimitSchemaCountField, fieldName)
	}
}

func WithRateLimitFieldLastRequestAt(fieldName string) SchemaConfigRateLimitOption {
	return func(s *SchemaConfig, r *RateLimitSchema) {
		s.setCoreSchemaField(CoreSchemaRateLimits, RateLimitSchemaLastRequestAtField, fieldName)
	}
}

// --- Account schema options ---

// WithAccountSerializer overrides the default account response serializer.
func WithAccountSerializer(serializer func(data *Account) map[string]any) SchemaConfigAccountOption {
	return func(c *SchemaConfig, s *AccountSchema) {
		s.BaseSchema.Serializer = func(data Model) map[string]any {
			return serializer(data.(*Account))
		}
	}
}

func WithAccountTableName(tableName SchemaTableName) SchemaConfigAccountOption {
	return func(c *SchemaConfig, s *AccountSchema) {
		c.setCoreSchemaTableName(CoreSchemaAccounts, tableName)
	}
}
