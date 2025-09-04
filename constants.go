package aegis

// PendingAction represents a pending action for a user after authentication
type PendingAction string

// defaults for pending actions
const (
	PendingActionEmailVerification     PendingAction = "email_verification"
	PendingActionTwoFactorVerification PendingAction = "two_factor_verification"
)
