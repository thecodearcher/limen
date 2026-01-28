package twofactor

import (
	"os"

	"github.com/thecodearcher/aegis"
)

func generateBackupCodes(count int, length int) []string {
	backupCodes := make([]string, count)
	for i := range count {
		halfLength := length / 2
		backupCodes[i] = aegis.GenerateRandomString(halfLength, aegis.CharSetAlphanumeric) + "-" + aegis.GenerateRandomString(halfLength, aegis.CharSetAlphanumeric)
	}
	return backupCodes
}

func generateRandomOTP(digits TOTPDigits) string {
	return aegis.GenerateRandomString(int(digits), aegis.CharSetNumeric)
}

func getTOTPSecret() []byte {
	totpSecret := os.Getenv("AEGIS_TOTP_SECRET")
	secret := os.Getenv("AEGIS_SECRET")
	if totpSecret == "" && secret == "" {
		return []byte("aegis_2fa_totp_secret_1234567890")
	}

	if totpSecret != "" {
		return []byte(totpSecret)
	}

	return []byte(secret)
}
