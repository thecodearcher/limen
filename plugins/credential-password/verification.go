package credentialpassword

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/thecodearcher/aegis"
)

// RequestEmailVerification requests an email verification for the given user
// and sends the verification email if shouldSendEmail is true and the send function is configured.
func (p *credentialPasswordPlugin) RequestEmailVerification(ctx context.Context, user *aegis.User, shouldSendEmail bool) (*aegis.Verification, error) {
	user, err := p.dbAction.FindUserByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	if user.EmailVerifiedAt != nil {
		return nil, ErrEmailAlreadyVerified
	}

	verification, err := p.CreateEmailVerification(ctx, user)
	if err != nil {
		return nil, err
	}

	if shouldSendEmail && verification != nil {
		p.SendVerificationEmail(ctx, user, verification)
	}

	return verification, nil
}

// CreateEmailVerification creates a new email verification token for the given user.
// Returns a Verification object containing the verification token.
func (p *credentialPasswordPlugin) CreateEmailVerification(ctx context.Context, user *aegis.User) (*aegis.Verification, error) {
	token, err := p.generateVerificationToken(user)
	if err != nil {
		return nil, err
	}
	return p.dbAction.CreateVerification(ctx, EmailVerificationAction, user.Email, token, p.config.emailVerificationExpiration)
}

// SendVerificationEmail sends the verification email if the send function is configured.
func (p *credentialPasswordPlugin) SendVerificationEmail(ctx context.Context, user *aegis.User, verification *aegis.Verification) {
	if p.config.sendVerificationEmail != nil {
		p.config.sendVerificationEmail(user.Email, verification.Value)
	}
}

// VerifyEmail verifies an email address using the provided verification token.
// Returns ErrEmailVerificationTokenInvalid if the token is invalid, expired, or for a different action.
func (p *credentialPasswordPlugin) VerifyEmail(ctx context.Context, token string) error {
	verification, err := p.dbAction.FindValidVerificationByToken(ctx, token)
	if err != nil {
		return ErrResetTokenInvalid
	}

	action, identifier := aegis.ParseVerificationAction(verification.Subject)
	if action != EmailVerificationAction {
		return ErrResetTokenInvalid
	}

	now := time.Now()
	err = p.core.WithTransaction(ctx, func(ctx context.Context) error {
		if err := p.dbAction.UpdateUser(ctx, &aegis.User{EmailVerifiedAt: &now},
			[]aegis.Where{
				aegis.Eq(p.userSchema.GetEmailField(), identifier),
			}); err != nil {
			return err
		}

		return p.dbAction.DeleteVerificationToken(ctx, verification.Value)
	})

	return err
}

// generateVerificationToken generates a cryptographically secure verification token.
// Uses the custom token generator if configured, otherwise generates a random 32-byte token.
func (p *credentialPasswordPlugin) generateVerificationToken(user *aegis.User) (string, error) {
	if p.config.generateResetToken != nil {
		return p.config.generateResetToken(user)
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}
