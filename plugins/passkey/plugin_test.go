package passkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestInitializeRequiresHandleMode(t *testing.T) {
	t.Parallel()

	_, core := limen.NewTestLimen(t)
	err := New().Initialize(core)
	require.ErrorIs(t, err, ErrHandleModeRequired)
}

func TestInitializeRequiresHooksWhenPublicRegistration(t *testing.T) {
	t.Parallel()

	schema := limen.NewDefaultSchemaConfig(limen.WithPublicIDs(testPasskeyPublicIDConfig()))
	_, core := limen.NewTestLimenWithSchema(t, schema)
	err := New(WithRequireSession(false)).Initialize(core)
	require.ErrorIs(t, err, ErrRegistrationHooksRequired)
}

func TestInitializeAllowsInternalHandleWithoutIDGenerator(t *testing.T) {
	t.Parallel()

	_, core := limen.NewTestLimen(t)
	plugin := New(WithAllowInternalUserIDAsHandle(true))
	require.NoError(t, plugin.Initialize(core))

	got, err := plugin.reserveRegistrationHandle(t.Context(), &RegistrationIntent{ID: "explicit-id"})
	require.NoError(t, err)
	assert.Equal(t, "explicit-id", got)

	_, err = plugin.reserveRegistrationHandle(t.Context(), &RegistrationIntent{})
	require.ErrorIs(t, err, ErrIDGeneratorRequired)
}
