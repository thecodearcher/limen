package credentialpassword

import (
	"github.com/thecodearcher/aegis"
)

// CredentialPasswordUserSchema extends UserSchema with username-specific functionality.
type CredentialPasswordUserSchema struct {
	*aegis.UserSchema
}

// GetUsernameField returns the resolved username field column name.
func (s *CredentialPasswordUserSchema) GetUsernameField() string {
	return s.GetField("username")
}
