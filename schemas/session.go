package schemas

import (
	"encoding/json"
	"time"
)

type Session struct {
	ID         string
	UserID     any
	Data       map[string]interface{}
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastAccess time.Time
	IPAddress  string
	UserAgent  string
	Metadata   map[string]interface{}
	raw        map[string]any
}

// Raw returns the session raw data as returned from the database
func (s Session) Raw() map[string]any {
	return s.raw
}

func (s Session) TableName() string {
	return string(SessionSchemaTableName)
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
	return time.Now().After(s.CreatedAt.Add(refreshInterval))
}

// Touch updates the last access time
func (s *Session) Touch() {
	s.LastAccess = time.Now()
}

type SessionSchema struct {
	// name of the table in the database
	TableName TableName
	// field name for the soft delete field - if not set, the soft delete field will not be used
	SoftDeleteField SchemaField
	// A function to return a map of additional fields to be added to the schema when creating a record
	AdditionalFields AdditionalFieldsFunc
	// mapping of the session schema to the database columns
	Fields SessionFields
}

type SessionFields struct {
	ID         string
	UserID     string
	Data       string
	CreatedAt  string
	ExpiresAt  string
	LastAccess string
	IPAddress  string
	UserAgent  string
	CSRFToken  string
	Metadata   string
}

func (s *SessionSchema) GetTableName() TableName {
	if s.TableName == "" {
		return SessionSchemaTableName
	}
	return s.TableName
}

func (s *SessionSchema) GetSoftDeleteField() SchemaField {
	return s.SoftDeleteField
}

func (s *SessionSchema) GetIDField() string {
	return getFieldOrDefault(s.Fields.ID, SchemaIDField)
}

func (s *SessionSchema) GetUserIDField() string {
	return getFieldOrDefault(s.Fields.UserID, SessionSchemaUserIDField)
}

func (s *SessionSchema) GetDataField() string {
	return getFieldOrDefault(s.Fields.Data, SessionSchemaDataField)
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

func (s *SessionSchema) GetIPAddressField() string {
	return getFieldOrDefault(s.Fields.IPAddress, SessionSchemaIPAddressField)
}

func (s *SessionSchema) GetUserAgentField() string {
	return getFieldOrDefault(s.Fields.UserAgent, SessionSchemaUserAgentField)
}

func (s *SessionSchema) GetCSRFTokenField() string {
	return getFieldOrDefault(s.Fields.CSRFToken, SessionSchemaCSRFTokenField)
}

func (s *SessionSchema) GetMetadataField() string {
	return getFieldOrDefault(s.Fields.Metadata, SessionSchemaMetadataField)
}

func (s *SessionSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return s.AdditionalFields
}

func (s *SessionSchema) FromStorage(data map[string]any) *Session {
	session := &Session{
		ID:         data[s.GetIDField()].(string),
		UserID:     data[s.GetUserIDField()].(string),
		CreatedAt:  data[s.GetCreatedAtField()].(time.Time),
		ExpiresAt:  data[s.GetExpiresAtField()].(time.Time),
		LastAccess: data[s.GetLastAccessField()].(time.Time),
		raw:        data,
	}

	// Handle optional fields
	if ipAddr, ok := data[s.GetIPAddressField()].(string); ok {
		session.IPAddress = ipAddr
	}

	if userAgent, ok := data[s.GetUserAgentField()].(string); ok {
		session.UserAgent = userAgent
	}

	// Deserialize JSON fields
	if dataStr, ok := data[s.GetDataField()].(string); ok && dataStr != "" {
		var sessionData map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &sessionData); err == nil {
			session.Data = sessionData
		} else {
			session.Data = make(map[string]interface{})
		}
	} else {
		session.Data = make(map[string]interface{})
	}

	if metadataStr, ok := data[s.GetMetadataField()].(string); ok && metadataStr != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
			session.Metadata = metadata
		} else {
			session.Metadata = make(map[string]interface{})
		}
	} else {
		session.Metadata = make(map[string]interface{})
	}

	return session
}

func (s *SessionSchema) ToStorage(data *Session) map[string]any {
	result := map[string]any{
		s.GetUserIDField():     data.UserID,
		s.GetCreatedAtField():  data.CreatedAt,
		s.GetExpiresAtField():  data.ExpiresAt,
		s.GetLastAccessField(): data.LastAccess,
	}

	// Handle optional fields
	if data.IPAddress != "" {
		result[s.GetIPAddressField()] = data.IPAddress
	}

	if data.UserAgent != "" {
		result[s.GetUserAgentField()] = data.UserAgent
	}

	// Serialize JSON fields
	if data.Data != nil {
		if dataJSON, err := json.Marshal(data.Data); err == nil {
			result[s.GetDataField()] = string(dataJSON)
		}
	}

	if data.Metadata != nil {
		if metadataJSON, err := json.Marshal(data.Metadata); err == nil {
			result[s.GetMetadataField()] = string(metadataJSON)
		}
	}

	return result
}
