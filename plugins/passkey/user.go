package passkey

import (
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/thecodearcher/limen"
)

// webAuthnUser presents a Limen user and their stored passkeys in the shape the
// WebAuthn library expects during a ceremony.
type webAuthnUser struct {
	user     *limen.User
	passkeys []*Passkey
	plugin   *passkeyPlugin
}

func newWebAuthnUser(plugin *passkeyPlugin, user *limen.User, passkeys []*Passkey) *webAuthnUser {
	return &webAuthnUser{
		user:     user,
		passkeys: passkeys,
		plugin:   plugin,
	}
}

func (u *webAuthnUser) User() *limen.User {
	return u.user
}

func (u *webAuthnUser) WebAuthnID() []byte {
	if u.plugin.config.allowInternalUserIDAsHandle {
		handle, err := opaqueIDString(u.user.ID)
		if err != nil {
			return nil
		}
		return []byte(handle)
	}

	encoded, ok := u.plugin.core.EncodePublicID(u.plugin.core.Schema.User, u.user)
	if !ok {
		return nil
	}
	return []byte(encoded)
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

func (p *passkeyPlugin) userFromIntent(intent *RegistrationIntent, reservedHandle string) *limen.User {
	schema := p.core.Schema.User
	data := map[string]any{
		schema.GetEmailField(): intent.Email,
	}
	if p.config.allowInternalUserIDAsHandle {
		data[schema.GetIDField()] = reservedHandle
	} else {
		data[p.core.PublicIDColumn(schema)] = reservedHandle
	}
	return schema.FromStorage(data).(*limen.User)
}
