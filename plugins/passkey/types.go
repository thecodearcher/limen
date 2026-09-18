package passkey

import (
	"context"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/thecodearcher/limen"
)

// AuthenticatorSelection narrows which authenticators may create a passkey, and
// how they must behave while doing it. It applies to registration only.
type AuthenticatorSelection = protocol.AuthenticatorSelection

// AuthenticationExtensions is the WebAuthn AuthenticationExtensionsClientInputs
// bag passed through to begin-registration / begin-authentication options.
type AuthenticationExtensions = protocol.AuthenticationExtensions

// CredentialCreation is the WebAuthn registration options payload.
type CredentialCreation = protocol.CredentialCreation

// CredentialAssertion is the WebAuthn authentication options payload.
type CredentialAssertion = protocol.CredentialAssertion

// ExtensionsResolver builds extension inputs per request (e.g. PRF eval salts),
// for callers whose extensions depend on who is asking.
type ExtensionsResolver func(ctx context.Context, r *http.Request) (AuthenticationExtensions, error)

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

// RegistrationIntent is the validated signup intent returned by PrepareRegistration.
type RegistrationIntent struct {
	Email       string
	Name        string // WebAuthn name; default Email
	DisplayName string // default Name
	// ID is an optional app-minted public id or opaque primary key. When set, it
	// becomes the reserved WebAuthn userHandle.
	ID             string
	AdditionalData map[string]any
}

// PrepareRegistrationFunc validates opaque registration context at begin.
// It must not create a user.
type PrepareRegistrationFunc func(ctx context.Context, r *http.Request, registrationContext string) (*RegistrationIntent, error)

// CreateRegistrationUserFunc runs after attestation verifies. Create the account
// in your store with reservedHandle as the user's permanent id, and return it.
type CreateRegistrationUserFunc func(ctx context.Context, r *http.Request, intent *RegistrationIntent, reservedHandle string) (*limen.User, error)

type config struct {
	rpID           string
	rpName         string
	allowedOrigins []string

	authenticatorSelection AuthenticatorSelection
	challengeCookieName    string

	requireSession              bool
	allowInternalUserIDAsHandle bool
	prepareRegistration         PrepareRegistrationFunc
	createRegistrationUser      CreateRegistrationUserFunc

	// Either an AuthenticationExtensions or an ExtensionsResolver.
	registrationExtensions   any
	authenticationExtensions any
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

// WithRequireSession controls whether registration requires a signed-in session.
// Defaults to true. When false, passkey-first signup is enabled and
// PrepareRegistration and CreateRegistrationUser are required.
func WithRequireSession(require bool) ConfigOption {
	return func(c *config) {
		c.requireSession = require
	}
}

// WithAllowInternalUserIDAsHandle opts into using your user primary key as the
// WebAuthn userHandle when public IDs are off. Use this only if those IDs are
// already opaque (e.g. UUID via an ID generator), not auto-increment. Defaults to false.
func WithAllowInternalUserIDAsHandle(allow bool) ConfigOption {
	return func(c *config) {
		c.allowInternalUserIDAsHandle = allow
	}
}

// WithPrepareRegistration sets the callback that validates the client's opaque
// registration context and returns a RegistrationIntent. Do not create a user
// here. Required when requireSession is false.
func WithPrepareRegistration(fn PrepareRegistrationFunc) ConfigOption {
	return func(c *config) {
		c.prepareRegistration = fn
	}
}

// WithCreateRegistrationUser sets the callback that creates the account after
// attestation verifies. Persist reservedHandle as the user's permanent id and
// return the created user. Required when requireSession is false.
func WithCreateRegistrationUser(fn CreateRegistrationUserFunc) ConfigOption {
	return func(c *config) {
		c.createRegistrationUser = fn
	}
}

// WithRegistrationExtensions sets static WebAuthn extension inputs for
// registration ceremonies (e.g. credProps, largeBlob support, PRF probe).
func WithRegistrationExtensions(ext AuthenticationExtensions) ConfigOption {
	return func(c *config) {
		c.registrationExtensions = ext
	}
}

// WithAuthenticationExtensions sets static WebAuthn extension inputs for
// authentication ceremonies (e.g. PRF eval, largeBlob read/write).
func WithAuthenticationExtensions(ext AuthenticationExtensions) ConfigOption {
	return func(c *config) {
		c.authenticationExtensions = ext
	}
}

// WithRegistrationExtensionsResolver sets a per-request resolver for
// registration extension inputs.
func WithRegistrationExtensionsResolver(resolver ExtensionsResolver) ConfigOption {
	return func(c *config) {
		c.registrationExtensions = resolver
	}
}

// WithAuthenticationExtensionsResolver sets a per-request resolver for
// authentication extension inputs.
func WithAuthenticationExtensionsResolver(resolver ExtensionsResolver) ConfigOption {
	return func(c *config) {
		c.authenticationExtensions = resolver
	}
}
