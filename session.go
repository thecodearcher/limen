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
	// name of the table in the database
	TableName TableName
	// A function to return a map of additional fields to be added to the schema when creating a record
	AdditionalFields AdditionalFieldsFunc
	// mapping of the session schema to the database columns
	Fields SessionFields
}

type SessionFields struct {
	ID         string
	Token      string
	UserID     string
	CreatedAt  string
	ExpiresAt  string
	LastAccess string
	Metadata   string
}

type SessionSchemaOption func(*SessionSchema)

// NewDefaultSessionSchema creates a new SessionSchema with default values
func NewDefaultSessionSchema(opts ...SessionSchemaOption) *SessionSchema {
	schema := &SessionSchema{
		TableName: SessionSchemaTableName,
		Fields: SessionFields{
			ID:         string(SchemaIDField),
			Token:      string(SessionSchemaTokenField),
			UserID:     string(SessionSchemaUserIDField),
			CreatedAt:  string(SessionSchemaCreatedAtField),
			ExpiresAt:  string(SessionSchemaExpiresAtField),
			LastAccess: string(SessionSchemaLastAccessField),
			Metadata:   string(SessionSchemaMetadataField),
		},
	}

	for _, opt := range opts {
		opt(schema)
	}

	return schema
}

func (s *SessionSchema) GetTableName() TableName {
	return s.TableName
}

func (s *SessionSchema) GetSoftDeleteField() string {
	// sessions should not have a soft delete field
	return ""
}

func (s *SessionSchema) GetIDField() string {
	return s.Fields.ID
}

func (s *SessionSchema) GetUserIDField() string {
	return s.Fields.UserID
}

func (s *SessionSchema) GetTokenField() string {
	return s.Fields.Token
}

func (s *SessionSchema) GetCreatedAtField() string {
	return s.Fields.CreatedAt
}

func (s *SessionSchema) GetExpiresAtField() string {
	return s.Fields.ExpiresAt
}

func (s *SessionSchema) GetLastAccessField() string {
	return s.Fields.LastAccess
}

func (s *SessionSchema) GetMetadataField() string {
	return s.Fields.Metadata
}

func (s *SessionSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return s.AdditionalFields
}

func (s *SessionSchema) FromStorage(data map[string]any) *Session {
	return &Session{
		ID:         data[s.GetIDField()],
		Token:      data[s.GetTokenField()].(string),
		UserID:     data[s.GetUserIDField()],
		CreatedAt:  data[s.GetCreatedAtField()].(time.Time),
		ExpiresAt:  data[s.GetExpiresAtField()].(time.Time),
		LastAccess: data[s.GetLastAccessField()].(time.Time),
		raw:        data,
	}
}

func (s *SessionSchema) ToStorage(data *Session) map[string]any {
	result := map[string]any{
		s.GetTokenField():      data.Token,
		s.GetUserIDField():     data.UserID,
		s.GetCreatedAtField():  data.CreatedAt,
		s.GetExpiresAtField():  data.ExpiresAt,
		s.GetLastAccessField(): data.LastAccess,
	}

	if data.Metadata != nil {
		if metadataJSON, err := json.Marshal(data.Metadata); err == nil {
			result[s.GetMetadataField()] = string(metadataJSON)
		}
	}

	return result
}

func WithSessionTableName(tableName TableName) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.TableName = tableName
	}
}

func WithSessionAdditionalFields(fn AdditionalFieldsFunc) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.AdditionalFields = fn
	}
}

func WithSessionFields(fields SessionFields) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.Fields = fields
	}
}

func WithSessionFieldID(fieldName string) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.Fields.ID = fieldName
	}
}

func WithSessionFieldToken(fieldName string) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.Fields.Token = fieldName
	}
}

func WithSessionFieldUserID(fieldName string) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.Fields.UserID = fieldName
	}
}

func WithSessionFieldCreatedAt(fieldName string) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.Fields.CreatedAt = fieldName
	}
}

func WithSessionFieldExpiresAt(fieldName string) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.Fields.ExpiresAt = fieldName
	}
}

func WithSessionFieldLastAccess(fieldName string) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.Fields.LastAccess = fieldName
	}
}

func WithSessionFieldMetadata(fieldName string) SessionSchemaOption {
	return func(s *SessionSchema) {
		s.Fields.Metadata = fieldName
	}
}

// Introspect implements SchemaIntrospector for SessionSchema
func (s *SessionSchema) Introspect() SchemaIntrospector {
	return NewIntrospector(
		s,
		s.TableName,
		func(schema *SessionSchema) []ColumnDefinition {
			return []ColumnDefinition{
				{
					Name:         schema.Fields.ID,
					LogicalField: string(SchemaIDField),
					Type:         ColumnTypeAny,
					IsNullable:   false,
					IsPrimaryKey: true,
					Tags: map[string]string{
						"json": "id",
					},
				},
				{
					Name:         schema.Fields.Token,
					LogicalField: string(SessionSchemaTokenField),
					Type:         ColumnTypeString,
					IsNullable:   false,
					IsPrimaryKey: false,
					Tags: map[string]string{
						"json": "token",
					},
				},
				{
					Name:         schema.Fields.UserID,
					LogicalField: string(SessionSchemaUserIDField),
					Type:         ColumnTypeAny,
					IsNullable:   false,
					IsPrimaryKey: false,
					Tags: map[string]string{
						"json": "user_id",
					},
				},
				{
					Name:         schema.Fields.CreatedAt,
					LogicalField: string(SessionSchemaCreatedAtField),
					Type:         ColumnTypeTime,
					IsNullable:   false,
					IsPrimaryKey: false,
					Tags: map[string]string{
						"json": "created_at",
					},
				},
				{
					Name:         schema.Fields.ExpiresAt,
					LogicalField: string(SessionSchemaExpiresAtField),
					Type:         ColumnTypeTime,
					IsNullable:   false,
					IsPrimaryKey: false,
					Tags: map[string]string{
						"json": "expires_at",
					},
				},
				{
					Name:         schema.Fields.LastAccess,
					LogicalField: string(SessionSchemaLastAccessField),
					Type:         ColumnTypeTime,
					IsNullable:   false,
					IsPrimaryKey: false,
					Tags: map[string]string{
						"json": "last_access",
					},
				},
				{
					Name:         schema.Fields.Metadata,
					LogicalField: string(SessionSchemaMetadataField),
					Type:         ColumnTypeMapStringAny,
					IsNullable:   true,
					IsPrimaryKey: false,
					Tags: map[string]string{
						"json": "metadata",
					},
				},
			}
		},
		func(schema *SessionSchema) []IndexDefinition {
			return []IndexDefinition{
				{
					Name:    "idx_sessions_token",
					Columns: []string{schema.Fields.Token},
					Unique:  true,
				},
				{
					Name:    "idx_sessions_user_id",
					Columns: []string{schema.Fields.UserID},
					Unique:  false,
				},
			}
		},
		func(schema *SessionSchema) []ForeignKeyDefinition {
			return []ForeignKeyDefinition{
				{
					Name:             "fk_sessions_user_id",
					Column:           schema.Fields.UserID,
					ReferencedSchema: string(CoreSchemaUsers),
					ReferencedField:  string(SchemaIDField),
					OnDelete:         FKActionCascade,
					OnUpdate:         FKActionCascade,
				},
			}
		},
		func(schema *SessionSchema) *CoreSchemaName {
			return nil
		},
	)
}
