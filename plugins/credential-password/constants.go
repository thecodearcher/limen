package credentialpassword

import "github.com/thecodearcher/aegis"

const (
	defaultMinPasswordLength        = 4
	defaultPasswordRequireUppercase = true
	defaultPasswordRequireNumbers   = true
	defaultPasswordRequireSymbols   = false
)

const (
	defaultMinUsernameLength = 3
	defaultMaxUsernameLength = 30
)

const (
	PasswordResetAction     = "password_reset"
	EmailVerificationAction = "email_verification"
)

const (
	CredentialPasswordUserSchemaUsernameField aegis.SchemaField = "username"
)
