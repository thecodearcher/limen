package passkey

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// webAuthnUser adapts a Limen user plus their stored credentials into
// the interface required by the go-webauthn library.
type webAuthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webAuthnUser) WebAuthnName() string                       { return u.name }
func (u *webAuthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// userHandleFor produces the WebAuthnID for a given Limen user id.
// WebAuthn requires a non-PII handle <= 64 bytes; we use a deterministic
// string derived from the user id so the same user always presents the
// same handle to their authenticator.
func userHandleFor(userID any) []byte {
	return []byte(fmt.Sprintf("limen-user-%v", userID))
}

// recordToCredential rehydrates a stored PasskeyRecord into the shape
// the go-webauthn library expects when verifying an authentication
// assertion or excluding an already-registered credential from
// registration.
func recordToCredential(rec *PasskeyRecord) (webauthn.Credential, error) {
	credID, err := base64.RawURLEncoding.DecodeString(rec.CredentialIDBase64)
	if err != nil {
		// Some clients use base64 with padding; tolerate both.
		credID, err = base64.StdEncoding.DecodeString(rec.CredentialIDBase64)
		if err != nil {
			return webauthn.Credential{}, fmt.Errorf("invalid credential id encoding: %w", err)
		}
	}
	pubKey, err := base64.StdEncoding.DecodeString(rec.PublicKeyBase64)
	if err != nil {
		pubKey, err = base64.RawURLEncoding.DecodeString(rec.PublicKeyBase64)
		if err != nil {
			return webauthn.Credential{}, fmt.Errorf("invalid public key encoding: %w", err)
		}
	}

	transports := []protocol.AuthenticatorTransport{}
	if rec.Transports != "" {
		for _, t := range strings.Split(rec.Transports, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				transports = append(transports, protocol.AuthenticatorTransport(t))
			}
		}
	}

	aaguid := []byte{}
	if rec.AAGUID != "" {
		if decoded, err := base64.StdEncoding.DecodeString(rec.AAGUID); err == nil {
			aaguid = decoded
		}
	}

	flags := webauthn.CredentialFlags{
		BackupEligible: rec.BackedUp,
		BackupState:    rec.BackedUp,
	}

	return webauthn.Credential{
		ID:        credID,
		PublicKey: pubKey,
		Transport: transports,
		Flags:     flags,
		Authenticator: webauthn.Authenticator{
			AAGUID:    aaguid,
			SignCount: uint32(rec.Counter),
		},
	}, nil
}
