package twofactor

import (
	"os"
	"strings"

	"github.com/thecodearcher/limen"
)

func generateBackupCodes(count int, length int) []string {
	backupCodes := make([]string, count)
	for i := range count {
		halfLength := length / 2
		backupCodes[i] = limen.GenerateRandomString(halfLength, limen.CharSetAlphanumeric) + "-" + limen.GenerateRandomString(halfLength, limen.CharSetAlphanumeric)
	}
	return backupCodes
}

func generateRandomOTP(digits TOTPDigits) string {
	return limen.GenerateRandomString(int(digits), limen.CharSetNumeric)
}

func getTOTPSecret() []byte {
	totpSecret := os.Getenv("AEGIS_TOTP_SECRET")
	secret := os.Getenv("AEGIS_SECRET")
	if totpSecret == "" && secret == "" {
		return nil
	}
	if totpSecret != "" {
		return []byte(totpSecret)
	}
	return []byte(secret)
}

func looksLikeBackupCode(code string) bool {
	return strings.Contains(code, "-")
}
