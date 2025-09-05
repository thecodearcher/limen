package aegis

import (
	"time"

	"github.com/thecodearcher/aegis/schemas"
)

// this file contains the types for the aegis library
// NO FEATURES SHOULD BE ADDED TO THIS FILE those go in the feature.go file

// TokenGenerator defines the interface for JWT token generation and validation
type TokenGenerator interface {
	GenerateToken(claims map[string]interface{}, duration time.Duration) (string, error)
	GenerateAccessToken(user *schemas.User) (string, error)
	GenerateRefreshToken(claims map[string]any) (string, error)
	VerifyToken(token string) (map[string]any, error)
}

// AuthenticationResult represents the result of an authentication process and includes additional
type AuthenticationResult struct {
	// User represents the authenticated user
	User *User
	// Pending actions to be completed by the user before they can access the application e.g: two-factor authentication, email verification etc.
	PendingActions []PendingAction
}

// alias for the user model
type User = schemas.User
type SchemaConfig = schemas.Config

type UserSchema = schemas.UserSchema
type UserFields = schemas.UserFields
type VerificationSchema = schemas.VerificationSchema
type VerificationFields = schemas.VerificationFields
