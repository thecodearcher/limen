package schemas

import "context"

type SchemaField string
type TableName string

// defaults for schemas table names
const (
	UserSchemaTableName         TableName = "users"
	VerificationSchemaTableName TableName = "verifications"
)

// defaults for schemas fields
const (
	SchemaIDField             SchemaField = "id"
	UserSchemaEmailField      SchemaField = "email"
	UserSchemaFirstNameField  SchemaField = "first_name"
	UserSchemaLastNameField   SchemaField = "last_name"
	UserSchemaPasswordField   SchemaField = "password"
	UserSchemaCreatedAtField  SchemaField = "created_at"
	UserSchemaUpdatedAtField  SchemaField = "updated_at"
	UserSchemaSoftDeleteField SchemaField = "deleted_at"

	VerificationSchemaSubjectField   SchemaField = "subject"
	VerificationSchemaValueField     SchemaField = "value"
	VerificationSchemaExpiresAtField SchemaField = "expires_at"
	VerificationSchemaCreatedAtField SchemaField = "created_at"
	VerificationSchemaUpdatedAtField SchemaField = "updated_at"
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
