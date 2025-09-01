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
	config     *config
	userSchema *aegis.UserSchema
}

// Config defines the configuration for the email password feature.
type config struct {
	passwordMinLength        int                                              // Minimum length of the password
	passwordRequireUppercase bool                                             // Require uppercase letters in the password
	passwordRequireNumbers   bool                                             // Require numbers in the password
	passwordRequireSymbols   bool                                             // Require symbols in the password
	hashFn                   func(password string) (string, error)            // Custom function to hash the password
	compareFn                func(password string, hash string) (bool, error) // Custom function to compare the password and the hash
	passwordHasherConfig     passwordHasherConfig                             // Custom Argon2id configuration for the password hasher
}

// New returns a new config with the default values.
// ConfigOptions can be provided to customize the configuration.
func New(opts ...ConfigOption) *emailPasswordFeature {
	config := &config{
		passwordMinLength:        defaultMinPasswordLength,
		passwordRequireUppercase: defaultPasswordRequireUppercase,
		passwordRequireNumbers:   defaultPasswordRequireNumbers,
		passwordRequireSymbols:   defaultPasswordRequireSymbols,
		passwordHasherConfig:     DefaultPasswordHasherConfig(),
	}

	for _, opt := range opts {
		opt(config)
	}

	return &emailPasswordFeature{
		config: config,
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

	if p.config.passwordMinLength < defaultMinPasswordLength {
		return fmt.Errorf("password min length must be at least 4")
	}

	return nil
}

func (p *emailPasswordFeature) SignInWithEmailAndPassword(ctx context.Context, email string, password string) (*aegis.AuthenticationResult, error) {
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
	fmt.Printf("user: %+v\n", user.Raw())
	accessToken, err := p.core.JWT.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &aegis.AuthenticationResult{
		User:        user,
		AccessToken: accessToken,
	}, nil
}

func (p *emailPasswordFeature) hashPassword(password string) (string, error) {
	if p.config.hashFn != nil {
		return p.config.hashFn(password)
	}

	return newPasswordHasher(p.config.passwordHasherConfig).hashPassword([]byte(password))
}

func (p *emailPasswordFeature) comparePassword(password string, hash string) (bool, error) {
	if p.config.compareFn != nil {
		return p.config.compareFn(password, hash)
	}

	return newPasswordHasher(p.config.passwordHasherConfig).verifyPassword([]byte(password), hash)
}
