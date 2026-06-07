package passkey

import (
	"time"

	"github.com/thecodearcher/limen"
)

const (
	// PasskeySchemaTableName is the physical table name used for stored credentials.
	PasskeySchemaTableName limen.SchemaTableName = "passkeys"

	PasskeySchemaUserIDField       limen.SchemaField = "user_id"
	PasskeySchemaNameField         limen.SchemaField = "name"
	PasskeySchemaCredentialIDField limen.SchemaField = "credential_id"
	PasskeySchemaPublicKeyField    limen.SchemaField = "public_key"
	PasskeySchemaCounterField      limen.SchemaField = "counter"
	PasskeySchemaDeviceTypeField   limen.SchemaField = "device_type"
	PasskeySchemaBackedUpField     limen.SchemaField = "backed_up"
	PasskeySchemaTransportsField   limen.SchemaField = "transports"
	PasskeySchemaAAGUIDField       limen.SchemaField = "aaguid"
)

const (
	// passkeyChallengeAction is the verification action used to persist a
	// challenge while the client completes a WebAuthn ceremony.
	passkeyChallengeAction = "passkey_challenge" // #nosec G101 -- action name, not a secret

	defaultRPName                = "Limen App"
	defaultChallengeCookieName   = "limen_passkey_challenge"
	defaultChallengeExpiration   = 5 * time.Minute
	defaultUserHandleByteLength  = 32
	verificationTokenByteLength  = 32
)

// CeremonyKind names the two WebAuthn ceremonies. It is persisted in the
// challenge state so a registration cookie cannot be replayed against an
// authentication endpoint, or vice versa.
type CeremonyKind string

const (
	CeremonyRegistration   CeremonyKind = "registration"
	CeremonyAuthentication CeremonyKind = "authentication"
)
