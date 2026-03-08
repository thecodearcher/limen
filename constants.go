package aegis

// ============================================================================
// Authentication & Session Constants
// ============================================================================

type EnvelopeMode int

const (
	EnvelopeOff EnvelopeMode = iota
	EnvelopeWrapSuccess
	EnvelopeAlways
)

type SessionStoreType string

const (
	SessionStoreTypeMemory   SessionStoreType = "in_memory"
	SessionStoreTypeDatabase SessionStoreType = "database"
)

// TokenDeliveryMethod specifies how tokens should be delivered
type TokenDeliveryMethod string

const (
	// TokenDeliveryCookie delivers tokens via HttpOnly cookies
	TokenDeliveryCookie TokenDeliveryMethod = "cookie"
)

type RateLimiterStoreType string

const (
	RateLimiterStoreTypeMemory   RateLimiterStoreType = "in_memory"
	RateLimiterStoreTypeDatabase RateLimiterStoreType = "database"
)

// ============================================================================
// Plugin Constants
// ============================================================================

// PluginName represents the name of a plugin/plugin
type PluginName string

// Plugin Names
const (
	PluginCredentialPassword PluginName = "credential-password"
	PluginTwoFactor          PluginName = "two-factor"
	PluginOAuth              PluginName = "oauth"
	PluginSessionJWT         PluginName = "session-jwt"
)

// ============================================================================
// Schema Constants
// ============================================================================

// SchemaName represents the logical name of a schema
type SchemaName string

// SchemaField represents a logical field name in a schema
type SchemaField string

// SchemaTableName represents the actual database table name
type SchemaTableName string

// Core Schema Names
const (
	// CoreSchemaUsers is the name of the users core schema
	CoreSchemaUsers SchemaName = "users"
	// CoreSchemaSessions is the name of the sessions core schema
	CoreSchemaSessions SchemaName = "sessions"
	// CoreSchemaVerifications is the name of the verifications core schema
	CoreSchemaVerifications SchemaName = "verifications"
	// CoreSchemaRateLimits is the name of the rate_limits core schema
	CoreSchemaRateLimits SchemaName = "rate_limits"
	// CoreSchemaAccounts is the name of the accounts core schema (OAuth linked accounts)
	CoreSchemaAccounts SchemaName = "accounts"
)

// Schema Table Names
const (
	UserSchemaTableName         SchemaTableName = "users"
	VerificationSchemaTableName SchemaTableName = "verifications"
	SessionSchemaTableName      SchemaTableName = "sessions"
	RateLimitSchemaTableName    SchemaTableName = "rate_limits"
	AccountSchemaTableName      SchemaTableName = "accounts"
)

// Schema Field Names
const (
	// Common schema fields
	SchemaIDField         SchemaField = "id"
	SchemaCreatedAtField  SchemaField = "created_at"
	SchemaUpdatedAtField  SchemaField = "updated_at"
	SchemaSoftDeleteField SchemaField = "deleted_at"

	// User schema fields
	UserSchemaFirstNameField       SchemaField = "first_name"
	UserSchemaLastNameField        SchemaField = "last_name"
	UserSchemaEmailField           SchemaField = "email"
	UserSchemaPasswordField        SchemaField = "password"
	UserSchemaEmailVerifiedAtField SchemaField = "email_verified_at"

	// Verification schema fields
	VerificationSchemaSubjectField   SchemaField = "subject"
	VerificationSchemaValueField     SchemaField = "value"
	VerificationSchemaExpiresAtField SchemaField = "expires_at"

	// Session schema fields
	SessionSchemaUserIDField     SchemaField = "user_id"
	SessionSchemaTokenField      SchemaField = "token"
	SessionSchemaCreatedAtField  SchemaField = "created_at"
	SessionSchemaExpiresAtField  SchemaField = "expires_at"
	SessionSchemaLastAccessField SchemaField = "last_access"
	SessionSchemaMetadataField   SchemaField = "metadata"

	// Rate limit schema fields
	RateLimitSchemaKeyField           SchemaField = "key"
	RateLimitSchemaCountField         SchemaField = "count"
	RateLimitSchemaLastRequestAtField SchemaField = "last_request_at"

	// Account schema fields (OAuth)
	AccountSchemaUserIDField               SchemaField = "user_id"
	AccountSchemaProviderField             SchemaField = "provider"
	AccountSchemaProviderAccountIDField    SchemaField = "provider_account_id"
	AccountSchemaAccessTokenField          SchemaField = "access_token"
	AccountSchemaRefreshTokenField         SchemaField = "refresh_token"
	AccountSchemaAccessTokenExpiresAtField SchemaField = "access_token_expires_at"
	AccountSchemaScopeField                SchemaField = "scope"
	AccountSchemaIDTokenField              SchemaField = "id_token"
)

// ============================================================================
// Database Schema Constants
// ============================================================================

// ColumnType represents a Go type for a database column
type ColumnType string

const (
	// ColumnTypeUUID represents the uuid string type
	ColumnTypeUUID ColumnType = "uuid"
	// ColumnTypeString represents the string (VARCHAR(255)) type
	ColumnTypeString ColumnType = "string"
	// ColumnTypeText represents the text (TEXT) type
	ColumnTypeText ColumnType = "text"
	// ColumnTypeInt represents the int type
	ColumnTypeInt ColumnType = "int"
	// ColumnTypeInt32 represents the int32 type
	ColumnTypeInt32 ColumnType = "int32"
	// ColumnTypeInt64 represents the int64 type
	ColumnTypeInt64 ColumnType = "int64"
	// ColumnTypeBool represents the bool type
	ColumnTypeBool ColumnType = "bool"
	// ColumnTypeTime represents the time.Time type
	ColumnTypeTime ColumnType = "time.Time"
	// ColumnTypeAny represents the any type
	ColumnTypeAny ColumnType = "any"
	// ColumnTypeMapStringAny represents the map[string]any type
	ColumnTypeMapStringAny ColumnType = "map[string]any"
)

// ForeignKeyAction represents a SQL foreign key action
type ForeignKeyAction string

const (
	// FKActionCascade represents CASCADE action
	FKActionCascade ForeignKeyAction = "CASCADE"
	// FKActionSetNull represents SET NULL action
	FKActionSetNull ForeignKeyAction = "SET NULL"
	// FKActionRestrict represents RESTRICT action
	FKActionRestrict ForeignKeyAction = "RESTRICT"
	// FKActionNoAction represents NO ACTION
	FKActionNoAction ForeignKeyAction = "NO ACTION"
	// FKActionSetDefault represents SET DEFAULT action
	FKActionSetDefault ForeignKeyAction = "SET DEFAULT"
)

// DatabaseDefaultValue represents special default value constants that map to database-specific SQL
type DatabaseDefaultValue string

const (
	// DefaultValuePrefix prefix for special database default values e.g CURRENT_TIMESTAMP
	DatabaseDefaultValuePrefix = "@"
	// DatabaseDefaultValueNow represents a timestamp default that maps to CURRENT_TIMESTAMP
	DatabaseDefaultValueNow DatabaseDefaultValue = DatabaseDefaultValuePrefix + "now()"
	// DatabaseDefaultValueUUID represents a UUID generation default that maps to uuid_generate_v4() (PostgreSQL) or UUID() (MySQL)
	DatabaseDefaultValueUUID DatabaseDefaultValue = DatabaseDefaultValuePrefix + "uuid()"
)
