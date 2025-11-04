package aegis

// PendingAction represents a pending action for a user after authentication
type PendingAction string

// defaults for pending actions
const (
	PendingActionEmailVerification     PendingAction = "email_verification"
	PendingActionTwoFactorVerification PendingAction = "two_factor_verification"
)

type JWTAlgorithm string

// jwt algorithms
const (
	JWTAlgorithmHS256 JWTAlgorithm = "HS256"
	JWTAlgorithmHS384 JWTAlgorithm = "HS384"
	JWTAlgorithmHS512 JWTAlgorithm = "HS512"
	JWTAlgorithmRS256 JWTAlgorithm = "RS256"
	JWTAlgorithmRS384 JWTAlgorithm = "RS384"
	JWTAlgorithmRS512 JWTAlgorithm = "RS512"
	JWTAlgorithmES256 JWTAlgorithm = "ES256"
	JWTAlgorithmES384 JWTAlgorithm = "ES384"
	JWTAlgorithmES512 JWTAlgorithm = "ES512"
)

// SessionStrategyType represents the type of session strategy
type SessionStrategyType string

// Session strategy types
const (
	SessionStrategyServerSide SessionStrategyType = "server_side"
	SessionStrategyJWT        SessionStrategyType = "jwt"
	SessionStrategyHybrid     SessionStrategyType = "hybrid"
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
