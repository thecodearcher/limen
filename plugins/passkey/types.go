package passkey

import (
	"time"

	"github.com/go-webauthn/webauthn/protocol"
)

type config struct {
	rpID                 string
	rpName               string
	origins              []string
	challengeExpiration  time.Duration
	challengeCookieName  string
	requireSessionToReg  bool
	requireUserPresence  bool
	requireUserVerified  bool
	authenticatorSelect  *protocol.AuthenticatorSelection
}

// ConfigOption configures the passkey plugin.
type ConfigOption func(*config)

// WithRPID sets the WebAuthn Relying Party ID. Defaults to the hostname
// parsed from the Limen BaseURL (e.g., "localhost" or "example.com").
// The RPID must match the effective domain of every origin that uses
// passkeys for the application.
func WithRPID(id string) ConfigOption {
	return func(c *config) { c.rpID = id }
}

// WithRPName sets the human-readable Relying Party name shown to the user
// by their authenticator. Defaults to "Limen App".
func WithRPName(name string) ConfigOption {
	return func(c *config) { c.rpName = name }
}

// WithOrigins sets the list of allowed origins (e.g.,
// "https://app.example.com"). If unset, the Limen BaseURL is used.
func WithOrigins(origins ...string) ConfigOption {
	return func(c *config) { c.origins = append([]string{}, origins...) }
}

// WithChallengeExpiration sets how long a WebAuthn challenge remains
// valid. Defaults to 5 minutes — the browser will typically time out
// well before this.
func WithChallengeExpiration(d time.Duration) ConfigOption {
	return func(c *config) { c.challengeExpiration = d }
}

// WithChallengeCookieName overrides the signed cookie name used to
// correlate an in-flight ceremony with its server-side challenge state.
func WithChallengeCookieName(name string) ConfigOption {
	return func(c *config) { c.challengeCookieName = name }
}

// WithRequireSessionForRegistration controls whether a user must be
// signed in to register a new passkey. Defaults to true. When false, the
// caller must provide credentials via another means (e.g. a magic-link
// signed cookie) before invoking the registration endpoints.
func WithRequireSessionForRegistration(require bool) ConfigOption {
	return func(c *config) { c.requireSessionToReg = require }
}

// WithRequireUserVerification requires that the authenticator perform
// user verification (PIN, biometric, etc.) before issuing a credential
// or assertion. Defaults to false, matching better-auth.
func WithRequireUserVerification(require bool) ConfigOption {
	return func(c *config) { c.requireUserVerified = require }
}

// WithAuthenticatorSelection overrides the default authenticator
// selection criteria sent in registration options.
func WithAuthenticatorSelection(sel *protocol.AuthenticatorSelection) ConfigOption {
	return func(c *config) { c.authenticatorSelect = sel }
}

// Passkey is the public representation of a stored credential returned
// by management endpoints. The PublicKey is omitted from JSON since it
// is implementation detail that callers should not need.
type Passkey struct {
	ID           any       `json:"id"`
	UserID       any       `json:"user_id"`
	Name         string    `json:"name,omitempty"`
	CredentialID string    `json:"credential_id"`
	DeviceType   string    `json:"device_type"`
	BackedUp     bool      `json:"backed_up"`
	Transports   string    `json:"transports,omitempty"`
	AAGUID       string    `json:"aaguid,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
