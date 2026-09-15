package passkey

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/thecodearcher/limen"
)

func (p *passkeyPlugin) BeginAuthentication(ctx context.Context) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	opts, sessionData, err := p.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, nil, err
	}

	return opts, sessionData, nil
}

func (p *passkeyPlugin) FinishAuthentication(r *http.Request) (*limen.User, error) {
	sessionData, err := p.getSessionDataFromCookie(r)
	if err != nil {
		return nil, err
	}

	passkeyUser, credential, err := p.webAuthn.FinishPasskeyLogin(p.discoverableUserHandler(r.Context()), *sessionData, r)
	if err != nil {
		return nil, toPasskeyError(err)
	}

	user := passkeyUser.(*webAuthnUser).User()

	updateData := map[limen.SchemaField]any{
		PasskeySchemaSignCountField:      credential.Authenticator.SignCount,
		PasskeySchemaBackupEligibleField: credential.Flags.BackupEligible,
		PasskeySchemaBackupStateField:    credential.Flags.BackupState,
		PasskeySchemaLastUsedAtField:     time.Now(),
	}

	if err := p.core.Update(r.Context(), p.passkeySchema, updateData, []limen.Where{
		limen.Eq(p.passkeySchema.GetCredentialIDField(), base64.RawURLEncoding.EncodeToString(credential.ID)),
	}); err != nil {
		return nil, err
	}

	return user, nil
}

func (p *passkeyPlugin) discoverableUserHandler(ctx context.Context) webauthn.DiscoverableUserHandler {
	return func(rawID, userHandle []byte) (user webauthn.User, err error) {
		userModel, err := p.core.DBAction.FindUser(ctx, []limen.Where{
			limen.Eq(p.core.Schema.User.GetIDField(), string(userHandle)),
		})
		if err != nil {
			return nil, err
		}
		passkeys, err := p.FindPasskeysByUserID(ctx, userModel.ID)
		if err != nil {
			return nil, err
		}

		return newWebAuthnUser(p.core, userModel, passkeys), nil
	}
}
