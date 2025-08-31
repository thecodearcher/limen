package aegis

import (
	"time"
)

// this file contains the types for the aegis library
// NO FEATURES SHOULD BE ADDED TO THIS FILE those go in the feature.go file

// TokenGenerator defines the interface for JWT token generation and validation
type TokenGenerator interface {
	GenerateToken(claims map[string]interface{}, duration time.Duration) (string, error)
	GenerateAccessToken(user *User) (string, error)
	GenerateRefreshToken(claims map[string]any) (string, error)
	VerifyToken(token string) (map[string]any, error)
}

// AuthenticationResult extends the existing result to include JWT tokens
type AuthenticationResult struct {
	User         *User
	AccessToken  string
	RefreshToken *string
}
