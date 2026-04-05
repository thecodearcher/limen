package limen

import (
	"time"
)

type emailVerificationConfig struct {
	expiration    time.Duration
	sendEmail     func(email string, token string)
	generateToken func(*User) (string, error)
	enabled       bool
}

type EmailVerificationConfigOption func(*emailVerificationConfig)

// DefaultEmailVerification creates a default email verification config.
func DefaultEmailVerification(opts ...EmailVerificationConfigOption) *emailVerificationConfig {
	c := &emailVerificationConfig{
		expiration: 24 * time.Hour,
		enabled:    true,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithDisableEmailVerification disables email verification.
func WithDisableEmailVerification() EmailVerificationConfigOption {
	return func(c *emailVerificationConfig) {
		c.enabled = false
	}
}

// WithEmailVerificationExpiration sets the token expiration duration.
func WithEmailVerificationExpiration(d time.Duration) EmailVerificationConfigOption {
	return func(c *emailVerificationConfig) {
		c.expiration = d
	}
}

// WithSendEmailVerificationMail sets the callback invoked to deliver the
// verification email.
func WithSendEmailVerificationMail(fn func(email string, token string)) EmailVerificationConfigOption {
	return func(c *emailVerificationConfig) {
		c.sendEmail = fn
	}
}

// WithEmailVerificationTokenGenerator overrides the default random token
// generator (e.g. to produce TOTP codes or signed JWTs).
func WithEmailVerificationTokenGenerator(fn func(*User) (string, error)) EmailVerificationConfigOption {
	return func(c *emailVerificationConfig) {
		c.generateToken = fn
	}
}
