package limen

import (
	"time"
)

type emailConfig struct {
	verification *emailVerificationConfig
}

type emailVerificationConfig struct {
	expiration    time.Duration
	sendEmail     func(email string, token string)
	generateToken func(*User) (string, error)
	enabled       bool
}

type EmailVerificationConfigOption func(*emailVerificationConfig)

type EmailConfigOption func(*emailConfig)

func NewDefaultEmailConfig(opts ...EmailConfigOption) *emailConfig {
	c := &emailConfig{
		verification: NewDefaultEmailVerification(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithEmailVerification(opts ...EmailVerificationConfigOption) EmailConfigOption {
	return func(c *emailConfig) {
		c.verification = NewDefaultEmailVerification(opts...)
	}
}

// NewDefaultEmailVerification creates a default email verification config.
func NewDefaultEmailVerification(opts ...EmailVerificationConfigOption) *emailVerificationConfig {
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
