package aegis

import "context"

// This file contains the interfaces for the features of the aegis library.
// and serves as a contract for the features of the library.
// Ensures that the features are implemented correctly in their respective modules.

type FeatureName string

const (
	FeatureEmailPassword FeatureName = "email-password"
)

type Feature interface {
	Name() FeatureName
	Initialize(core *AegisCore) error
}

type EmailPasswordFeature interface {
	SignInWithEmailAndPassword(ctx context.Context, email string, password string) (*User, error)
}
