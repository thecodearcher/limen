package passkey

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

// seedPasskey directly inserts a row into the passkeys table so tests
// can exercise List/Delete/Update without going through the WebAuthn
// registration ceremony (which needs a real browser).
func seedPasskey(t *testing.T, plugin *passkeyPlugin, userID any, credID, name string) *PasskeyRecord {
	t.Helper()
	now := time.Now()
	rec := &PasskeyRecord{
		UserID:             userID,
		Name:               name,
		CredentialIDBase64: credID,
		PublicKeyBase64:    "AAAA",
		Counter:            0,
		DeviceType:         "multiDevice",
		BackedUp:           true,
		Transports:         "internal",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	require.NoError(t, plugin.core.Create(context.Background(), plugin.passkeySchema, rec, nil))
	stored, err := plugin.findByCredentialID(context.Background(), credID)
	require.NoError(t, err)
	return stored
}

func TestListPasskeys_ReturnsCallerCredentialsOnly(t *testing.T) {
	t.Parallel()

	l, plugin := newTestLimenAndPlugin(t)
	alice := limen.SeedTestUser(t, l, "alice@test.com")
	bob := limen.SeedTestUser(t, l, "bob@test.com")

	seedPasskey(t, plugin, alice.ID, "cred-a-1", "Alice MacBook")
	seedPasskey(t, plugin, alice.ID, "cred-a-2", "Alice iPhone")
	seedPasskey(t, plugin, bob.ID, "cred-b-1", "Bob Laptop")

	got, err := plugin.ListPasskeys(context.Background(), alice.ID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
	for _, pk := range got {
		assert.Equal(t, alice.ID, pk.UserID)
	}
}

func TestDeletePasskey_OwnerCanDelete(t *testing.T) {
	t.Parallel()

	l, plugin := newTestLimenAndPlugin(t)
	alice := limen.SeedTestUser(t, l, "owner@test.com")
	rec := seedPasskey(t, plugin, alice.ID, "cred-owned", "Owned")

	require.NoError(t, plugin.DeletePasskey(context.Background(), alice.ID, rec.ID))

	list, err := plugin.ListPasskeys(context.Background(), alice.ID)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestDeletePasskey_NonOwnerForbidden(t *testing.T) {
	t.Parallel()

	l, plugin := newTestLimenAndPlugin(t)
	alice := limen.SeedTestUser(t, l, "owner@test.com")
	bob := limen.SeedTestUser(t, l, "intruder@test.com")
	rec := seedPasskey(t, plugin, alice.ID, "cred-protected", "Alice key")

	err := plugin.DeletePasskey(context.Background(), bob.ID, rec.ID)
	assert.ErrorIs(t, err, ErrForbidden)

	// Row must still exist.
	list, err := plugin.ListPasskeys(context.Background(), alice.ID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestDeletePasskey_NotFound(t *testing.T) {
	t.Parallel()

	l, plugin := newTestLimenAndPlugin(t)
	alice := limen.SeedTestUser(t, l, "u@test.com")

	err := plugin.DeletePasskey(context.Background(), alice.ID, 999999)
	assert.ErrorIs(t, err, ErrPasskeyNotFound)
}

func TestUpdatePasskey_RenamesOwnedCredential(t *testing.T) {
	t.Parallel()

	l, plugin := newTestLimenAndPlugin(t)
	alice := limen.SeedTestUser(t, l, "rename@test.com")
	rec := seedPasskey(t, plugin, alice.ID, "cred-rename", "Old name")

	updated, err := plugin.UpdatePasskey(context.Background(), alice.ID, rec.ID, "Brand new name")
	require.NoError(t, err)
	assert.Equal(t, "Brand new name", updated.Name)
}

func TestUpdatePasskey_NonOwnerForbidden(t *testing.T) {
	t.Parallel()

	l, plugin := newTestLimenAndPlugin(t)
	alice := limen.SeedTestUser(t, l, "owner@test.com")
	bob := limen.SeedTestUser(t, l, "intruder@test.com")
	rec := seedPasskey(t, plugin, alice.ID, "cred-protected2", "Original")

	_, err := plugin.UpdatePasskey(context.Background(), bob.ID, rec.ID, "Hijacked")
	assert.ErrorIs(t, err, ErrForbidden)
}
