package sessionjwt

import (
	"time"

	"github.com/thecodearcher/aegis"
)

// ============================================================================
// Constants
// ============================================================================

const (
	RefreshTokenSchemaTableName aegis.SchemaTableName = "jwt_refresh_tokens"

	RefreshTokenSchemaTokenField     aegis.SchemaField = "token"
	RefreshTokenSchemaUserIDField    aegis.SchemaField = "user_id"
	RefreshTokenSchemaJWTIDField     aegis.SchemaField = "jwt_id"
	RefreshTokenSchemaFamilyField    aegis.SchemaField = "family"
	RefreshTokenSchemaExpiresAtField aegis.SchemaField = "expires_at"
	RefreshTokenSchemaCreatedAtField aegis.SchemaField = "created_at"
)

const (
	BlacklistSchemaTableName aegis.SchemaTableName = "jwt_blacklist"

	BlacklistSchemaJTIField       aegis.SchemaField = "jti"
	BlacklistSchemaExpiresAtField aegis.SchemaField = "expires_at"
)

// ============================================================================
// Refresh Token Schema
// ============================================================================

type refreshTokenSchema struct {
	aegis.BaseSchema
}

func newRefreshTokenSchema() *refreshTokenSchema {
	return &refreshTokenSchema{BaseSchema: aegis.BaseSchema{}}
}

func (s *refreshTokenSchema) GetSoftDeleteField() string { return "" }

func (s *refreshTokenSchema) GetTokenField() string {
	return s.GetField(RefreshTokenSchemaTokenField)
}

func (s *refreshTokenSchema) GetUserIDField() string {
	return s.GetField(RefreshTokenSchemaUserIDField)
}

func (s *refreshTokenSchema) GetJWTIDField() string {
	return s.GetField(RefreshTokenSchemaJWTIDField)
}

func (s *refreshTokenSchema) GetFamilyField() string {
	return s.GetField(RefreshTokenSchemaFamilyField)
}

func (s *refreshTokenSchema) GetExpiresAtField() string {
	return s.GetField(RefreshTokenSchemaExpiresAtField)
}

func (s *refreshTokenSchema) GetCreatedAtField() string {
	return s.GetField(RefreshTokenSchemaCreatedAtField)
}

func (s *refreshTokenSchema) ToStorage(data aegis.Model) map[string]any {
	rt := data.(*RefreshToken)
	return map[string]any{
		s.GetTokenField():     rt.Token,
		s.GetUserIDField():    rt.UserID,
		s.GetJWTIDField():     rt.JWTID,
		s.GetFamilyField():    rt.Family,
		s.GetExpiresAtField(): rt.ExpiresAt,
		s.GetCreatedAtField(): rt.CreatedAt,
	}
}

func (s *refreshTokenSchema) FromStorage(data map[string]any) aegis.Model {
	return &RefreshToken{
		ID:        data[s.GetIDField()],
		Token:     data[s.GetTokenField()].(string),
		UserID:    data[s.GetUserIDField()],
		JWTID:     data[s.GetJWTIDField()].(string),
		Family:    data[s.GetFamilyField()].(string),
		ExpiresAt: data[s.GetExpiresAtField()].(time.Time),
		CreatedAt: data[s.GetCreatedAtField()].(time.Time),
		raw:       data,
	}
}

// ============================================================================
// Blacklist Schema
// ============================================================================

type blacklistSchema struct {
	aegis.BaseSchema
}

func newBlacklistSchema() *blacklistSchema {
	return &blacklistSchema{BaseSchema: aegis.BaseSchema{}}
}

func (s *blacklistSchema) GetSoftDeleteField() string { return "" }

func (s *blacklistSchema) GetJTIField() string {
	return s.GetField(BlacklistSchemaJTIField)
}

func (s *blacklistSchema) GetExpiresAtField() string {
	return s.GetField(BlacklistSchemaExpiresAtField)
}

func (s *blacklistSchema) ToStorage(data aegis.Model) map[string]any {
	entry := data.(*BlacklistEntry)
	return map[string]any{
		s.GetJTIField():       entry.JTI,
		s.GetExpiresAtField(): entry.ExpiresAt,
	}
}

func (s *blacklistSchema) FromStorage(data map[string]any) aegis.Model {
	return &BlacklistEntry{
		JTI:       data[s.GetJTIField()].(string),
		ExpiresAt: data[s.GetExpiresAtField()].(time.Time),
		raw:       data,
	}
}

// ============================================================================
// Schema Definitions (used in GetSchemas)
// ============================================================================

func buildRefreshTokenTableDef(schemaConfig *aegis.SchemaConfig, schema *refreshTokenSchema) *aegis.SchemaDefinition {
	return aegis.NewSchemaDefinitionForTable(
		aegis.SchemaName(RefreshTokenSchemaTableName),
		RefreshTokenSchemaTableName,
		schema,
		aegis.WithSchemaIDField(schemaConfig),
		aegis.WithSchemaField(string(RefreshTokenSchemaTokenField), aegis.ColumnTypeString),
		aegis.WithSchemaField(string(RefreshTokenSchemaUserIDField), schemaConfig.GetIDColumnType()),
		aegis.WithSchemaField(string(RefreshTokenSchemaJWTIDField), aegis.ColumnTypeString),
		aegis.WithSchemaField(string(RefreshTokenSchemaFamilyField), aegis.ColumnTypeString),
		aegis.WithSchemaField(string(RefreshTokenSchemaExpiresAtField), aegis.ColumnTypeTime),
		aegis.WithSchemaField(string(RefreshTokenSchemaCreatedAtField), aegis.ColumnTypeTime, aegis.WithDefaultValue(string(aegis.DatabaseDefaultValueNow))),
		aegis.WithSchemaUniqueIndex("idx_jwt_refresh_tokens_token", []aegis.SchemaField{RefreshTokenSchemaTokenField}),
		aegis.WithSchemaIndex("idx_jwt_refresh_tokens_user_id", []aegis.SchemaField{RefreshTokenSchemaUserIDField}),
		aegis.WithSchemaIndex("idx_jwt_refresh_tokens_family", []aegis.SchemaField{RefreshTokenSchemaFamilyField}),
		aegis.WithSchemaForeignKey(aegis.ForeignKeyDefinition{
			Name:             "fk_jwt_refresh_tokens_users_user_id",
			Column:           RefreshTokenSchemaUserIDField,
			ReferencedSchema: aegis.CoreSchemaUsers,
			ReferencedField:  aegis.SchemaIDField,
			OnDelete:         aegis.FKActionCascade,
			OnUpdate:         aegis.FKActionCascade,
		}),
	)
}

func buildBlacklistTableDef(schema *blacklistSchema) *aegis.SchemaDefinition {
	return aegis.NewSchemaDefinitionForTable(
		aegis.SchemaName(BlacklistSchemaTableName),
		BlacklistSchemaTableName,
		schema,
		aegis.WithSchemaField(string(BlacklistSchemaJTIField), aegis.ColumnTypeString, aegis.WithPrimaryKey(true)),
		aegis.WithSchemaField(string(BlacklistSchemaExpiresAtField), aegis.ColumnTypeTime),
		aegis.WithSchemaIndex("idx_jwt_blacklist_expires_at", []aegis.SchemaField{BlacklistSchemaExpiresAtField}),
	)
}
