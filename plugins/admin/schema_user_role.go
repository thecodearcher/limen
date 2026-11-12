package admin

import (
	"time"

	"github.com/thecodearcher/limen"
)

type UserRole struct {
	ID     any
	UserID any
	Role   string

	CreatedAt time.Time

	raw map[string]any
}

func (r *UserRole) Raw() map[string]any {
	return r.raw
}

const (
	UserRoleSchemaTableName limen.SchemaTableName = "user_roles"

	UserRoleSchemaUserIDField limen.SchemaField = "user_id"
	UserRoleSchemaRoleField   limen.SchemaField = "role"
)

type userRoleSchema struct {
	limen.BaseSchema
}

func newUserRoleSchema() *userRoleSchema {
	return &userRoleSchema{}
}

func (s *userRoleSchema) GetUserIDField() string {
	return s.GetField(UserRoleSchemaUserIDField)
}

func (s *userRoleSchema) GetRoleField() string {
	return s.GetField(UserRoleSchemaRoleField)
}

func (s *userRoleSchema) GetCreatedAtField() string {
	return s.GetField(limen.SchemaCreatedAtField)
}

func (s *userRoleSchema) ToStorage(data limen.Model) map[string]any {
	userRole := data.(*UserRole)
	return map[string]any{
		s.GetUserIDField(): userRole.UserID,
		s.GetRoleField():   userRole.Role,
	}
}

func (s *userRoleSchema) FromStorage(data map[string]any) limen.Model {
	return &UserRole{
		ID:        data[s.GetIDField()],
		UserID:    limen.GetValue[any](data[s.GetUserIDField()]),
		Role:      limen.GetValue[string](data[s.GetRoleField()]),
		CreatedAt: limen.GetValue[time.Time](data[s.GetCreatedAtField()]),
		raw:       data,
	}
}

func buildUserRoleTableDef(schemaConfig *limen.SchemaConfig, schema *userRoleSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(UserRoleSchemaTableName),
		UserRoleSchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(UserRoleSchemaUserIDField, schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(UserRoleSchemaRoleField, limen.ColumnTypeString),
		limen.WithSchemaCreatedAtField(),

		limen.WithSchemaUniqueIndex("idx_user_roles_user_role", []limen.SchemaField{
			UserRoleSchemaUserIDField,
			UserRoleSchemaRoleField,
		}),

		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_user_roles_users_user_id",
			Column:           UserRoleSchemaUserIDField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
		limen.WithDisablePublicID(),
	)
}
