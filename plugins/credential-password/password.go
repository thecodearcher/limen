package credentialpassword

import (
	"context"
	"fmt"

	"github.com/thecodearcher/limen"
)

// HashPassword hashes a password using the configured hashing function or the default Argon2id hasher.
// Returns the hashed password string or an error if hashing fails.
func (p *credentialPasswordPlugin) HashPassword(password string) (string, error) {
	if p.config.hashFn != nil {
		return p.config.hashFn(password)
	}

	return newPasswordHasher(p.config.passwordHasherConfig).hashPassword([]byte(password))
}

// ComparePassword compares a plain text password with a hashed password.
// Returns true if they match, false otherwise, or an error if comparison fails.
func (p *credentialPasswordPlugin) ComparePassword(password string, hash *string) (bool, error) {
	if hash == nil {
		// this is only possible when user signs in with oauth
		return false, ErrPasswordNotSet
	}

	if p.config.compareFn != nil {
		return p.config.compareFn(password, *hash)
	}

	return newPasswordHasher(p.config.passwordHasherConfig).verifyPassword([]byte(password), *hash)
}

// RequestPasswordReset generates a password reset token for the given email address.
// Returns a Verification object containing the reset token on success.
func (p *credentialPasswordPlugin) RequestPasswordReset(ctx context.Context, email string) (*limen.Verification, error) {
	user, err := p.dbAction.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrEmailNotFound
	}

	token, err := p.generateVerificationToken(user)
	if err != nil {
		return nil, err
	}

	verification, err := p.dbAction.CreateVerification(ctx, PasswordResetAction, email, token, p.config.resetTokenExpiration)
	if err != nil {
		return nil, err
	}

	if p.config.sendPasswordResetEmail != nil {
		p.config.sendPasswordResetEmail(email, verification.Value)
	}

	return verification, nil
}

// ResetPassword resets a user's password using a valid reset token.
func (p *credentialPasswordPlugin) ResetPassword(ctx context.Context, token string, newPassword string) error {
	verification, err := p.dbAction.FindValidVerificationByToken(ctx, token)
	if err != nil {
		return ErrResetTokenInvalid
	}

	action, identifier := limen.ParseVerificationAction(verification.Subject)
	if action != PasswordResetAction {
		return ErrResetTokenInvalid
	}

	if err := p.validatePassword(newPassword); err != nil {
		return err
	}

	hashedPassword, err := p.HashPassword(newPassword)
	if err != nil {
		return err
	}

	err = p.core.WithTransaction(ctx, func(ctx context.Context) error {
		if err := p.dbAction.UpdateUser(ctx, &limen.User{Password: &hashedPassword}, []limen.Where{
			limen.Eq(p.userSchema.GetEmailField(), identifier),
		}); err != nil {
			return fmt.Errorf("failed to update user password: %w", err)
		}

		return p.dbAction.DeleteVerificationToken(ctx, token)
	})
	if err != nil {
		return err
	}

	if p.config.onPasswordResetSuccess != nil {
		user, err := p.dbAction.FindUserByEmail(ctx, identifier)
		if err != nil {
			return err
		}

		p.config.onPasswordResetSuccess(ctx, user)
	}

	return nil
}

// SetPassword sets a password for a user who doesn't have one (e.g., signed up via OAuth).
//
// Note: If revokeOtherSessions is true, the current session will be revoked and a new session should be created.
func (p *credentialPasswordPlugin) SetPassword(ctx context.Context, user *limen.User, newPassword string, revokeOtherSessions bool) error {
	if user.Password != nil {
		return ErrPasswordAlreadySet
	}

	if err := p.validatePassword(newPassword); err != nil {
		return err
	}

	hashedPassword, err := p.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return p.core.WithTransaction(ctx, func(ctx context.Context) error {
		if err := p.dbAction.UpdateUser(ctx, &limen.User{Password: &hashedPassword}, []limen.Where{
			limen.Eq(p.userSchema.GetIDField(), user.ID),
			limen.IsNull(p.userSchema.GetPasswordField()),
			limen.Eq(p.userSchema.GetPasswordField(), "").Or(),
		}); err != nil {
			return err
		}
		if revokeOtherSessions {
			return p.dbAction.DeleteSessionByUserID(ctx, user.ID)
		}
		return nil
	})
}

// UpdatePassword updates the password for the given user and revokes other sessions if requested.
//
// Note: If revokeOtherSessions is true, the current session will be revoked and a new session should be created.
func (p *credentialPasswordPlugin) UpdatePassword(ctx context.Context, user *limen.User, currentPassword string, newPassword string, revokeOtherSessions bool) error {
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

	return p.core.WithTransaction(ctx, func(ctx context.Context) error {
		if err := p.dbAction.UpdateUser(ctx, &limen.User{Password: &hashedPassword}, []limen.Where{
			limen.Eq(p.userSchema.GetIDField(), user.ID),
		}); err != nil {
			return err
		}
		if revokeOtherSessions {
			return p.dbAction.DeleteSessionByUserID(ctx, user.ID)
		}
		return nil
	})
}
