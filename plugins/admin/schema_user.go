package admin

import (
	"github.com/thecodearcher/limen"
)

const (
	UserSchemaBannedAtField     limen.SchemaField = "banned_at"
	UserSchemaBanReasonField    limen.SchemaField = "ban_reason"
	UserSchemaBanExpiresAtField limen.SchemaField = "ban_expires_at"
)

type userSchema struct {
	*limen.UserSchema
}

func (s *userSchema) GetBannedAtField() string {
	return s.GetField(UserSchemaBannedAtField)
}

func (s *userSchema) GetBanReasonField() string {
	return s.GetField(UserSchemaBanReasonField)
}

func (s *userSchema) GetBanExpiresAtField() string {
	return s.GetField(UserSchemaBanExpiresAtField)
}

func buildUserBanExtension(schema *userSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForExtension(
		limen.CoreSchemaUsers,
		schema,
		limen.WithSchemaField(UserSchemaBannedAtField, limen.ColumnTypeTime, limen.WithNullable(true)),
		limen.WithSchemaField(UserSchemaBanReasonField, limen.ColumnTypeText, limen.WithNullable(true)),
		limen.WithSchemaField(UserSchemaBanExpiresAtField, limen.ColumnTypeTime, limen.WithNullable(true)),
	)
}
