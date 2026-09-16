package passkey

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/thecodearcher/limen"
)

type RegisterPasskeyRequest struct {
	AuthenticatorAttachment string `json:"authenticator_attachment"`
	Context                 string `json:"context"`
}

type FinishRegistrationRequest struct {
	Name           string         `json:"name"`
	CreateSession  bool           `json:"create_session"`
	AdditionalData map[string]any `json:"-"`
}

type UpdatePasskeyRequest struct {
	Name string `json:"name"`
}

type challengeCookie struct {
	Session        webauthn.SessionData `json:"session"`
	Context        string               `json:"context,omitempty"`
	ReservedHandle string               `json:"reserved_handle,omitempty"`
	Intent         *RegistrationIntent  `json:"intent,omitempty"`
}

func (p *passkeyPlugin) BeginRegistration(r *http.Request, user *limen.User, request *RegisterPasskeyRequest) (*protocol.CredentialCreation, *challengeCookie, error) {
	var passkeys []*Passkey
	if user.ID != nil {
		found, err := p.FindPasskeysByUserID(r.Context(), user.ID)
		if err != nil {
			return nil, nil, err
		}
		passkeys = found
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

	ext, err := p.resolveExtensions(r, protocol.CreateCeremony)
	if err != nil {
		return nil, nil, err
	}
	opts = append(opts, webauthn.WithExtensions(webauthn.WithExtensionInputs(ext)))

	webAuthnUser := newWebAuthnUser(p, user, passkeys)
	if len(webAuthnUser.WebAuthnID()) == 0 {
		return nil, nil, ErrHandleMismatch
	}

	credentialCreation, sessionData, err := p.webAuthn.BeginRegistration(webAuthnUser, opts...)
	if err != nil {
		return nil, nil, err
	}

	return credentialCreation, &challengeCookie{Session: *sessionData}, nil
}

func (p *passkeyPlugin) BeginPublicRegistration(r *http.Request, request *RegisterPasskeyRequest) (*protocol.CredentialCreation, *challengeCookie, error) {
	registrationContext := strings.TrimSpace(request.Context)
	if registrationContext == "" {
		return nil, nil, ErrRegistrationContextRequired
	}

	intent, err := p.resolveRegistrationIntent(r, registrationContext)
	if err != nil {
		return nil, nil, err
	}

	reservedHandle, err := p.reserveRegistrationHandle(r.Context(), intent)
	if err != nil {
		return nil, nil, err
	}

	credentialCreation, cookie, err := p.BeginRegistration(r, p.userFromIntent(intent, reservedHandle), request)
	if err != nil {
		return nil, nil, err
	}

	cookie.Context = registrationContext
	cookie.ReservedHandle = reservedHandle
	cookie.Intent = intent
	return credentialCreation, cookie, nil
}

func (p *passkeyPlugin) FinishRegistration(r *http.Request, user *limen.User, request *FinishRegistrationRequest) (*Passkey, error) {
	cookie, err := p.getChallengeCookie(r)
	if err != nil {
		return nil, err
	}

	credential, err := p.finishRegistrationCeremony(r, user, cookie)
	if err != nil {
		return nil, err
	}
	return p.storePasskey(r.Context(), user, credential, request)
}

func (p *passkeyPlugin) FinishPublicRegistration(r *http.Request, request *FinishRegistrationRequest) (*limen.User, *Passkey, error) {
	cookie, err := p.getChallengeCookie(r)
	if err != nil {
		return nil, nil, err
	}
	if cookie.Intent == nil || cookie.ReservedHandle == "" {
		return nil, nil, ErrChallengeInvalid
	}

	credential, err := p.finishRegistrationCeremony(r, p.userFromIntent(cookie.Intent, cookie.ReservedHandle), cookie)
	if err != nil {
		return nil, nil, err
	}

	user, err := p.config.createRegistrationUser(r.Context(), r, cookie.Intent, cookie.ReservedHandle)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrRegistrationUserMissing
	}

	if !bytes.Equal(newWebAuthnUser(p, user, nil).WebAuthnID(), cookie.Session.UserID) {
		return nil, nil, ErrHandleMismatch
	}

	passkey, err := p.storePasskey(r.Context(), user, credential, request)
	if err != nil {
		return nil, nil, err
	}
	return user, passkey, nil
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

func (p *passkeyPlugin) FindPasskeyByCredentialID(ctx context.Context, credentialID string) (*Passkey, error) {
	passkey, err := p.core.FindOne(ctx, p.passkeySchema, []limen.Where{
		limen.Eq(p.passkeySchema.GetCredentialIDField(), credentialID),
	}, nil)
	if err != nil {
		return nil, err
	}
	return passkey.(*Passkey), nil
}

func (p *passkeyPlugin) ListPasskeys(ctx context.Context, user *limen.User, opts *limen.QueryOptions) (*limen.Page[*Passkey], error) {
	page, err := p.core.FindWithOptions(ctx, p.passkeySchema, []limen.Where{
		limen.Eq(p.passkeySchema.GetUserIDField(), user.ID),
	}, opts)
	if err != nil {
		return nil, err
	}

	return limen.MapPage[*Passkey](page), nil
}

func (p *passkeyPlugin) UpdatePasskey(ctx context.Context, user *limen.User, id any, request *UpdatePasskeyRequest) (*Passkey, error) {
	if err := p.ensureUserPasskeyExists(ctx, user, id); err != nil {
		return nil, err
	}

	passkey, err := p.core.UpdateAndReturn(ctx, p.passkeySchema, map[limen.SchemaField]any{
		PasskeySchemaNameField: request.Name,
	}, p.userPasskeyConditions(user, id), id)
	if err != nil {
		return nil, err
	}

	return passkey.(*Passkey), nil
}

func (p *passkeyPlugin) DeletePasskey(ctx context.Context, user *limen.User, id any) error {
	if err := p.ensureUserPasskeyExists(ctx, user, id); err != nil {
		return err
	}

	return p.core.Delete(ctx, p.passkeySchema, p.userPasskeyConditions(user, id))
}

func (p *passkeyPlugin) ensureUserPasskeyExists(ctx context.Context, user *limen.User, id any) error {
	exists, err := p.core.Exists(ctx, p.passkeySchema, p.userPasskeyConditions(user, id))
	if err != nil {
		return err
	}
	if !exists {
		return ErrPasskeyNotFound
	}

	return nil
}

func (p *passkeyPlugin) userPasskeyConditions(user *limen.User, id any) []limen.Where {
	return []limen.Where{
		limen.Eq(p.passkeySchema.GetIDField(), id),
		limen.Eq(p.passkeySchema.GetUserIDField(), user.ID),
	}
}

func (p *passkeyPlugin) finishRegistrationCeremony(r *http.Request, user *limen.User, cookie *challengeCookie) (*webauthn.Credential, error) {
	credential, err := p.webAuthn.FinishRegistration(newWebAuthnUser(p, user, nil), cookie.Session, r)
	if err != nil {
		return nil, toPasskeyError(err)
	}
	return credential, nil
}

func (p *passkeyPlugin) storePasskey(ctx context.Context, user *limen.User, credential *webauthn.Credential, request *FinishRegistrationRequest) (*Passkey, error) {
	payload := fromWebAuthnCredential(user.ID, credential)
	if request.Name != "" {
		payload.Name = &request.Name
	}

	passkey, err := p.core.CreateAndReturn(ctx, p.passkeySchema, payload, request.AdditionalData, PasskeySchemaCredentialIDField)
	if err != nil {
		return nil, toPasskeyError(err)
	}
	return passkey.(*Passkey), nil
}

func (p *passkeyPlugin) resolveRegistrationIntent(r *http.Request, registrationContext string) (*RegistrationIntent, error) {
	intent, err := p.config.prepareRegistration(r.Context(), r, registrationContext)
	if err != nil {
		return nil, err
	}
	if intent == nil || strings.TrimSpace(intent.Email) == "" {
		return nil, ErrRegistrationIntentInvalid
	}

	intent.Email = strings.TrimSpace(intent.Email)
	if intent.Name == "" {
		intent.Name = intent.Email
	}
	if intent.DisplayName == "" {
		intent.DisplayName = intent.Name
	}
	return intent, nil
}

func (p *passkeyPlugin) getChallengeCookie(r *http.Request) (*challengeCookie, error) {
	value, err := p.core.Cookies().GetSignedCookie(r, p.config.challengeCookieName)
	if err != nil {
		return nil, ErrChallengeMissing
	}

	var cookie challengeCookie
	if err := json.Unmarshal([]byte(value), &cookie); err != nil {
		return nil, ErrChallengeInvalid
	}
	if cookie.Session.Challenge == "" {
		return nil, ErrChallengeInvalid
	}
	return &cookie, nil
}

func (p *passkeyPlugin) setChallengeCookie(w http.ResponseWriter, cookie *challengeCookie) error {
	value, err := json.Marshal(cookie)
	if err != nil {
		return err
	}

	return p.core.Cookies().SetSignedCookie(w, p.config.challengeCookieName, string(value), 300)
}

func (p *passkeyPlugin) deleteChallengeCookie(w http.ResponseWriter) {
	p.core.Cookies().Delete(w, p.config.challengeCookieName)
}
