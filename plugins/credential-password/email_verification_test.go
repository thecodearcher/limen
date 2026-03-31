package credentialpassword

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestRequestEmailVerification(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "verify@test.com", "Password1")

	verification, err := plugin.RequestEmailVerification(context.Background(), &limen.User{Email: "verify@test.com"}, false)
	require.NoError(t, err)
	assert.NotEmpty(t, verification.Value)
}

func TestRequestEmailVerification_AlreadyVerified(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "verified@test.com", "Password1")

	verification, err := plugin.RequestEmailVerification(context.Background(), &limen.User{Email: "verified@test.com"}, false)
	require.NoError(t, err)

	err = plugin.VerifyEmail(context.Background(), verification.Value)
	require.NoError(t, err)

	_, err = plugin.RequestEmailVerification(context.Background(), &limen.User{Email: "verified@test.com"}, false)
	assert.ErrorIs(t, err, ErrEmailAlreadyVerified)
}

func TestVerifyEmail(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "doVerify@test.com", "Password1")

	verification, err := plugin.RequestEmailVerification(context.Background(), &limen.User{Email: "doVerify@test.com"}, false)
	require.NoError(t, err)

	err = plugin.VerifyEmail(context.Background(), verification.Value)
	require.NoError(t, err)
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	err := plugin.VerifyEmail(context.Background(), "bad-token")
	assert.ErrorIs(t, err, ErrResetTokenInvalid)
}

func TestVerifyEmail_TokenConsumed(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "consumed@test.com", "Password1")

	verification, err := plugin.RequestEmailVerification(context.Background(), &limen.User{Email: "consumed@test.com"}, false)
	require.NoError(t, err)

	err = plugin.VerifyEmail(context.Background(), verification.Value)
	require.NoError(t, err)

	err = plugin.VerifyEmail(context.Background(), verification.Value)
	assert.ErrorIs(t, err, ErrResetTokenInvalid, "reusing a consumed token should fail")
}

func TestRequestEmailVerification_NonExistentUser(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	_, err := plugin.RequestEmailVerification(context.Background(), &limen.User{Email: "ghost@test.com"}, false)
	assert.Error(t, err)
}

func TestSignUp_SendsVerificationEmail(t *testing.T) {
	t.Parallel()

	var sentEmail, sentToken string
	plugin := newTestLimenWithPlugin(t,
		WithRequireEmailVerification(true),
		WithSendVerificationEmail(func(email, token string) {
			sentEmail = email
			sentToken = token
		}),
	)

	pw := "Password1"
	_, err := plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    "sendemail@test.com",
		Password: &pw,
	}, nil)
	require.NoError(t, err)

	assert.Equal(t, "sendemail@test.com", sentEmail)
	assert.NotEmpty(t, sentToken)
}

