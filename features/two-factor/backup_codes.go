package twofactor

import (
	"context"
	"slices"
	"strings"

	"github.com/thecodearcher/aegis"
)

type backupCodes struct {
	*backupCodesConfig
	plugin *twoFactorFeature
}

func newBackupCodes(plugin *twoFactorFeature, config *backupCodesConfig) *backupCodes {
	return &backupCodes{
		backupCodesConfig: config,
		plugin:            plugin,
	}
}

// RegisterRoutes registers backup codes-specific routes
func (b *backupCodes) registerRoutes(httpCore *aegis.AegisHTTPCore, routeBuilder *aegis.RouteBuilder) {
	handlers := newBackupCodesHandlers(b, httpCore.Responder)
	routeBuilder.ProtectedGET("/backup-codes", "get-backup-codes", handlers.GetBackupCodes)
	routeBuilder.ProtectedPUT("/backup-codes", "update-backup-codes", handlers.UpdateBackupCodes)
	routeBuilder.ProtectedPOST("/backup-codes/verify", "verify-backup-code", handlers.VerifyBackupCode)
}

func (b *backupCodes) GenerateBackupCodes() []string {
	if b.customGenerator != nil {
		return b.customGenerator()
	}

	return generateBackupCodes(b.count, b.length)
}

func (b *backupCodes) decryptBackupCodes(backupCodes string) ([]string, error) {
	decryptedBackupCodes, err := b.plugin.decrypt(backupCodes)
	if err != nil {
		return nil, err
	}
	return strings.Split(decryptedBackupCodes, ","), nil
}

func (b *backupCodes) encryptBackupCodes(backupCodes []string) (string, error) {
	return b.plugin.encrypt(strings.Join(backupCodes, ","))
}

func (b *backupCodes) UpdateBackupCodes(ctx context.Context, userID any) ([]string, error) {
	backupCodes := b.GenerateBackupCodes()
	encryptedBackupCodes, err := b.encryptBackupCodes(backupCodes)
	if err != nil {
		return nil, err
	}
	twoFactor, err := b.plugin.FindTwoFactorByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	updatedData := &TwoFactor{
		BackupCodes: encryptedBackupCodes,
	}

	err = b.plugin.core.Update(ctx, b.plugin.twoFactorSchema, updatedData, []aegis.Where{
		aegis.Eq(b.plugin.twoFactorSchema.GetIDField(), twoFactor.ID),
	})
	if err != nil {
		return nil, err
	}
	return backupCodes, nil
}

func (b *backupCodes) GetBackupCodes(ctx context.Context, userID any) ([]string, error) {
	twoFactor, err := b.plugin.FindTwoFactorByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	decryptedBackupCodes, err := b.decryptBackupCodes(twoFactor.BackupCodes)
	if err != nil {
		return nil, err
	}
	return decryptedBackupCodes, nil
}

func (b *backupCodes) VerifyBackupCode(ctx context.Context, userID any, backupCode string) error {
	twoFactor, err := b.plugin.FindTwoFactorByUserID(ctx, userID)
	if err != nil {
		return ErrTwoFactorNotEnabled
	}

	decryptedBackupCodes, err := b.decryptBackupCodes(twoFactor.BackupCodes)
	if err != nil {
		return ErrInvalidBackupCode
	}

	encryptedBackupCodes, valid := b.checkAndExpireBackupCode(decryptedBackupCodes, backupCode)
	if !valid {
		return ErrInvalidBackupCode
	}
	updatedData := &TwoFactor{
		BackupCodes: encryptedBackupCodes,
	}

	return b.plugin.core.Update(ctx, b.plugin.twoFactorSchema, updatedData, []aegis.Where{
		aegis.Eq(b.plugin.twoFactorSchema.GetIDField(), twoFactor.ID),
	})
}

func (b *backupCodes) checkAndExpireBackupCode(backupCodes []string, backupCode string) (string, bool) {
	if !slices.Contains(backupCodes, backupCode) {
		return "", false
	}
	backupCodes = slices.DeleteFunc(backupCodes, func(code string) bool {
		return code == backupCode
	})

	encryptedBackupCodes, err := b.encryptBackupCodes(backupCodes)
	if err != nil {
		return "", false
	}
	return encryptedBackupCodes, true
}
