package twofactor

import (
	"context"

	"github.com/thecodearcher/aegis"
)

// InitiateTwoFactorSetup initiates the 2FA setup process
func (t *twoFactorFeature) InitiateTwoFactorSetup(ctx context.Context, user *UserWithTwoFactor, password string) (*TwoFactorSetupURI, error) {
	if user.TwoFactorEnabled {
		return nil, ErrTwoFactorAlreadyEnabled
	}

	if err := t.checkPassword(user.User, password); err != nil {
		return nil, err
	}

	result, err := t.totp.GenerateSetupURI(user.Email, "")
	if err != nil {
		return nil, err
	}

	backupCodes := t.backupCodes.GenerateBackupCodes()
	encryptedBackupCodes, err := t.backupCodes.encryptBackupCodes(backupCodes)
	if err != nil {
		return nil, err
	}

	encryptedSecret, err := t.encrypt(result.Secret)
	if err != nil {
		return nil, err
	}

	if err := t.DeleteTwoFactor(ctx, user.ID); err != nil {
		return nil, err
	}

	if err := t.CreateTwoFactor(ctx, user.ID, encryptedSecret, encryptedBackupCodes); err != nil {
		return nil, err
	}

	return result, nil
}

// FinalizeTwoFactorSetup finalizes the 2FA setup process
// It verifies the provided code and enables 2FA for the user
func (t *twoFactorFeature) FinalizeTwoFactorSetup(ctx context.Context, user *UserWithTwoFactor, code string) error {
	if user.TwoFactorEnabled {
		return ErrTwoFactorAlreadyEnabled
	}

	valid := t.totp.VerifyCode(ctx, user.ID, code)
	if !valid {
		return ErrInvalidCode
	}

	updatedUser := &UserWithTwoFactor{TwoFactorEnabled: true}
	if err := t.core.Update(ctx, t.userSchema, updatedUser, []aegis.Where{
		aegis.Eq(t.userSchema.GetIDField(), user.ID),
	}); err != nil {
		return err
	}

	return nil
}

// DisableTwoFactor disables 2FA for a user
func (t *twoFactorFeature) DisableTwoFactor(ctx context.Context, userID any, password string) error {
	user, err := t.core.DBAction.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err = t.checkPassword(user, password); err != nil {
		return err
	}

	return t.core.WithTransaction(ctx, func(ctx context.Context) error {
		if err := t.DeleteTwoFactor(ctx, userID); err != nil {
			return err
		}

		updatedUser := &UserWithTwoFactor{TwoFactorEnabled: false}

		return t.core.UpdateRaw(ctx, t.userSchema, updatedUser, []aegis.Where{
			aegis.Eq(t.userSchema.GetIDField(), userID),
		}, false)
	})
}

func (t *twoFactorFeature) checkPassword(user *aegis.User, password string) error {
	credentialPassword := t.core.GetCredentialPasswordFeature()
	if credentialPassword == nil {
		return ErrCredentialPasswordFeatureNotAvailable
	}

	isValid, err := credentialPassword.ComparePassword(password, user.Password)
	if err != nil {
		return err
	}
	if !isValid {
		return ErrInvalidPassword
	}

	return nil
}
