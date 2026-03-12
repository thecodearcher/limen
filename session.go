package aegis

import (
	"encoding/json"
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
