// Package emailpassword provides email/password authentication for the aegis library.
package emailpassword

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/internal/database"
)

type emailPasswordFeature struct {
	core               *aegis.AegisCore
	config             *config
	userSchema         *aegis.UserSchema
	verificationSchema *aegis.VerificationSchema
	dbAction           *database.DatabaseActionHelper
}

// Config defines the configuration for the email password feature.
type config struct {
	passwordMinLength          int                                              // Minimum length of the password
	passwordRequireUppercase   bool                                             // Require uppercase letters in the password
	passwordRequireNumbers     bool                                             // Require numbers in the password
	passwordRequireSymbols     bool                                             // Require symbols in the password
	hashFn                     func(password string) (string, error)            // Custom function to hash the password
	compareFn                  func(password string, hash string) (bool, error) // Custom function to compare the password and the hash
	passwordHasherConfig       passwordHasherConfig                             // Custom Argon2id configuration for the password hasher
	requireEmailVerification   bool                                             // require email verification after sign up
	resetTokenExpiration       time.Duration                                    // Custom expiration duration for the reset token
	generateResetToken         func(*aegis.User) (string, error)                // custom function to generate the reset token e.g generating TOTP code
	removeExpiredVerifications bool                                             // remove expired verifications after reset password
	autoSignInOnSignUp         bool                                             // auto sign in the user after sign up
	sendVerificationEmail      func(email string, token string) error           // custom function to send the email verification message
	sendPasswordResetEmail     func(email string, token string) error           // custom function to send the password reset message
}

// New returns a new config with the default values.
// ConfigOptions can be provided to customize the configuration.
func New(opts ...ConfigOption) *emailPasswordFeature {
	config := &config{
		passwordMinLength:          defaultMinPasswordLength,
		passwordRequireUppercase:   defaultPasswordRequireUppercase,
		passwordRequireNumbers:     defaultPasswordRequireNumbers,
		passwordRequireSymbols:     defaultPasswordRequireSymbols,
		passwordHasherConfig:       DefaultPasswordHasherConfig(),
		resetTokenExpiration:       30 * time.Minute,
		removeExpiredVerifications: true,
		autoSignInOnSignUp:         true,
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
	p.dbAction = database.NewCommonDatabaseActionsHelper(core)
	if p.config == nil {
		return fmt.Errorf("config is required")
	}

	if p.config.passwordMinLength < defaultMinPasswordLength {
		return fmt.Errorf("password min length must be at least 4")
	}

	return nil
}

func (p *emailPasswordFeature) SignInWithEmailAndPassword(ctx context.Context, email string, password string) (*aegis.AuthenticationResult, error) {
	user, err := p.dbAction.FindUserByEmail(ctx, email)
	if err != nil {
		// hash the password to avoid timing attacks when the user is not found
		// this allows constant response time for both valid and invalid credentials
		_, _ = p.HashPassword(password)
		return nil, ErrEmailNotFound
	}

	isValid, err := p.ComparePassword(password, user.Password)
	if err != nil {
		return nil, err
	}

	if !isValid {
		return nil, ErrInvalidPassword
	}

	pendingActions := []aegis.PendingAction{}
	if p.config.requireEmailVerification && user.EmailVerifiedAt == nil {
		pendingActions = append(pendingActions, aegis.PendingActionEmailVerification)
	}

	return &aegis.AuthenticationResult{
		User:           user,
		PendingActions: pendingActions,
	}, nil
}

func (p *emailPasswordFeature) SignUpWithEmailAndPassword(ctx context.Context, user *aegis.User, additionalFields map[string]any) (*aegis.AuthenticationResult, error) {
	if err := p.validateUser(user); err != nil {
		return nil, err
	}

	userExists, err := database.Exists(ctx, p.core, p.userSchema, []aegis.Where{aegis.Eq(p.userSchema.GetEmailField(), user.Email)})
	if err != nil {
		return nil, err
	}

	if userExists {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := p.HashPassword(user.Password)
	if err != nil {
		return nil, err
	}

	err = p.dbAction.CreateUser(ctx, &aegis.User{
		Email:    user.Email,
		Password: hashedPassword,
	}, additionalFields)

	if err != nil {
		return nil, err
	}

	pendingActions := []aegis.PendingAction{}

	if p.config.requireEmailVerification {
		if _, err := p.RequestEmailVerification(ctx, user); err != nil {
			return nil, err
		}
		pendingActions = append(pendingActions, aegis.PendingActionEmailVerification)
	}

	user, err = p.dbAction.FindUserByEmail(ctx, user.Email)

	return &aegis.AuthenticationResult{
		User:           user,
		PendingActions: pendingActions,
	}, err
}

func (p *emailPasswordFeature) HashPassword(password string) (string, error) {
	if p.config.hashFn != nil {
		return p.config.hashFn(password)
	}

	return newPasswordHasher(p.config.passwordHasherConfig).hashPassword([]byte(password))
}

func (p *emailPasswordFeature) ComparePassword(password string, hash string) (bool, error) {
	if p.config.compareFn != nil {
		return p.config.compareFn(password, hash)
	}

	return newPasswordHasher(p.config.passwordHasherConfig).verifyPassword([]byte(password), hash)
}

func (p *emailPasswordFeature) RequestPasswordReset(ctx context.Context, email string) (*aegis.Verification, error) {
	user, err := database.FindOne(ctx, p.core, p.userSchema, []aegis.Where{
		aegis.Eq(p.userSchema.GetEmailField(), email),
	}, nil)
	if err != nil {
		return nil, err
	}

	token, err := p.generateVerificationToken(user)
	if err != nil {
		return nil, err
	}

	verification, err := p.dbAction.CreateVerification(ctx, PasswordResetAction, email, token, p.config.resetTokenExpiration)
	if err != nil {
		return nil, err
	}

	return verification, nil
}

func (p *emailPasswordFeature) ResetPassword(ctx context.Context, token string, newPassword string) error {
	verification, err := p.dbAction.FindVerificationByToken(ctx, token)
	if err != nil {
		return err
	}

	action, identifier := database.ParseVerificationAction(verification.Subject)
	if action != PasswordResetAction {
		return ErrResetTokenInvalid
	}

	if verification.ExpiresAt.Before(time.Now().UTC()) {
		return ErrResetTokenInvalid
	}

	hashedPassword, err := p.HashPassword(newPassword)
	if err != nil {
		return err
	}

	err = p.dbAction.UpdateUser(ctx, &aegis.User{Password: hashedPassword}, []aegis.Where{
		aegis.Eq(p.userSchema.GetEmailField(), identifier),
	})
	if err != nil {
		return err
	}

	if p.config.removeExpiredVerifications {
		err = p.dbAction.DeleteExpiredVerifications(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *emailPasswordFeature) UpdatePassword(ctx context.Context, user *aegis.User, currentPassword string, newPassword string) error {
	if err := p.validatePassword(newPassword); err != nil {
		return err
	}

	hashedPassword, err := p.HashPassword(newPassword)
	if err != nil {
		return err
	}

	isValid, err := p.ComparePassword(currentPassword, user.Password)
	if err != nil {
		return err
	}

	if !isValid {
		return ErrInvalidCurrentPassword
	}

	err = p.dbAction.UpdateUser(ctx, &aegis.User{Password: hashedPassword}, []aegis.Where{
		aegis.Eq(p.userSchema.GetIDField(), user.ID),
	})
	if err != nil {
		return err
	}
	return nil
}

// RequestEmailVerification requests an email verification for the given user
// and sends the verification email if the function is set.
func (p *emailPasswordFeature) RequestEmailVerification(ctx context.Context, user *aegis.User) (*aegis.Verification, error) {
	user, err := p.dbAction.FindUserByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	if user.EmailVerifiedAt != nil {
		return nil, ErrEmailAlreadyVerified
	}

	token, err := p.generateVerificationToken(user)
	if err != nil {
		return nil, err
	}

	verification, err := p.dbAction.CreateVerification(ctx, EmailVerificationAction, user.Email, token, p.config.resetTokenExpiration)
	if err != nil {
		return nil, err
	}

	if p.config.removeExpiredVerifications {
		if err = p.dbAction.DeleteExpiredVerifications(ctx); err != nil {
			return nil, err
		}
	}

	if p.config.sendVerificationEmail != nil {
		if err := p.config.sendVerificationEmail(user.Email, verification.Value); err != nil {
			return nil, err
		}
	}

	return verification, nil
}

func (p *emailPasswordFeature) VerifyEmail(ctx context.Context, token string) error {
	verification, err := p.dbAction.FindVerificationByToken(ctx, token)
	if err != nil {
		return err
	}

	action, identifier := database.ParseVerificationAction(verification.Subject)
	if action != EmailVerificationAction {
		return ErrResetTokenInvalid
	}

	now := time.Now()
	return p.dbAction.UpdateUser(ctx, &aegis.User{EmailVerifiedAt: &now}, []aegis.Where{
		aegis.Eq(p.userSchema.GetEmailField(), identifier),
	})
}

func (p *emailPasswordFeature) generateVerificationToken(user *aegis.User) (string, error) {
	if p.config.generateResetToken != nil {
		return p.config.generateResetToken(user)
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

func (p *emailPasswordFeature) validatePassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}
	if len(password) < p.config.passwordMinLength {
		return ErrPasswordTooShort
	}
	if p.config.passwordRequireUppercase && !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return ErrPasswordRequiresUppercase
	}
	if p.config.passwordRequireNumbers && !strings.ContainsAny(password, "0123456789") {
		return ErrPasswordRequiresNumbers
	}
	if p.config.passwordRequireSymbols && !strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
		return ErrPasswordRequiresSymbols
	}
	return nil
}

func (p *emailPasswordFeature) validateUser(user *aegis.User) error {
	if user.Email == "" {
		return ErrEmailRequired
	}
	if user.Password == "" {
		return ErrPasswordRequired
	}
	if err := p.validatePassword(user.Password); err != nil {
		return err
	}
	return nil
}
