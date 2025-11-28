package aegis

// PendingAction represents a pending action for a user after authentication
type PendingAction string

// defaults for pending actions
const (
	PendingActionEmailVerification     PendingAction = "email_verification"
	PendingActionTwoFactorVerification PendingAction = "two_factor_verification"
)

// SessionStrategyType represents the type of session strategy
type SessionStrategyType string

// Session strategy types
const (
	SessionStrategyOpaqueToken SessionStrategyType = "opaque_token"
)

type EnvelopeMode int

const (
	EnvelopeOff EnvelopeMode = iota
	EnvelopeWrapSuccess
	EnvelopeAlways
)

type SessionStoreType string

const (
	SessionStoreTypeMemory   SessionStoreType = "in_memory"
	SessionStoreTypeDatabase SessionStoreType = "database"
)

// TokenDeliveryMethod specifies how tokens should be delivered
type TokenDeliveryMethod string

const (
	// TokenDeliveryCookie delivers tokens via HttpOnly cookies
	TokenDeliveryCookie TokenDeliveryMethod = "cookie"
	// TokenDeliveryHeader delivers tokens in response headers
	TokenDeliveryHeader TokenDeliveryMethod = "header"
)
