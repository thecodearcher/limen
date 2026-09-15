package passkey

import (
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/thecodearcher/limen"
)

// webAuthnUser presents a Limen user and their stored passkeys in the shape the
// WebAuthn library expects during a ceremony.
type webAuthnUser struct {
	core     *limen.LimenCore
	user     *limen.User
	passkeys []*Passkey
}

func newWebAuthnUser(core *limen.LimenCore, user *limen.User, passkeys []*Passkey) *webAuthnUser {
	return &webAuthnUser{
		core:     core,
		user:     user,
		passkeys: passkeys,
	}
}

func (u *webAuthnUser) User() *limen.User {
	return u.user
}

func (u *webAuthnUser) WebAuthnID() []byte {
	if encoded, ok := u.core.EncodePublicID(u.core.Schema.User, u.user); ok {
		return []byte(encoded)
	}
	return fmt.Append(nil, u.user.ID)
}

func (u *webAuthnUser) WebAuthnName() string {
	return u.user.Email
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	return u.user.Email
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	credentials := make([]webauthn.Credential, 0, len(u.passkeys))

	for _, passkey := range u.passkeys {
		credentials = append(credentials, toWebAuthnCredential(passkey))
	}

	return credentials
}
