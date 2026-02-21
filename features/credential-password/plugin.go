// Package credentialpassword provides credential(email/username) and password authentication for the aegis library.
package credentialpassword

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/thecodearcher/aegis"
)

type credentialPasswordFeature struct {
	core               *aegis.AegisCore
	config             *config
	userSchema         *aegis.UserSchema
	verificationSchema *aegis.VerificationSchema
	dbAction           *aegis.DatabaseActionHelper
}

// getUsernameField returns the resolved username field column name if username support is enabled.
func (p *credentialPasswordFeature) getUsernameField() string {
	if !p.config.enableUsername {
		return ""
	}
	return p.userSchema.GetField("username")
}

// Config defines the configuration for the credential password feature.
type config struct {
	passwordMinLength           int                                              // Minimum length of the password
	passwordRequireUppercase    bool                                             // Require uppercase letters in the password
	passwordRequireNumbers      bool                                             // Require numbers in the password
	passwordRequireSymbols      bool                                             // Require symbols in the password
	hashFn                      func(password string) (string, error)            // Custom function to hash the password
	compareFn                   func(password string, hash string) (bool, error) // Custom function to compare the password and the hash
	passwordHasherConfig        passwordHasherConfig                             // Custom Argon2id configuration for the password hasher
	requireEmailVerification    bool                                             // require email verification after sign up
	emailVerificationExpiration time.Duration                                    // Custom expiration duration for the email verification
	resetTokenExpiration        time.Duration                                    // Custom expiration duration for the reset token
	generateResetToken          func(*aegis.User) (string, error)                // custom function to generate the reset token e.g generating TOTP code
	autoSignInOnSignUp          bool                                             // auto sign in the user after sign up
	sendVerificationEmail       func(email string, token string)                 // function to send the email verification message
	sendPasswordResetEmail      func(email string, token string)                 // function to send the password reset message
	onPasswordResetSuccess      func(ctx context.Context, user *aegis.User)      // function to call when the password reset is successful
	enableUsername              bool                                             // enable username support (default: false)
	usernameMinLength           int                                              // Minimum length of the username
	usernameMaxLength           int                                              // Maximum length of the username
	usernameValidationRegex     *regexp.Regexp                                   // Custom regex pattern for username validation
	usernameRequiredOnSignup    bool                                             // require username during sign up
	usernameValidationFunc      func(username string) error                      // custom function to validate the username
}

// New returns a new config with the default values.
// ConfigOptions can be provided to customize the configuration.
func New(opts ...ConfigOption) *credentialPasswordFeature {
	config := &config{
		passwordMinLength:           defaultMinPasswordLength,
		passwordRequireUppercase:    defaultPasswordRequireUppercase,
		passwordRequireNumbers:      defaultPasswordRequireNumbers,
		passwordRequireSymbols:      defaultPasswordRequireSymbols,
		passwordHasherConfig:        DefaultPasswordHasherConfig(),
		resetTokenExpiration:        30 * time.Minute,
		autoSignInOnSignUp:          true,
		requireEmailVerification:    false,
		emailVerificationExpiration: 24 * time.Hour,
		enableUsername:              false,
		usernameMinLength:           defaultMinUsernameLength,
		usernameMaxLength:           defaultMaxUsernameLength,
		usernameValidationRegex:     regexp.MustCompile(`^[a-zA-Z0-9_-]+$`), // alphanumeric, underscore, hyphen
		usernameRequiredOnSignup:    false,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &credentialPasswordFeature{
		config: config,
	}
}

func (p *credentialPasswordFeature) Name() aegis.FeatureName {
	return aegis.FeatureCredentialPassword
}

func (p *credentialPasswordFeature) GetSchemas(schema *aegis.SchemaConfig) []aegis.SchemaIntrospector {
	if !p.config.enableUsername {
		return []aegis.SchemaIntrospector{}
	}

	userWithUsername := &CredentialPasswordUserSchema{
		UserSchema: schema.User,
	}
	extension := aegis.NewSchemaDefinitionForExtension(
		aegis.CoreSchemaUsers,
		userWithUsername,
		aegis.WithSchemaField("username", aegis.ColumnTypeString, aegis.WithNullable(true)),
		aegis.WithSchemaIndex("idx_users_username", []aegis.SchemaField{CredentialPasswordUserSchemaUsernameField}),
	)

	return []aegis.SchemaIntrospector{extension}
}

func (p *credentialPasswordFeature) Initialize(core *aegis.AegisCore) error {
	p.core = core
	p.userSchema = core.Schema.User
	p.dbAction = core.DBAction
	if p.config == nil {
		return fmt.Errorf("config is required")
	}

	if p.config.passwordMinLength < defaultMinPasswordLength {
		return fmt.Errorf("password min length must be at least 4")
	}

	return nil
}
