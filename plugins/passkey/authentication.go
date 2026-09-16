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

func (p *passkeyPlugin) BeginAuthentication(r *http.Request) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	ext, err := p.resolveExtensions(r, protocol.AssertCeremony)
	if err != nil {
		return nil, nil, err
	}
	loginOpts := []webauthn.LoginOption{
		webauthn.WithAssertionExtensions(webauthn.WithExtensionInputs(ext)),
	}

	if user, passkeys, ok := p.sessionPasskeysForLogin(r); ok {
		return p.webAuthn.BeginLogin(newWebAuthnUser(p.core, user, passkeys), loginOpts...)
	}

	return p.webAuthn.BeginDiscoverableLogin(loginOpts...)
}

func (p *passkeyPlugin) sessionPasskeysForLogin(r *http.Request) (*limen.User, []*Passkey, bool) {
	session, err := p.core.SessionManager.ValidateSession(r.Context(), r)
	if err != nil {
		return nil, nil, false
	}

	passkeys, err := p.FindPasskeysByUserID(r.Context(), session.User.ID)
	if err != nil || len(passkeys) == 0 {
		return nil, nil, false
	}

	return session.User, passkeys, true
}

func (p *passkeyPlugin) FinishAuthentication(r *http.Request) (*limen.User, error) {
	sessionData, err := p.getSessionDataFromCookie(r)
	if err != nil {
		return nil, err
	}

	user, credential, err := p.finishAuthenticationCeremony(r, sessionData)
	if err != nil {
		return nil, toPasskeyError(err)
	}

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

func (p *passkeyPlugin) finishAuthenticationCeremony(r *http.Request, sessionData *webauthn.SessionData) (*limen.User, *webauthn.Credential, error) {
	handler := p.discoverableUserHandler(r.Context())

	if len(sessionData.UserID) > 0 {
		passkeyUser, err := handler(nil, sessionData.UserID)
		if err != nil {
			return nil, nil, err
		}
		credential, err := p.webAuthn.FinishLogin(passkeyUser, *sessionData, r)
		if err != nil {
			return nil, nil, err
		}
		return passkeyUser.(*webAuthnUser).User(), credential, nil
	}

	passkeyUser, credential, err := p.webAuthn.FinishPasskeyLogin(handler, *sessionData, r)
	if err != nil {
		return nil, nil, err
	}
	return passkeyUser.(*webAuthnUser).User(), credential, nil
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
