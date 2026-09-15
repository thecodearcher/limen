package passkey

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/thecodearcher/limen"
)

type RegisterPasskeyRequest struct {
	AuthenticatorAttachment string `json:"authenticator_attachment"`
}

type FinishRegistrationRequest struct {
	Name           string         `json:"name"`
	AdditionalData map[string]any `json:"-"`
}

func (p *passkeyPlugin) BeginRegistration(ctx context.Context, user *limen.User, request *RegisterPasskeyRequest) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	passkeys, err := p.FindPasskeysByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	excludedPasskeys := make([]protocol.CredentialDescriptor, 0, len(passkeys))
	for _, passkey := range passkeys {
		credential := toWebAuthnCredential(passkey)
		excludedPasskeys = append(excludedPasskeys, credential.Descriptor())
	}

	opts := []webauthn.RegistrationOption{
		webauthn.WithExclusions(excludedPasskeys),
	}
	if request.AuthenticatorAttachment != "" {
		selection := p.config.authenticatorSelection
		selection.AuthenticatorAttachment = protocol.AuthenticatorAttachment(request.AuthenticatorAttachment)
		opts = append(opts, webauthn.WithAuthenticatorSelection(selection))
	}
	webAuthnUser := newWebAuthnUser(p.core, user, passkeys)
	credentialCreation, sessionData, err := p.webAuthn.BeginRegistration(webAuthnUser, opts...)
	if err != nil {
		return nil, nil, err
	}

	return credentialCreation, sessionData, nil
}

func (p *passkeyPlugin) FinishRegistration(r *http.Request, user *limen.User, request *FinishRegistrationRequest) (*Passkey, error) {
	sessionData, err := p.getSessionDataFromCookie(r)
	if err != nil {
		return nil, err
	}
	webAuthnUser := newWebAuthnUser(p.core, user, nil)

	credential, err := p.webAuthn.FinishRegistration(webAuthnUser, *sessionData, r)
	if err != nil {
		return nil, toPasskeyError(err)
	}

	payload := fromWebAuthnCredential(user.ID, credential)

	if request.Name != "" {
		payload.Name = &request.Name
	}

	passkey, err := p.core.CreateAndReturn(r.Context(), p.passkeySchema, payload, request.AdditionalData, PasskeySchemaCredentialIDField)
	if err != nil {
		return nil, toPasskeyError(err)
	}
	return passkey.(*Passkey), nil
}

func (p *passkeyPlugin) FindPasskeysByUserID(ctx context.Context, userID any) ([]*Passkey, error) {
	passkeys, err := p.core.FindMany(ctx, p.passkeySchema, []limen.Where{
		limen.Eq(p.passkeySchema.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}

	return limen.MapToSliceOfType[*Passkey](passkeys), nil
}

func (p *passkeyPlugin) getSessionDataFromCookie(r *http.Request) (*webauthn.SessionData, error) {
	value, err := p.core.Cookies().GetSignedCookie(r, p.config.challengeCookieName)
	if err != nil {
		return nil, ErrChallengeMissing
	}

	var sessionData webauthn.SessionData
	err = json.Unmarshal([]byte(value), &sessionData)
	if err != nil {
		return nil, ErrChallengeInvalid
	}
	return &sessionData, nil
}

func (p *passkeyPlugin) setSessionDataToCookie(w http.ResponseWriter, sessionData *webauthn.SessionData) error {
	value, err := json.Marshal(sessionData)
	if err != nil {
		return err
	}

	return p.core.Cookies().SetSignedCookie(w, p.config.challengeCookieName, string(value), 300)
}

func (p *passkeyPlugin) deleteSessionDataFromCookie(w http.ResponseWriter) {
	p.core.Cookies().Delete(w, p.config.challengeCookieName)
}
