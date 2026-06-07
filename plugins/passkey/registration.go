package passkey

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/thecodearcher/limen"
)

// BeginRegistration creates the WebAuthn registration options the
// caller's client must hand to navigator.credentials.create(). It also
// persists the per-ceremony challenge state and sets a signed cookie
// the corresponding FinishRegistration call will use to look it up.
//
// nameHint, when non-empty, is stored on the resulting PasskeyRecord
// so users can manage credentials by a human-friendly label (e.g.
// "MacBook Air — Touch ID").
func (p *passkeyPlugin) BeginRegistration(ctx context.Context, r *http.Request, w http.ResponseWriter, nameHint string) (*protocol.CredentialCreation, error) {
	session, err := requireValidatedSession(r)
	if err != nil {
		return nil, err
	}
	user := session.User

	existing, err := p.listForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	waUser, err := p.toWebAuthnUser(user, existing)
	if err != nil {
		return nil, err
	}

	options, sessionData, err := p.webauthn.BeginRegistration(
		waUser,
		webauthn.WithExclusions(excludedDescriptors(existing)),
	)
	if err != nil {
		return nil, fmt.Errorf("passkey: begin registration: %w", err)
	}

	if err := p.storeChallenge(ctx, w, &challengeState{
		Ceremony: CeremonyRegistration,
		UserID:   user.ID,
		UserName: nameHint,
		Session:  *sessionData,
	}); err != nil {
		return nil, err
	}

	return options, nil
}

// FinishRegistration validates the WebAuthn registration response sent
// by the client, persists the new credential, and returns the public
// passkey record. The challenge state is consumed regardless of
// outcome so the same response cannot be replayed.
func (p *passkeyPlugin) FinishRegistration(ctx context.Context, r *http.Request, body io.Reader) (*Passkey, error) {
	session, err := requireValidatedSession(r)
	if err != nil {
		return nil, err
	}
	state, err := p.consumeChallenge(ctx, r)
	if err != nil {
		return nil, err
	}
	if state.Ceremony != CeremonyRegistration {
		return nil, ErrChallengeMismatch
	}
	if !sameID(state.UserID, session.User.ID) {
		return nil, ErrForbidden
	}

	parsed, err := protocol.ParseCredentialCreationResponseBody(body)
	if err != nil {
		return nil, ErrInvalidResponse
	}

	existing, err := p.listForUser(ctx, session.User.ID)
	if err != nil {
		return nil, err
	}
	waUser, err := p.toWebAuthnUser(session.User, existing)
	if err != nil {
		return nil, err
	}

	credential, err := p.webauthn.CreateCredential(waUser, state.Session, parsed)
	if err != nil {
		return nil, ErrRegistrationFailed
	}

	credIDB64 := base64.RawURLEncoding.EncodeToString(credential.ID)
	for _, e := range existing {
		if e.CredentialIDBase64 == credIDB64 {
			return nil, ErrPasskeyAlreadyExists
		}
	}

	transports := make([]string, 0, len(credential.Transport))
	for _, t := range credential.Transport {
		if s := strings.TrimSpace(string(t)); s != "" {
			transports = append(transports, s)
		}
	}

	now := time.Now()
	rec := &PasskeyRecord{
		UserID:             session.User.ID,
		Name:               state.UserName,
		CredentialIDBase64: credIDB64,
		PublicKeyBase64:    base64.StdEncoding.EncodeToString(credential.PublicKey),
		Counter:            int64(credential.Authenticator.SignCount),
		DeviceType:         deviceTypeFromFlags(credential.Flags),
		BackedUp:           credential.Flags.BackupState,
		Transports:         strings.Join(transports, ","),
		AAGUID:             base64.StdEncoding.EncodeToString(credential.Authenticator.AAGUID),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := p.core.Create(ctx, p.passkeySchema, rec, nil); err != nil {
		return nil, err
	}

	stored, err := p.findByCredentialID(ctx, credIDB64)
	if err != nil {
		return nil, err
	}
	pk := publicPasskey(stored)
	return &pk, nil
}

// requireValidatedSession pulls the active session out of the request
// context. The ProtectedXxx route helpers wire the session middleware
// for us — this just returns a typed error when the middleware was not
// applied (e.g., during direct programmatic calls).
func requireValidatedSession(r *http.Request) (*limen.ValidatedSession, error) {
	if r == nil {
		return nil, ErrSessionRequired
	}
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil || session == nil || session.User == nil {
		return nil, ErrSessionRequired
	}
	return session, nil
}

func (p *passkeyPlugin) toWebAuthnUser(user *limen.User, recs []*PasskeyRecord) (*webAuthnUser, error) {
	creds := make([]webauthn.Credential, 0, len(recs))
	for _, rec := range recs {
		c, err := recordToCredential(rec)
		if err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	name := user.Email
	if name == "" {
		name = fmt.Sprintf("user-%v", user.ID)
	}
	return &webAuthnUser{
		id:          userHandleFor(user.ID),
		name:        name,
		displayName: name,
		credentials: creds,
	}, nil
}

func excludedDescriptors(recs []*PasskeyRecord) []protocol.CredentialDescriptor {
	out := make([]protocol.CredentialDescriptor, 0, len(recs))
	for _, rec := range recs {
		c, err := recordToCredential(rec)
		if err != nil {
			continue
		}
		out = append(out, protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: c.ID,
			Transport:    c.Transport,
		})
	}
	return out
}

// sameID compares two id values typed as any. Limen adapters return
// int64, string, or driver-specific types depending on the database,
// so we compare on the string form.
func sameID(a, b any) bool {
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func deviceTypeFromFlags(f webauthn.CredentialFlags) string {
	if f.BackupEligible {
		return "multiDevice"
	}
	return "singleDevice"
}

