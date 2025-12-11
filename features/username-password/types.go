package usernamepassword

import (
	"regexp"

	"github.com/thecodearcher/aegis"
)

type ConfigOption func(*config)

type config struct {
	usernameMinLength      int
	usernameMaxLength      int
	usernameValidationRegex *regexp.Regexp
	emailPassword          aegis.EmailPasswordFeature // Reference to email-password plugin
}

// New returns a new config with the default values.
// ConfigOptions can be provided to customize the configuration.
func New(opts ...ConfigOption) *usernamePasswordFeature {
	config := &config{
		usernameMinLength: defaultMinUsernameLength,
		usernameMaxLength: defaultMaxUsernameLength,
		usernameValidationRegex: regexp.MustCompile(`^[a-zA-Z0-9_-]+$`), // alphanumeric, underscore, hyphen
	}

	for _, opt := range opts {
		opt(config)
	}

	return &usernamePasswordFeature{
		config: config,
	}
}

// WithUsernameMinLength sets the minimum length of the username
func WithUsernameMinLength(minLength int) ConfigOption {
	return func(c *config) {
		c.usernameMinLength = minLength
	}
}

// WithUsernameMaxLength sets the maximum length of the username
func WithUsernameMaxLength(maxLength int) ConfigOption {
	return func(c *config) {
		c.usernameMaxLength = maxLength
	}
}

// WithUsernameValidationRegex sets a custom regex pattern for username validation
func WithUsernameValidationRegex(pattern *regexp.Regexp) ConfigOption {
	return func(c *config) {
		c.usernameValidationRegex = pattern
	}
}

// SetEmailPasswordFeature sets the email-password feature reference.
// This is called after all features are initialized.
func (p *usernamePasswordFeature) SetEmailPasswordFeature(emailPassword aegis.EmailPasswordFeature) {
	p.config.emailPassword = emailPassword
}
