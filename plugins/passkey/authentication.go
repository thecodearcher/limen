package passkey

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/thecodearcher/limen"
)

// BeginAuthentication kicks off a passkey sign-in. We use the
// "discoverable credential" flow: no user identifier is required up
// front; the client's authenticator presents a credential picker and
// the server resolves the user from the returned credential ID. This
// is what every modern passkey UX expects.
func (p *passkeyPlugin) BeginAuthentication(ctx context.Context, w http.ResponseWriter) (*protocol.CredentialAssertion, error) {
	options, sessionData, err := p.webauthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, err
	}

	if err := p.storeChallenge(ctx, w, &challengeState{
		Ceremony: CeremonyAuthentication,
		Session:  *sessionData,
	}); err != nil {
		return nil, err
	}

	return options, nil
}

// FinishAuthentication validates the assertion returned by the
// client's authenticator, looks up the registered credential, updates
// its sign counter, and returns the authenticated user.
func (p *passkeyPlugin) FinishAuthentication(ctx context.Context, r *http.Request, body io.Reader) (*limen.AuthenticationResult, error) {
	state, err := p.consumeChallenge(ctx, r)
	if err != nil {
		return nil, err
	}
	if state.Ceremony != CeremonyAuthentication {
		return nil, ErrChallengeMismatch
	}

	parsed, err := protocol.ParseCredentialRequestResponseBody(body)
	if err != nil {
		return nil, ErrInvalidResponse
	}

	// Discoverable login means the authenticator picked which credential
	// to present; we resolve the user via the handler the library calls
	// with the rawID + userHandle from the assertion.
	var resolvedOwner *limen.User
	var resolvedRecord *PasskeyRecord
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		credIDB64 := base64.RawURLEncoding.EncodeToString(rawID)
		rec, err := p.findByCredentialID(ctx, credIDB64)
		if err != nil {
			return nil, err
		}
		owner, err := p.findUserByID(ctx, rec.UserID)
		if err != nil {
			return nil, err
		}
		waUser, err := p.toWebAuthnUser(owner, []*PasskeyRecord{rec})
		if err != nil {
			return nil, err
		}
		resolvedOwner = owner
		resolvedRecord = rec
		return waUser, nil
	}

	credential, err := p.webauthn.ValidateDiscoverableLogin(handler, state.Session, parsed)
	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, ErrPasskeyNotFound
		}
		return nil, ErrAuthenticationFailed
	}
	if resolvedOwner == nil || resolvedRecord == nil {
		return nil, ErrAuthenticationFailed
	}
	rec := resolvedRecord
	owner := resolvedOwner

	// Update sign counter / backup state on every successful auth.
	rec.Counter = int64(credential.Authenticator.SignCount)
	rec.BackedUp = credential.Flags.BackupState
	if err := p.core.Update(ctx, p.passkeySchema, rec, []limen.Where{
		limen.Eq(p.passkeySchema.GetIDField(), rec.ID),
	}); err != nil {
		return nil, err
	}

	return &limen.AuthenticationResult{User: owner}, nil
}

func (p *passkeyPlugin) findUserByID(ctx context.Context, id any) (*limen.User, error) {
	res, err := p.core.FindOne(ctx, p.userSchema, []limen.Where{
		limen.Eq(p.userSchema.GetIDField(), id),
	}, nil)
	if err != nil {
		return nil, err
	}
	return res.(*limen.User), nil
}

// discoverableHandler matches the signature used by ValidateDiscoverable
// in newer go-webauthn versions; we keep it for completeness even
// though the discoverable login path used here resolves the user via
// FinishAuthentication's explicit credential lookup.
//
//nolint:unused // kept for future use when go-webauthn formalizes the API
func (p *passkeyPlugin) discoverableHandler(ctx context.Context) webauthn.DiscoverableUserHandler {
	return func(rawID, userHandle []byte) (webauthn.User, error) {
		credIDB64 := base64.RawURLEncoding.EncodeToString(rawID)
		rec, err := p.findByCredentialID(ctx, credIDB64)
		if err != nil {
			return nil, err
		}
		owner, err := p.findUserByID(ctx, rec.UserID)
		if err != nil {
			return nil, err
		}
		return p.toWebAuthnUser(owner, []*PasskeyRecord{rec})
	}
}
