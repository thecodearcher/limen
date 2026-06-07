package passkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-webauthn/webauthn/webauthn"
)

func TestEncodeDecodeChallengeState_RoundTrips(t *testing.T) {
	t.Parallel()

	state := &challengeState{
		Ceremony: CeremonyRegistration,
		UserID:   "user-123",
		UserName: "MacBook Air",
		Session: webauthn.SessionData{
			Challenge:        "challenge-bytes",
			UserID:           []byte("limen-user-user-123"),
			UserVerification: "preferred",
		},
	}

	encoded, err := encodeChallengeState(state)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)

	got, err := decodeChallengeState(encoded)
	require.NoError(t, err)
	assert.Equal(t, state.Ceremony, got.Ceremony)
	assert.Equal(t, state.UserName, got.UserName)
	assert.Equal(t, state.Session.Challenge, got.Session.Challenge)
}

func TestNewVerificationToken_Unique(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		tok, err := newVerificationToken()
		require.NoError(t, err)
		assert.NotEmpty(t, tok)
		assert.False(t, seen[tok], "duplicate token returned: %s", tok)
		seen[tok] = true
	}
}

func TestSameID_StringifiedComparison(t *testing.T) {
	t.Parallel()

	assert.True(t, sameID(int64(7), int64(7)))
	assert.True(t, sameID(int64(7), "7"), "id comparison should be type-insensitive")
	assert.False(t, sameID(int64(7), int64(8)))
	assert.False(t, sameID(nil, int64(7)))
}
