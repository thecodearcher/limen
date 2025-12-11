package aegis

import (
	"encoding/json"
	"time"
)

type Session struct {
	ID         any
	Token      string
	UserID     any
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastAccess time.Time
	Metadata   map[string]any
	raw        map[string]any
}

// Raw returns the session raw data as returned from the database
func (s Session) Raw() map[string]any {
	return s.raw
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired(idleTimeout time.Duration) bool {
	if idleTimeout == 0 {
		return time.Now().After(s.ExpiresAt)
	}
	return time.Now().After(s.ExpiresAt) || time.Now().After(s.LastAccess.Add(idleTimeout))
}

// ShouldExtendExpiration checks if the session should be extended
func (s *Session) ShouldExtendExpiration(expiresIn, updateAge time.Duration) bool {
	if updateAge == 0 {
		return false
	}

	lastExtendedAt := s.ExpiresAt.Add(-expiresIn)
	nextExtensionAt := lastExtendedAt.Add(updateAge)

	return time.Now().After(nextExtensionAt) || time.Now().Equal(nextExtensionAt)
}

// Touch updates the last access time
func (s *Session) Touch() {
	s.LastAccess = time.Now()
}

type SessionSchema struct {
	BaseSchema
}

type SchemaConfigSessionOption func(*SchemaConfig, *SessionSchema)

func newDefaultSessionSchema(c *SchemaConfig, opts ...SchemaConfigSessionOption) *SessionSchema {
	schema := &SessionSchema{
		BaseSchema: BaseSchema{},
	}

	for _, opt := range opts {
		opt(c, schema)
	}

	return schema
}

func (s *SessionSchema) GetSoftDeleteField() string {
	// sessions should not have a soft delete field
	return ""
}

func (s *SessionSchema) GetUserIDField() string {
	return s.GetField(string(SessionSchemaUserIDField))
}

func (s *SessionSchema) GetTokenField() string {
	return s.GetField(string(SessionSchemaTokenField))
}

func (s *SessionSchema) GetCreatedAtField() string {
	return s.GetField(string(SessionSchemaCreatedAtField))
}

func (s *SessionSchema) GetExpiresAtField() string {
	return s.GetField(string(SessionSchemaExpiresAtField))
}

func (s *SessionSchema) GetLastAccessField() string {
	return s.GetField(string(SessionSchemaLastAccessField))
}

func (s *SessionSchema) GetMetadataField() string {
	return s.GetField(string(SessionSchemaMetadataField))
}

func (s *SessionSchema) FromStorage(data map[string]any) Model {
	session := &Session{
		ID:         data[s.GetIDField()],
		Token:      data[s.GetTokenField()].(string),
		UserID:     data[s.GetUserIDField()],
		CreatedAt:  data[s.GetCreatedAtField()].(time.Time),
		ExpiresAt:  data[s.GetExpiresAtField()].(time.Time),
		LastAccess: data[s.GetLastAccessField()].(time.Time),
		raw:        data,
	}

	// Parse Metadata if it exists and is a string (JSON)
	if metadataValue, exists := data[s.GetMetadataField()]; exists && metadataValue != nil {
		if metadataStr, ok := metadataValue.(string); ok && metadataStr != "" {
			var metadata map[string]any
			if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
				session.Metadata = metadata
			}
		} else if metadataMap, ok := metadataValue.(map[string]any); ok {
			session.Metadata = metadataMap
		}
	}

	return session
}

func (s *SessionSchema) ToStorage(data Model) map[string]any {
	session := data.(*Session)
	result := map[string]any{
		s.GetTokenField():      session.Token,
		s.GetUserIDField():     session.UserID,
		s.GetCreatedAtField():  session.CreatedAt,
		s.GetExpiresAtField():  session.ExpiresAt,
		s.GetLastAccessField(): session.LastAccess,
	}

	if session.Metadata != nil {
		if metadataJSON, err := json.Marshal(session.Metadata); err == nil {
			result[s.GetMetadataField()] = string(metadataJSON)
		}
	}

	return result
}

func WithSessionTableName(tableName SchemaTableName) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaTableName(CoreSchemaSessions, tableName)
	}
}

func WithSessionAdditionalFields(fn AdditionalFieldsFunc) SchemaConfigSessionOption {
	return func(c *SchemaConfig, sess *SessionSchema) {
		sess.additionalFields = fn
	}
}

func WithSessionFieldID(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, string(SchemaIDField), fieldName)
	}
}

func WithSessionFieldToken(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, string(SessionSchemaTokenField), fieldName)
	}
}

func WithSessionFieldUserID(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, string(SessionSchemaUserIDField), fieldName)
	}
}

func WithSessionFieldCreatedAt(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, string(SessionSchemaCreatedAtField), fieldName)
	}
}

func WithSessionFieldExpiresAt(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, string(SessionSchemaExpiresAtField), fieldName)
	}
}

func WithSessionFieldLastAccess(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, string(SessionSchemaLastAccessField), fieldName)
	}
}

func WithSessionFieldMetadata(fieldName string) SchemaConfigSessionOption {
	return func(s *SchemaConfig, sess *SessionSchema) {
		s.setCoreSchemaField(CoreSchemaSessions, string(SessionSchemaMetadataField), fieldName)
	}
}

func (s *SessionSchema) Introspect() SchemaIntrospector {
	tableName := SessionSchemaTableName
	return &SchemaDefinition{
		TableName: &tableName,
		Columns:   s.getDefaultColumns(),
		Indexes: []IndexDefinition{
			{
				Name:    "idx_sessions_token",
				Columns: []string{s.GetTokenField()},
				Unique:  true,
			},
			{
				Name:    "idx_sessions_user_id",
				Columns: []string{s.GetUserIDField()},
				Unique:  false,
			},
		},
		ForeignKeys: []ForeignKeyDefinition{
			{
				Name:             "fk_sessions_user_id",
				Column:           s.GetUserIDField(),
				ReferencedSchema: UserSchemaTableName,
				ReferencedField:  SchemaIDField,
				OnDelete:         FKActionCascade,
				OnUpdate:         FKActionCascade,
			},
		},
		SchemaName: string(CoreSchemaSessions),
		Extends:    nil,
		Schema:     s,
	}
}

func (s *SessionSchema) getDefaultColumns() []ColumnDefinition {
	return []ColumnDefinition{
		{
			Name:         string(SchemaIDField),
			LogicalField: string(SchemaIDField),
			Type:         ColumnTypeAny,
			IsNullable:   false,
			IsPrimaryKey: true,
			Tags: map[string]string{
				"json": "id",
			},
		},
		{
			Name:         string(SessionSchemaTokenField),
			LogicalField: string(SessionSchemaTokenField),
			Type:         ColumnTypeString,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "token",
			},
		},
		{
			Name:         string(SessionSchemaUserIDField),
			LogicalField: string(SessionSchemaUserIDField),
			Type:         ColumnTypeAny,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "user_id",
			},
		},
		{
			Name:         string(SessionSchemaCreatedAtField),
			LogicalField: string(SessionSchemaCreatedAtField),
			Type:         ColumnTypeTime,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "created_at",
			},
		},
		{
			Name:         string(SessionSchemaExpiresAtField),
			LogicalField: string(SessionSchemaExpiresAtField),
			Type:         ColumnTypeTime,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "expires_at",
			},
		},
		{
			Name:         string(SessionSchemaLastAccessField),
			LogicalField: string(SessionSchemaLastAccessField),
			Type:         ColumnTypeTime,
			IsNullable:   false,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "last_access",
			},
		},
		{
			Name:         string(SessionSchemaMetadataField),
			LogicalField: string(SessionSchemaMetadataField),
			Type:         ColumnTypeMapStringAny,
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": "metadata",
			},
		},
	}
}
