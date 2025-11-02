package schemas

import (
	"context"
)

type SchemaField string
type TableName string

// defaults for schemas table names
const (
	UserSchemaTableName         TableName = "users"
	VerificationSchemaTableName TableName = "verifications"
	SessionSchemaTableName      TableName = "sessions"
)

// defaults for schemas fields
const (
	SchemaIDField                  SchemaField = "id"
	UserSchemaEmailField           SchemaField = "email"
	UserSchemaPasswordField        SchemaField = "password"
	UserSchemaEmailVerifiedAtField SchemaField = "email_verified_at"
	UserSchemaCreatedAtField       SchemaField = "created_at"
	UserSchemaUpdatedAtField       SchemaField = "updated_at"
	UserSchemaSoftDeleteField      SchemaField = "deleted_at"

	VerificationSchemaSubjectField   SchemaField = "subject"
	VerificationSchemaValueField     SchemaField = "value"
	VerificationSchemaExpiresAtField SchemaField = "expires_at"
	VerificationSchemaCreatedAtField SchemaField = "created_at"
	VerificationSchemaUpdatedAtField SchemaField = "updated_at"

	SessionSchemaUserIDField     SchemaField = "user_id"
	SessionSchemaDataField       SchemaField = "data"
	SessionSchemaCreatedAtField  SchemaField = "created_at"
	SessionSchemaExpiresAtField  SchemaField = "expires_at"
	SessionSchemaLastAccessField SchemaField = "last_access"
	SessionSchemaIPAddressField  SchemaField = "ip_address"
	SessionSchemaUserAgentField  SchemaField = "user_agent"
	SessionSchemaCSRFTokenField  SchemaField = "csrf_token"
	SessionSchemaMetadataField   SchemaField = "metadata"
)

type AdditionalFieldsFunc func(ctx context.Context) map[string]any

type Schema[T Model] interface {
	GetTableName() TableName
	ToStorage(data *T) map[string]any
	FromStorage(data map[string]any) *T
	GetSoftDeleteField() SchemaField
	GetAdditionalFields() AdditionalFieldsFunc
}

type Model interface {
	// Name returns the table name of the model
	TableName() string
	// Raw returns the model raw data as returned from the database
	Raw() map[string]any
}

func getFieldOrDefault(fieldValue string, defaultValue SchemaField) string {
	if fieldValue == "" {
		return string(defaultValue)
	}
	return fieldValue
}

func getNullableValue[T any](value any) *T {
	if value == nil {
		return nil
	}
	v := value.(T)
	return &v
}
