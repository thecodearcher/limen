package limen

import (
	"context"
	"time"
)

// EmailVerificationEnabled reports whether email verification is enabled.
func (c *LimenCore) EmailVerificationEnabled() bool {
	return c.config.EmailVerification.enabled
}

// RequestEmailVerification looks up the user by email, ensures the address
// is not already verified, creates a verification token and optionally sends
// the email.
func (c *LimenCore) RequestEmailVerification(ctx context.Context, user *User, shouldSendEmail bool) (*Verification, error) {
	user, err := c.DBAction.FindUserByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	if user.EmailVerifiedAt != nil {
		return nil, ErrEmailAlreadyVerified
	}

	verification, err := c.CreateEmailVerification(ctx, user)
	if err != nil {
		return nil, err
	}

	if shouldSendEmail && verification != nil {
		c.SendEmailVerificationMail(user, verification)
	}

	return verification, nil
}

// CreateEmailVerification creates a new verification token for the user.
func (c *LimenCore) CreateEmailVerification(ctx context.Context, user *User) (*Verification, error) {
	token, err := c.generateEmailVerificationToken(user)
	if err != nil {
		return nil, err
	}
	return c.DBAction.CreateVerification(ctx, EmailVerificationAction, user.Email, token, c.config.EmailVerification.expiration)
}

// SendEmailVerificationMail dispatches the verification email when a callback is configured.
func (c *LimenCore) SendEmailVerificationMail(user *User, verification *Verification) {
	if c.config.EmailVerification.sendEmail != nil {
		c.config.EmailVerification.sendEmail(user.Email, verification.Value)
	}
}

// VerifyEmail validates the token, marks the user's email as verified, and
// deletes the consumed token
func (c *LimenCore) VerifyEmail(ctx context.Context, token string) error {
	verification, err := c.DBAction.FindValidVerificationByToken(ctx, token)
	if err != nil {
		return ErrEmailVerificationTokenInvalid
	}

	action, identifier := ParseVerificationAction(verification.Subject)
	if action != EmailVerificationAction {
		return ErrEmailVerificationTokenInvalid
	}

	now := time.Now()
	return c.WithTransaction(ctx, func(ctx context.Context) error {
		if err := c.DBAction.UpdateUser(ctx, &User{EmailVerifiedAt: &now},
			[]Where{
				Eq(c.Schema.User.GetEmailField(), identifier),
			}); err != nil {
			return err
		}

		return c.DBAction.DeleteVerificationToken(ctx, verification.Value)
	})
}

func (c *LimenCore) generateEmailVerificationToken(user *User) (string, error) {
	if c.config.EmailVerification.generateToken != nil {
		return c.config.EmailVerification.generateToken(user)
	}

	return generateCryptoSecureRandomString(), nil
}
