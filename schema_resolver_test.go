package limen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaResolver_GetField(t *testing.T) {
	t.Parallel()

	schemas := map[SchemaName]SchemaDefinition{
		"users": {
			TableName: "users",
			Columns: []ColumnDefinition{
				{LogicalField: "id", Name: "id"},
				{LogicalField: UserSchemaEmailField, Name: "email"},
				{LogicalField: UserSchemaPasswordField, Name: "hashed_password"},
			},
		},
		"sessions": {
			TableName: "sessions",
			Columns: []ColumnDefinition{
				{LogicalField: "id", Name: "id"},
				{LogicalField: SessionSchemaTokenField, Name: "token"},
			},
		},
	}

	resolver := newFieldResolver(schemas)

	assert.Equal(t, "email", resolver.GetField("users", UserSchemaEmailField))
	assert.Equal(t, "hashed_password", resolver.GetField("users", UserSchemaPasswordField))
	assert.Equal(t, "token", resolver.GetField("sessions", SessionSchemaTokenField))
	assert.Equal(t, "", resolver.GetField("users", "nonexistent"))
	assert.Equal(t, "", resolver.GetField("unknown_schema", "id"))
}

func TestSchemaResolver_GetTableName(t *testing.T) {
	t.Parallel()

	schemas := map[SchemaName]SchemaDefinition{
		"users":    {TableName: "app_users"},
		"sessions": {TableName: "auth_sessions"},
	}

	resolver := newFieldResolver(schemas)

	assert.Equal(t, SchemaTableName("app_users"), resolver.GetTableName("users"))
	assert.Equal(t, SchemaTableName("auth_sessions"), resolver.GetTableName("sessions"))
	assert.Equal(t, SchemaTableName(""), resolver.GetTableName("unknown"))
}

func TestSchemaResolver_GetFields(t *testing.T) {
	t.Parallel()

	schemas := map[SchemaName]SchemaDefinition{
		"users": {
			TableName: "users",
			Columns: []ColumnDefinition{
				{LogicalField: "id", Name: "id"},
				{LogicalField: UserSchemaEmailField, Name: "email_something"},
			},
		},
	}

	resolver := newFieldResolver(schemas)
	fields := resolver.GetFields("users")

	assert.Len(t, fields, 2)
	assert.Equal(t, "id", fields["id"])
	assert.Equal(t, "email_something", fields[UserSchemaEmailField])
}

func TestSchemaInfo_GetField(t *testing.T) {
	t.Parallel()

	schemas := map[SchemaName]SchemaDefinition{
		"users": {
			TableName: "users",
			Columns: []ColumnDefinition{
				{LogicalField: "id", Name: "user_id"},
				{LogicalField: UserSchemaEmailField, Name: "user_email"},
			},
		},
	}

	resolver := newFieldResolver(schemas)
	info := newSchemaInfo("users", "users", resolver)

	assert.Equal(t, "user_id", info.GetField("id"))
	assert.Equal(t, "user_email", info.GetField(UserSchemaEmailField))
}
