package passkey

import "github.com/go-webauthn/webauthn/protocol"

// AuthenticatorSelection narrows which authenticators may create a passkey, and
// how they must behave while doing it. It applies to registration only.
type AuthenticatorSelection = protocol.AuthenticatorSelection

// UserVerification says how hard an authenticator has to work to confirm who the
// user is, rather than just that somebody is standing there.
type UserVerification = protocol.UserVerificationRequirement

const (
	// UserVerificationRequired turns away a passkey the authenticator did not check
	// a fingerprint, face, or PIN for.
	UserVerificationRequired = protocol.VerificationRequired
	// UserVerificationPreferred takes the check when the authenticator can do it and
	// carries on when it cannot.
	UserVerificationPreferred = protocol.VerificationPreferred
	// UserVerificationDiscouraged skips the check, leaving a passkey that only proves
	// the user has the device.
	UserVerificationDiscouraged = protocol.VerificationDiscouraged
)

// ResidentKey says whether the authenticator should keep the passkey on itself,
// which is what lets someone sign in without typing an email first.
type ResidentKey = protocol.ResidentKeyRequirement

const (
	ResidentKeyDiscouraged = protocol.ResidentKeyRequirementDiscouraged
	ResidentKeyPreferred   = protocol.ResidentKeyRequirementPreferred
	ResidentKeyRequired    = protocol.ResidentKeyRequirementRequired
)

// AuthenticatorAttachment picks between passkeys built into the device the user is
// on and roaming ones such as security keys.
type AuthenticatorAttachment = protocol.AuthenticatorAttachment

const (
	AuthenticatorPlatform      = protocol.Platform
	AuthenticatorCrossPlatform = protocol.CrossPlatform
)

type config struct {
	rpID           string
	rpName         string
	allowedOrigins []string

	authenticatorSelection AuthenticatorSelection
	challengeCookieName    string
}

type ConfigOption func(*config)

// WithRPID sets the domain passkeys are tied to. Defaults to the host of your
// base URL.
//
// Point it at the parent domain (e.g. "example.com" while auth runs on
// "auth.example.com") so one passkey works across all your subdomains.
func WithRPID(rpID string) ConfigOption {
	return func(c *config) {
		c.rpID = rpID
	}
}

// WithRPName sets the name a device shows the user when it asks them to create
// or use a passkey, usually your product name.
func WithRPName(rpName string) ConfigOption {
	return func(c *config) {
		c.rpName = rpName
	}
}

// WithAllowedOrigins sets which origins passkeys may be used from. Defaults to
// your base URL's origin. Each one has to sit on the passkey domain or under it,
// so "https://app.example.com" is fine for "example.com".
func WithAllowedOrigins(origins ...string) ConfigOption {
	return func(c *config) {
		c.allowedOrigins = origins
	}
}

// WithAuthenticatorSelection narrows which authenticators may create a passkey.
// Defaults to verifying the user and preferring a passkey the authenticator keeps,
// so the passkey stands on its own and can sign someone in without an email first.
func WithAuthenticatorSelection(selection AuthenticatorSelection) ConfigOption {
	return func(c *config) {
		c.authenticatorSelection = selection
	}
}

// WithChallengeCookieName sets the name of the cookie that stores the challenge
// for the passkey authentication. Defaults to "limen-passkey".
func WithChallengeCookieName(cookieName string) ConfigOption {
	return func(c *config) {
		c.challengeCookieName = cookieName
	}
}
