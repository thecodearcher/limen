package usernamepassword

import (
	"github.com/thecodearcher/aegis"
)

// UsernamePasswordUserSchema extends UserSchema with username-specific functionality.
// It wraps the core UserSchema and provides methods for working with username fields.
type UsernamePasswordUserSchema struct {
	*aegis.UserSchema
}

type UserWithUsername struct {
	*aegis.User
	Username string
}

// NewUsernamePasswordUserSchema creates a new UsernamePasswordUserSchema that wraps the core UserSchema.
func NewUsernamePasswordUserSchema(schema *aegis.SchemaConfig) *UsernamePasswordUserSchema {
	return &UsernamePasswordUserSchema{
		UserSchema: schema.User,
	}
}

// GetUsernameField returns the resolved username field column name.
// It uses schema metadata to resolve the logical field name to the actual column name.
func (s *UsernamePasswordUserSchema) GetUsernameField() string {
	return s.GetField("username")
}

// FromStorage creates a User from database storage data.
// The username field is already in the raw data from the database, so no modification needed.
func (s *UsernamePasswordUserSchema) FromStorage(data map[string]any) aegis.Model {
	user := s.UserSchema.FromStorage(data).(*aegis.User)
	return &UserWithUsername{
		User:     user,
		Username: data[s.GetUsernameField()].(string),
	}
}

// ToStorage converts a User to storage format.
// It includes the username field from raw data using schema metadata.
func (s *UsernamePasswordUserSchema) ToStorage(data aegis.Model) map[string]any {
	result := s.UserSchema.ToStorage(data)
	// Add username field from raw data (read-only) if it exists
	username := data.Raw()[s.GetUsernameField()]
	if username != "" {
		result[s.GetUsernameField()] = username
	}

	return result
}
