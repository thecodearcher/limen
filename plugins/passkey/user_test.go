package passkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestWebAuthnIDUsesPublicID(t *testing.T) {
	t.Parallel()

	plugin, l := newTestPasskeyPlugin(t)
	user := limen.SeedTestUser(t, l, "handle@example.com")

	id := newWebAuthnUser(plugin, user, nil).WebAuthnID()
	require.NotEmpty(t, id)

	encoded, ok := plugin.core.EncodePublicID(plugin.core.Schema.User, user)
	require.True(t, ok)
	assert.Equal(t, []byte(encoded), id)
}
