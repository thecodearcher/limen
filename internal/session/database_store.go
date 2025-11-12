package session

import (
	"context"
	"maps"
	"time"

	"github.com/thecodearcher/aegis"
)

// DatabaseSessionStore implements SessionStore using a database adapter
type DatabaseSessionStore struct {
	core   *aegis.AegisCore
	schema *aegis.SessionSchema
}

// NewDatabaseSessionStore creates a new database-backed session store
func NewDatabaseSessionStore(core *aegis.AegisCore) *DatabaseSessionStore {
	return &DatabaseSessionStore{
		core:   core,
		schema: &core.Schema.Session,
	}
}

// Create creates a new session with the given ID and data
func (s *DatabaseSessionStore) Create(ctx context.Context, session *aegis.Session) error {
	payload := make(map[string]any)
	additionalFieldsContext := aegis.NewAdditionalFieldsContext(nil, nil)

	// Copy global additional fields first
	if s.core.Schema.AdditionalFields != nil {
		additionalFields, err := s.core.Schema.AdditionalFields(additionalFieldsContext)
		if err != nil {
			return err
		}
		maps.Copy(payload, additionalFields)
	}

	// Copy schema additional fields
	if s.schema.GetAdditionalFields() != nil {
		additionalFields, err := s.schema.GetAdditionalFields()(additionalFieldsContext)
		if err != nil {
			return err
		}
		maps.Copy(payload, additionalFields)
	}

	// Copy session data using ToStorage
	maps.Copy(payload, s.schema.ToStorage(session))

	// Include ID field explicitly since ToStorage doesn't include it
	payload[s.schema.GetIDField()] = session.ID

	// Convert empty strings to nil
	for key, value := range payload {
		if value == "" {
			payload[key] = nil
		}
	}

	return s.core.DB.Create(ctx, s.schema.GetTableName(), payload)
}

// Get retrieves a session by ID
func (s *DatabaseSessionStore) Get(ctx context.Context, sessionID string) (*aegis.Session, error) {
	conditions := []aegis.Where{
		aegis.Eq(s.schema.GetIDField(), sessionID),
	}

	// Apply soft delete filter if configured
	if s.schema.GetSoftDeleteField() != "" {
		conditions = append(conditions, aegis.IsNull(string(s.schema.GetSoftDeleteField())))
	}

	result, err := s.core.DB.FindOne(ctx, s.schema.GetTableName(), conditions, nil)
	if err != nil {
		return nil, err
	}

	return s.schema.FromStorage(result), nil
}

// Update updates an existing session
func (s *DatabaseSessionStore) Update(ctx context.Context, session *aegis.Session) error {
	conditions := []aegis.Where{
		aegis.Eq(s.schema.GetIDField(), session.ID),
	}

	payload := s.schema.ToStorage(session)

	// Remove empty values to avoid accidental NULL updates
	for key, value := range payload {
		if value == "" || value == 0 || value == nil {
			delete(payload, key)
		}
	}

	return s.core.DB.Update(ctx, s.schema.GetTableName(), conditions, payload)
}

// Delete removes a session by ID
func (s *DatabaseSessionStore) Delete(ctx context.Context, sessionID string) error {
	conditions := []aegis.Where{
		aegis.Eq(s.schema.GetIDField(), sessionID),
	}

	return s.core.DB.Delete(ctx, s.schema.GetTableName(), conditions)
}

// DeleteByUserID removes all sessions for a specific user
func (s *DatabaseSessionStore) DeleteByUserID(ctx context.Context, userID string) error {
	conditions := []aegis.Where{
		aegis.Eq(s.schema.GetUserIDField(), userID),
	}

	return s.core.DB.Delete(ctx, s.schema.GetTableName(), conditions)
}

// Cleanup removes expired sessions
func (s *DatabaseSessionStore) Cleanup(ctx context.Context) error {
	now := time.Now()
	conditions := []aegis.Where{
		aegis.Lt(s.schema.GetExpiresAtField(), now),
	}

	return s.core.DB.Delete(ctx, s.schema.GetTableName(), conditions)
}

// Exists checks if a session exists
func (s *DatabaseSessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	conditions := []aegis.Where{
		aegis.Eq(s.schema.GetIDField(), sessionID),
	}

	// Apply soft delete filter if configured
	if s.schema.GetSoftDeleteField() != "" {
		conditions = append(conditions, aegis.IsNull(string(s.schema.GetSoftDeleteField())))
	}

	return s.core.DB.Exists(ctx, s.schema.GetTableName(), conditions)
}

// Count returns the number of active sessions for a user
func (s *DatabaseSessionStore) Count(ctx context.Context, userID string) (int, error) {
	conditions := []aegis.Where{
		aegis.Eq(s.schema.GetUserIDField(), userID),
	}

	count, err := s.core.DB.Count(ctx, s.schema.GetTableName(), conditions)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}
