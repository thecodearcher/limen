package passkey

import (
	"context"
	"net/http"

	"github.com/thecodearcher/limen"
)

// storeChallenge persists a ceremony state into the verifications table
// and drops the lookup token into a signed cookie. The cookie is
// HttpOnly and short-lived, and the verification row is the only
// source of truth — we never trust client-supplied state when
// finishing a ceremony.
func (p *passkeyPlugin) storeChallenge(ctx context.Context, w http.ResponseWriter, state *challengeState) error {
	token, err := newVerificationToken()
	if err != nil {
		return err
	}
	value, err := encodeChallengeState(state)
	if err != nil {
		return err
	}
	if _, err := p.dbAction.CreateVerification(ctx, passkeyChallengeAction, token, value, p.config.challengeExpiration); err != nil {
		return err
	}
	maxAge := int(p.config.challengeExpiration.Seconds())
	return p.core.Cookies().SetSignedCookie(w, p.config.challengeCookieName, token, maxAge)
}

// consumeChallenge resolves the signed-cookie token to its persisted
// challenge state, then deletes the row so the response cannot be
// replayed. Any error (missing cookie, expired row, decode failure)
// surfaces as ErrChallengeNotFound to avoid leaking ceremony state.
func (p *passkeyPlugin) consumeChallenge(ctx context.Context, r *http.Request) (*challengeState, error) {
	token, err := p.core.Cookies().GetSignedCookie(r, p.config.challengeCookieName)
	if err != nil || token == "" {
		return nil, ErrChallengeNotFound
	}
	verification, err := p.dbAction.FindVerificationByAction(ctx, passkeyChallengeAction, token)
	if err != nil {
		return nil, ErrChallengeNotFound
	}
	// Single-use semantics: delete regardless of decode/verify outcome.
	if err := p.dbAction.DeleteVerification(ctx, verification.ID); err != nil {
		return nil, err
	}
	state, err := decodeChallengeState(verification.Value)
	if err != nil {
		return nil, ErrChallengeNotFound
	}
	return state, nil
}

func (p *passkeyPlugin) listForUser(ctx context.Context, userID any) ([]*PasskeyRecord, error) {
	rows, err := p.core.FindMany(ctx, p.passkeySchema, []limen.Where{
		limen.Eq(p.passkeySchema.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*PasskeyRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.(*PasskeyRecord))
	}
	return out, nil
}

func (p *passkeyPlugin) findByCredentialID(ctx context.Context, credentialIDB64 string) (*PasskeyRecord, error) {
	row, err := p.core.FindOne(ctx, p.passkeySchema, []limen.Where{
		limen.Eq(p.passkeySchema.GetCredentialIDField(), credentialIDB64),
	}, nil)
	if err != nil {
		return nil, err
	}
	return row.(*PasskeyRecord), nil
}

func (p *passkeyPlugin) findByID(ctx context.Context, id any) (*PasskeyRecord, error) {
	row, err := p.core.FindOne(ctx, p.passkeySchema, []limen.Where{
		limen.Eq(p.passkeySchema.GetIDField(), id),
	}, nil)
	if err != nil {
		return nil, err
	}
	return row.(*PasskeyRecord), nil
}
