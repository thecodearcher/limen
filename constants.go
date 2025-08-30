package aegis

type SchemaField string
type SchemaTableName string

// defaults for schemas table names
const (
	UserSchemaTableName SchemaTableName = "users"
)

// defaults for schemas fields
const (
	SchemaIDField            SchemaField = "id"
	UserSchemaEmailField     SchemaField = "email"
	UserSchemaFirstNameField SchemaField = "first_name"
	UserSchemaLastNameField  SchemaField = "last_name"
	UserSchemaPasswordField  SchemaField = "password"
	UserSchemaCreatedAtField SchemaField = "created_at"
	UserSchemaUpdatedAtField SchemaField = "updated_at"
)
