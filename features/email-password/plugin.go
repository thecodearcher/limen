// Package emailpassword provides email/password authentication for the aegis library.
package emailpassword

import (
	"context"
	"fmt"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/internal/database"
)

type emailPasswordFeature struct {
	core       *aegis.AegisCore
	config     *Config
	userSchema *aegis.UserSchema
}

// Config defines the configuration for the email password feature.
type Config struct {
	PasswordMinLength        int                                              // Minimum length of the password
	PasswordRequireUppercase bool                                             // Require uppercase letters in the password
	PasswordRequireNumbers   bool                                             // Require numbers in the password
	PasswordRequireSymbols   bool                                             // Require symbols in the password
	HashFn                   func(password string) (string, error)            // Custom function to hash the password
	CompareFn                func(password string, hash string) (bool, error) // Custom function to compare the password and the hash
	PasswordHasherConfig     PasswordHasherConfig                             // Custom Argon2id configuration for the password hasher
}

// New creates a new email-password feature instance with the given configuration.
func New(config *Config) *emailPasswordFeature {
	return &emailPasswordFeature{
		config: config,
	}
}

// DefaultConfig returns a new Config with the default values.
func DefaultConfig() *Config {
	return &Config{
		PasswordMinLength:        defaultMinPasswordLength,
		PasswordRequireUppercase: defaultPasswordRequireUppercase,
		PasswordRequireNumbers:   defaultPasswordRequireNumbers,
		PasswordRequireSymbols:   defaultPasswordRequireSymbols,
		PasswordHasherConfig:     DefaultPasswordHasherConfig,
	}
}

// Defaults returns a new EmailPasswordFeature with the default configuration.
func Defaults() *emailPasswordFeature {
	return &emailPasswordFeature{
		config: DefaultConfig(),
	}
}

func (p *emailPasswordFeature) Name() aegis.FeatureName {
	return aegis.FeatureEmailPassword
}

func (p *emailPasswordFeature) Initialize(core *aegis.AegisCore) error {
	p.core = core
	p.userSchema = &core.Schema.User

	if p.config == nil {
		return fmt.Errorf("config is required")
	}

	if p.config.PasswordMinLength < defaultMinPasswordLength {
		return fmt.Errorf("password min length must be at least 4")
	}

	return nil
}

func (p *emailPasswordFeature) SignInWithEmailAndPassword(ctx context.Context, email string, password string) (*aegis.User, error) {
	user, err := database.FindOne(ctx, p.core.DB, p.userSchema, []aegis.Where{aegis.Eq(p.userSchema.GetEmailField(), email)})
	if err != nil {
		return nil, ErrEmailNotFound
	}

	isValid, err := p.comparePassword(password, user.Password)
	if err != nil {
		return nil, err
	}

	if !isValid {
		return nil, ErrInvalidPassword
	}

	return user, nil
}

func (p *emailPasswordFeature) hashPassword(password string) (string, error) {
	if p.config.HashFn != nil {
		return p.config.HashFn(password)
	}

	return newPasswordHasher(p.config.PasswordHasherConfig).hashPassword([]byte(password))
}

func (p *emailPasswordFeature) comparePassword(password string, hash string) (bool, error) {
	if p.config.CompareFn != nil {
		return p.config.CompareFn(password, hash)
	}

	return newPasswordHasher(p.config.PasswordHasherConfig).verifyPassword([]byte(password), hash)
}
