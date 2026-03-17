package sessionjwt

import (
	"time"

	"github.com/thecodearcher/limen"
)

// ============================================================================
// Constants
// ============================================================================

const (
	RefreshTokenSchemaTableName limen.SchemaTableName = "jwt_refresh_tokens"

	RefreshTokenSchemaTokenField     limen.SchemaField = "token"
	RefreshTokenSchemaUserIDField    limen.SchemaField = "user_id"
	RefreshTokenSchemaJWTIDField     limen.SchemaField = "jwt_id"
	RefreshTokenSchemaFamilyField    limen.SchemaField = "family"
	RefreshTokenSchemaExpiresAtField limen.SchemaField = "expires_at"
	RefreshTokenSchemaCreatedAtField limen.SchemaField = "created_at"
)

const (
	BlacklistSchemaTableName limen.SchemaTableName = "jwt_blacklist"

	BlacklistSchemaJTIField       limen.SchemaField = "jti"
	BlacklistSchemaExpiresAtField limen.SchemaField = "expires_at"
)

// ============================================================================
// Refresh Token Schema
// ============================================================================

type refreshTokenSchema struct {
	limen.BaseSchema
}

func newRefreshTokenSchema() *refreshTokenSchema {
	return &refreshTokenSchema{BaseSchema: limen.BaseSchema{}}
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

func (s *refreshTokenSchema) ToStorage(data limen.Model) map[string]any {
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

func (s *refreshTokenSchema) FromStorage(data map[string]any) limen.Model {
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
	limen.BaseSchema
}

func newBlacklistSchema() *blacklistSchema {
	return &blacklistSchema{BaseSchema: limen.BaseSchema{}}
}

func (s *blacklistSchema) GetSoftDeleteField() string { return "" }

func (s *blacklistSchema) GetJTIField() string {
	return s.GetField(BlacklistSchemaJTIField)
}

func (s *blacklistSchema) GetExpiresAtField() string {
	return s.GetField(BlacklistSchemaExpiresAtField)
}

func (s *blacklistSchema) ToStorage(data limen.Model) map[string]any {
	entry := data.(*BlacklistEntry)
	return map[string]any{
		s.GetJTIField():       entry.JTI,
		s.GetExpiresAtField(): entry.ExpiresAt,
	}
}

func (s *blacklistSchema) FromStorage(data map[string]any) limen.Model {
	return &BlacklistEntry{
		JTI:       data[s.GetJTIField()].(string),
		ExpiresAt: data[s.GetExpiresAtField()].(time.Time),
		raw:       data,
	}
}

// ============================================================================
// Schema Definitions (used in GetSchemas)
// ============================================================================

func buildRefreshTokenTableDef(schemaConfig *limen.SchemaConfig, schema *refreshTokenSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(RefreshTokenSchemaTableName),
		RefreshTokenSchemaTableName,
		schema,
		limen.WithSchemaIDField(schemaConfig),
		limen.WithSchemaField(string(RefreshTokenSchemaTokenField), limen.ColumnTypeString),
		limen.WithSchemaField(string(RefreshTokenSchemaUserIDField), schemaConfig.GetIDColumnType()),
		limen.WithSchemaField(string(RefreshTokenSchemaJWTIDField), limen.ColumnTypeString),
		limen.WithSchemaField(string(RefreshTokenSchemaFamilyField), limen.ColumnTypeString),
		limen.WithSchemaField(string(RefreshTokenSchemaExpiresAtField), limen.ColumnTypeTime),
		limen.WithSchemaField(string(RefreshTokenSchemaCreatedAtField), limen.ColumnTypeTime, limen.WithDefaultValue(string(limen.DatabaseDefaultValueNow))),
		limen.WithSchemaUniqueIndex("idx_jwt_refresh_tokens_token", []limen.SchemaField{RefreshTokenSchemaTokenField}),
		limen.WithSchemaIndex("idx_jwt_refresh_tokens_user_id", []limen.SchemaField{RefreshTokenSchemaUserIDField}),
		limen.WithSchemaIndex("idx_jwt_refresh_tokens_family", []limen.SchemaField{RefreshTokenSchemaFamilyField}),
		limen.WithSchemaForeignKey(limen.ForeignKeyDefinition{
			Name:             "fk_jwt_refresh_tokens_users_user_id",
			Column:           RefreshTokenSchemaUserIDField,
			ReferencedSchema: limen.CoreSchemaUsers,
			ReferencedField:  limen.SchemaIDField,
			OnDelete:         limen.FKActionCascade,
			OnUpdate:         limen.FKActionCascade,
		}),
	)
}

func buildBlacklistTableDef(schema *blacklistSchema) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForTable(
		limen.SchemaName(BlacklistSchemaTableName),
		BlacklistSchemaTableName,
		schema,
		limen.WithSchemaField(string(BlacklistSchemaJTIField), limen.ColumnTypeString, limen.WithPrimaryKey(true)),
		limen.WithSchemaField(string(BlacklistSchemaExpiresAtField), limen.ColumnTypeTime),
		limen.WithSchemaIndex("idx_jwt_blacklist_expires_at", []limen.SchemaField{BlacklistSchemaExpiresAtField}),
	)
}
