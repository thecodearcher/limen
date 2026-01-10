package aegis

// PendingAction represents a pending action for a user after authentication
type PendingAction string

// defaults for pending actions
const (
	PendingActionEmailVerification     PendingAction = "email_verification"
	PendingActionTwoFactorVerification PendingAction = "two_factor_verification"
)

// SessionStrategyType represents the type of session strategy
type SessionStrategyType string

// Session strategy types
const (
	SessionStrategyOpaqueToken SessionStrategyType = "opaque_token"
)

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
	// TokenDeliveryHeader delivers tokens in response headers
	TokenDeliveryHeader TokenDeliveryMethod = "header"
)

type RateLimiterStoreType string

const (
	RateLimiterStoreTypeMemory   RateLimiterStoreType = "in_memory"
	RateLimiterStoreTypeDatabase RateLimiterStoreType = "database"
)

// CoreSchemaName represents a valid core schema name that can be extended by plugins
type CoreSchemaName string

const (
	// CoreSchemaUsers is the name of the users core schema
	CoreSchemaUsers CoreSchemaName = "users"
	// CoreSchemaSessions is the name of the sessions core schema
	CoreSchemaSessions CoreSchemaName = "sessions"
	// CoreSchemaVerifications is the name of the verifications core schema
	CoreSchemaVerifications CoreSchemaName = "verifications"
	// CoreSchemaRateLimits is the name of the rate_limits core schema
	CoreSchemaRateLimits CoreSchemaName = "rate_limits"
)

// ColumnType represents a Go type for a database column
type ColumnType string

const (
	// ColumnTypeUUID represents the uuid string type
	ColumnTypeUUID ColumnType = "uuid"
	// ColumnTypeString represents the string type
	ColumnTypeString ColumnType = "string"
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
