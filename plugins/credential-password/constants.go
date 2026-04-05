package credentialpassword

import "github.com/thecodearcher/limen"

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
	PasswordResetAction = "password_reset"
)

const (
	CredentialPasswordUserSchemaUsernameField limen.SchemaField = "username"
)
