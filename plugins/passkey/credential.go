package passkey

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// toWebAuthnCredential converts a stored passkey into the form the WebAuthn
// library verifies ceremonies against.
func toWebAuthnCredential(passkey *Passkey) webauthn.Credential {
	credentialID, _ := base64.RawURLEncoding.DecodeString(passkey.CredentialID)
	publicKey, _ := base64.RawURLEncoding.DecodeString(passkey.PublicKey)

	transports := make([]protocol.AuthenticatorTransport, 0, len(passkey.Transports))
	for _, transport := range passkey.Transports {
		transports = append(transports, protocol.AuthenticatorTransport(transport))
	}

	return webauthn.Credential{
		ID:        credentialID,
		PublicKey: publicKey,
		Transport: transports,
		Flags: webauthn.CredentialFlags{
			BackupEligible: passkey.BackupEligible,
			BackupState:    passkey.BackupState,
		},
		Authenticator: webauthn.Authenticator{
			SignCount: passkey.SignCount,
		},
	}
}

// fromWebAuthnCredential builds a passkey row out of a freshly registered credential.
func fromWebAuthnCredential(userID any, credential *webauthn.Credential) *Passkey {
	transports := make([]string, 0, len(credential.Transport))
	for _, transport := range credential.Transport {
		transports = append(transports, string(transport))
	}

	now := time.Now()
	return &Passkey{
		UserID:         userID,
		CredentialID:   base64.RawURLEncoding.EncodeToString(credential.ID),
		PublicKey:      base64.RawURLEncoding.EncodeToString(credential.PublicKey),
		SignCount:      credential.Authenticator.SignCount,
		Transports:     transports,
		AAGUID:         formatAAGUID(credential.Authenticator.AAGUID),
		BackupEligible: credential.Flags.BackupEligible,
		BackupState:    credential.Flags.BackupState,
		LastUsedAt:     &now,
	}
}

// formatAAGUID renders an authenticator model identifier as a UUID, which is the
// form FIDO's metadata service keys on.
func formatAAGUID(raw []byte) string {
	if len(raw) != 16 {
		return ""
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}
