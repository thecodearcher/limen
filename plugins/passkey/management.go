package passkey

import (
	"context"
	"errors"
	"time"

	"github.com/thecodearcher/limen"
)

// ListPasskeys returns every credential registered for the given user,
// newest first.
func (p *passkeyPlugin) ListPasskeys(ctx context.Context, userID any) ([]Passkey, error) {
	recs, err := p.listForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Passkey, 0, len(recs))
	for _, rec := range recs {
		out = append(out, publicPasskey(rec))
	}
	return out, nil
}

// DeletePasskey removes a single credential. The caller must own the
// credential; cross-user deletion attempts return ErrForbidden so we
// don't leak the existence of credentials registered to other users.
func (p *passkeyPlugin) DeletePasskey(ctx context.Context, userID, passkeyID any) error {
	rec, err := p.findByID(ctx, passkeyID)
	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return ErrPasskeyNotFound
		}
		return err
	}
	if !sameID(rec.UserID, userID) {
		return ErrForbidden
	}
	return p.core.Delete(ctx, p.passkeySchema, []limen.Where{
		limen.Eq(p.passkeySchema.GetIDField(), rec.ID),
	})
}

// UpdatePasskey renames a credential. Ownership is enforced the same
// way as DeletePasskey.
func (p *passkeyPlugin) UpdatePasskey(ctx context.Context, userID, passkeyID any, newName string) (*Passkey, error) {
	rec, err := p.findByID(ctx, passkeyID)
	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, ErrPasskeyNotFound
		}
		return nil, err
	}
	if !sameID(rec.UserID, userID) {
		return nil, ErrForbidden
	}

	rec.Name = newName
	rec.UpdatedAt = time.Now()
	if err := p.core.Update(ctx, p.passkeySchema, rec, []limen.Where{
		limen.Eq(p.passkeySchema.GetIDField(), rec.ID),
	}); err != nil {
		return nil, err
	}

	refreshed, err := p.findByID(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	out := publicPasskey(refreshed)
	return &out, nil
}

// publicPasskey converts a storage record into the caller-facing shape
// returned by every passkey endpoint.
func publicPasskey(rec *PasskeyRecord) Passkey {
	return Passkey{
		ID:           rec.ID,
		UserID:       rec.UserID,
		Name:         rec.Name,
		CredentialID: rec.CredentialIDBase64,
		DeviceType:   rec.DeviceType,
		BackedUp:     rec.BackedUp,
		Transports:   rec.Transports,
		AAGUID:       rec.AAGUID,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}
}
