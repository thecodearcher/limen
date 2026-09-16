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

func (p *passkeyPlugin) BeginAuthentication(r *http.Request) (*CredentialAssertion, *PasskeyChallenge, error) {
	ext, err := p.resolveExtensions(r, protocol.AssertCeremony)
	if err != nil {
		return nil, nil, err
	}
	loginOpts := []webauthn.LoginOption{
		webauthn.WithAssertionExtensions(webauthn.WithExtensionInputs(ext)),
	}

	if user, passkeys, ok := p.sessionPasskeysForLogin(r); ok {
		assertion, sessionData, err := p.webAuthn.BeginLogin(newWebAuthnUser(p, user, passkeys), loginOpts...)
		if err != nil {
			return nil, nil, err
		}
		return assertion, &PasskeyChallenge{Session: *sessionData}, nil
	}

	assertion, sessionData, err := p.webAuthn.BeginDiscoverableLogin(loginOpts...)
	if err != nil {
		return nil, nil, err
	}
	return assertion, &PasskeyChallenge{Session: *sessionData}, nil
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
	cookie, err := p.getChallengeCookie(r)
	if err != nil {
		return nil, err
	}

	user, credential, err := p.finishAuthenticationCeremony(r, &cookie.Session)
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
	if len(sessionData.UserID) > 0 {
		user, err := p.core.DBAction.FindUserByID(r.Context(), string(sessionData.UserID))
		if err != nil {
			return nil, nil, err
		}
		passkeys, err := p.FindPasskeysByUserID(r.Context(), user.ID)
		if err != nil {
			return nil, nil, err
		}
		passkeyUser := newWebAuthnUser(p, user, passkeys)
		credential, err := p.webAuthn.FinishLogin(passkeyUser, *sessionData, r)
		if err != nil {
			return nil, nil, err
		}
		return user, credential, nil
	}

	passkeyUser, credential, err := p.webAuthn.FinishPasskeyLogin(p.discoverableUserHandler(r.Context()), *sessionData, r)
	if err != nil {
		return nil, nil, err
	}
	return passkeyUser.(*webAuthnUser).User(), credential, nil
}

func (p *passkeyPlugin) discoverableUserHandler(ctx context.Context) webauthn.DiscoverableUserHandler {
	return func(rawID, _ []byte) (user webauthn.User, err error) {
		if len(rawID) == 0 {
			return nil, ErrUnknownPasskey
		}

		passkey, err := p.FindPasskeyByCredentialID(ctx, base64.RawURLEncoding.EncodeToString(rawID))
		if err != nil {
			return nil, err
		}

		userModel, err := p.core.DBAction.FindUserByID(ctx, passkey.UserID)
		if err != nil {
			return nil, err
		}

		passkeys, err := p.FindPasskeysByUserID(ctx, userModel.ID)
		if err != nil {
			return nil, err
		}

		return newWebAuthnUser(p, userModel, passkeys), nil
	}
}
