// Package credentialpassword provides credential(email/username) and password authentication for the limen library.
package credentialpassword

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/thecodearcher/limen"
)

type credentialPasswordPlugin struct {
	core               *limen.LimenCore
	config             *config
	userSchema         *limen.UserSchema
	verificationSchema *limen.VerificationSchema
	dbAction           *limen.DatabaseActionHelper
}

// getUsernameField returns the resolved username field column name if username support is enabled.
func (p *credentialPasswordPlugin) getUsernameField() string {
	if !p.config.enableUsername {
		return ""
	}
	return p.userSchema.GetField("username")
}

// Config defines the configuration for the credential password plugin.
type config struct {
	passwordMinLength        int                                              // Minimum length of the password
	passwordRequireUppercase bool                                             // Require uppercase letters in the password
	passwordRequireNumbers   bool                                             // Require numbers in the password
	passwordRequireSymbols   bool                                             // Require symbols in the password
	hashFn                   func(password string) (string, error)            // Custom function to hash the password
	compareFn                func(password string, hash string) (bool, error) // Custom function to compare the password and the hash
	passwordHasherConfig     passwordHasherConfig                             // Custom Argon2id configuration for the password hasher
	resetTokenExpiration     time.Duration                                    // Custom expiration duration for the reset token
	generateResetToken       func(*limen.User) (string, error)                // custom function to generate the reset token e.g generating TOTP code
	autoSignInOnSignUp       bool                                             // auto sign in the user after sign up
	sendPasswordResetEmail   func(email string, token string)                 // function to send the password reset message
	onPasswordResetSuccess   func(ctx context.Context, user *limen.User)      // function to call when the password reset is successful
	enableUsername           bool                                             // enable username support (default: false)
	usernameMinLength        int                                              // Minimum length of the username
	usernameMaxLength        int                                              // Maximum length of the username
	usernameValidationRegex  *regexp.Regexp                                   // Custom regex pattern for username validation
	usernameRequiredOnSignup bool                                             // require username during sign up
	usernameValidationFunc   func(username string) error                      // custom function to validate the username
}

// New returns a new config with the default values.
// ConfigOptions can be provided to customize the configuration.
func New(opts ...ConfigOption) *credentialPasswordPlugin {
	config := &config{
		passwordMinLength:        defaultMinPasswordLength,
		passwordRequireUppercase: defaultPasswordRequireUppercase,
		passwordRequireNumbers:   defaultPasswordRequireNumbers,
		passwordRequireSymbols:   defaultPasswordRequireSymbols,
		passwordHasherConfig:     DefaultPasswordHasherConfig(),
		resetTokenExpiration:     30 * time.Minute,
		autoSignInOnSignUp:       true,
		enableUsername:           false,
		usernameMinLength:        defaultMinUsernameLength,
		usernameMaxLength:        defaultMaxUsernameLength,
		usernameValidationRegex:  regexp.MustCompile(`^[a-zA-Z0-9_-]+$`), // alphanumeric, underscore, hyphen
		usernameRequiredOnSignup: false,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &credentialPasswordPlugin{
		config: config,
	}
}

func (p *credentialPasswordPlugin) Name() limen.PluginName {
	return limen.PluginCredentialPassword
}

func (p *credentialPasswordPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	if !p.config.enableUsername {
		return []limen.SchemaIntrospector{}
	}

	userWithUsername := &CredentialPasswordUserSchema{
		UserSchema: schema.User,
	}
	extension := limen.NewSchemaDefinitionForExtension(
		limen.CoreSchemaUsers,
		userWithUsername,
		limen.WithSchemaField("username", limen.ColumnTypeString, limen.WithNullable(true)),
		limen.WithSchemaIndex("idx_users_username", []limen.SchemaField{CredentialPasswordUserSchemaUsernameField}),
	)

	return []limen.SchemaIntrospector{extension}
}

func (p *credentialPasswordPlugin) Initialize(core *limen.LimenCore) error {
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
