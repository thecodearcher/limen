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

// ShouldRefresh checks if the session should be refreshed
func (s *Session) ShouldRefresh(refreshInterval time.Duration) bool {
	if refreshInterval == 0 {
		return false
	}
	// Slide-in window: refresh if we're within refreshInterval before expiration
	return time.Now().After(s.ExpiresAt.Add(-refreshInterval))
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

func (s *SessionSchema) GetTableName() TableName {
	if s.TableName == "" {
		return SessionSchemaTableName
	}
	return s.TableName
}

func (s *SessionSchema) GetSoftDeleteField() string {
	// sessions should not have a soft delete field
	return ""
}

func (s *SessionSchema) GetIDField() string {
	return getFieldOrDefault(s.Fields.ID, SchemaIDField)
}

func (s *SessionSchema) GetUserIDField() string {
	return getFieldOrDefault(s.Fields.UserID, SessionSchemaUserIDField)
}

func (s *SessionSchema) GetTokenField() string {
	return getFieldOrDefault(s.Fields.Token, SessionSchemaTokenField)
}

func (s *SessionSchema) GetCreatedAtField() string {
	return getFieldOrDefault(s.Fields.CreatedAt, SessionSchemaCreatedAtField)
}

func (s *SessionSchema) GetExpiresAtField() string {
	return getFieldOrDefault(s.Fields.ExpiresAt, SessionSchemaExpiresAtField)
}

func (s *SessionSchema) GetLastAccessField() string {
	return getFieldOrDefault(s.Fields.LastAccess, SessionSchemaLastAccessField)
}

func (s *SessionSchema) GetMetadataField() string {
	return getFieldOrDefault(s.Fields.Metadata, SessionSchemaMetadataField)
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
