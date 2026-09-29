package limen

import (
	"encoding/json"
	"fmt"
	"time"
)

type Session struct {
	ID         any            `json:"id,omitempty"`
	Token      string         `json:"token"`
	UserID     any            `json:"user_id"`
	CreatedAt  time.Time      `json:"created_at"`
	ExpiresAt  time.Time      `json:"expires_at"`
	LastAccess time.Time      `json:"last_access"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	raw        map[string]any
}

// Raw returns the session raw data as returned from storage
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
	now := time.Now()

	return now.After(nextExtensionAt) || now.Equal(nextExtensionAt)
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

// SessionData returns the value the session holds for the logical field.
func (s *SessionSchema) SessionData(session *Session, field SchemaField) (any, error) {
	column := s.GetField(field)
	if column == "" {
		return nil, fmt.Errorf("%w: %q", ErrUnknownSessionField, field)
	}
	return session.Raw()[column], nil
}

// sessionColumns resolves a field-keyed session payload to database columns,
// rejecting fields not registered on the sessions schema.
func (s *SessionSchema) sessionColumns(payload map[SchemaField]any) (map[string]any, error) {
	columns := make(map[string]any, len(payload))
	for field, value := range payload {
		column := s.GetField(field)
		if column == "" {
			return nil, fmt.Errorf("%w: %q", ErrUnknownSessionField, field)
		}
		columns[column] = value
	}
	return columns, nil
}

func (s *SessionSchema) GetUserIDField() string {
	return s.GetField(SessionSchemaUserIDField)
}

func (s *SessionSchema) GetTokenField() string {
	return s.GetField(SessionSchemaTokenField)
}

func (s *SessionSchema) GetCreatedAtField() string {
	return s.GetField(SessionSchemaCreatedAtField)
}

func (s *SessionSchema) GetExpiresAtField() string {
	return s.GetField(SessionSchemaExpiresAtField)
}

func (s *SessionSchema) GetLastAccessField() string {
	return s.GetField(SessionSchemaLastAccessField)
}

func (s *SessionSchema) GetMetadataField() string {
	return s.GetField(SessionSchemaMetadataField)
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
	fields := s.storageFields(data.(*Session))
	result := make(map[string]any, len(fields))
	for field, value := range fields {
		result[s.GetField(field)] = value
	}
	return result
}

func (s *SessionSchema) storageFields(session *Session) map[SchemaField]any {
	fields := map[SchemaField]any{
		SessionSchemaTokenField:      session.Token,
		SessionSchemaUserIDField:     session.UserID,
		SessionSchemaCreatedAtField:  session.CreatedAt,
		SessionSchemaExpiresAtField:  session.ExpiresAt,
		SessionSchemaLastAccessField: session.LastAccess,
	}

	if session.Metadata != nil {
		if metadataJSON, err := json.Marshal(session.Metadata); err == nil {
			fields[SessionSchemaMetadataField] = string(metadataJSON)
		}
	}

	return fields
}
