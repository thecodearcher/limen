package limen

import (
	"time"
)

type emailConfig struct {
	verification *emailVerificationConfig
}

const (
	defaultEmailVerificationChallengeTTL        = 30 * time.Minute
	defaultEmailVerificationChallengeCookieName = "limen_email_verify"
)

type emailVerificationConfig struct {
	expiration    time.Duration
	sendEmail     func(email string, token string)
	generateToken func(*User) (string, error)
	enabled       bool

	challengeEnabled            bool
	challengeTTL                time.Duration
	challengeCookieName         string
	requestByEmailEnabled       bool
	autoSignInAfterVerification bool
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
		expiration:          24 * time.Hour,
		enabled:             true,
		challengeEnabled:    true,
		challengeTTL:        defaultEmailVerificationChallengeTTL,
		challengeCookieName: defaultEmailVerificationChallengeCookieName,
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

// WithEmailVerificationChallenge enables or disables the waiting-room cookie
// issued when sign-up or sign-in withholds a session until the email is verified.
// The cookie is on by default.
func WithEmailVerificationChallenge(enabled bool) EmailVerificationConfigOption {
	return func(c *emailVerificationConfig) {
		c.challengeEnabled = enabled
	}
}

// WithVerificationChallengeTTL sets how long the waiting-room cookie stays valid.
func WithVerificationChallengeTTL(d time.Duration) EmailVerificationConfigOption {
	return func(c *emailVerificationConfig) {
		c.challengeTTL = d
	}
}

// WithEmailVerificationRequestByEmail allows POST /email-verifications to accept
// a bare email with no session or challenge cookie.
func WithEmailVerificationRequestByEmail() EmailVerificationConfigOption {
	return func(c *emailVerificationConfig) {
		c.requestByEmailEnabled = true
	}
}

// WithAutoSignInAfterVerification creates a session after a successful verify.
func WithAutoSignInAfterVerification() EmailVerificationConfigOption {
	return func(c *emailVerificationConfig) {
		c.autoSignInAfterVerification = true
	}
}
