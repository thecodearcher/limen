package limen

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLimenWithEmailVerification(t *testing.T, opts ...EmailVerificationConfigOption) *Limen {
	t.Helper()

	l, err := New(&Config{
		BaseURL:           "http://localhost:8080",
		Database:          newTestMemoryAdapter(t),
		Secret:            TestSecret,
		EmailVerification: DefaultEmailVerification(opts...),
	})
	require.NoError(t, err)
	return l
}

func TestRequestEmailVerification(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithEmailVerification(t)
	SeedTestUser(t, l, "verify@test.com")

	verification, err := l.RequestEmailVerification(context.Background(), &User{Email: "verify@test.com"}, false)
	require.NoError(t, err)
	assert.NotEmpty(t, verification.Value)
}

func TestRequestEmailVerification_AlreadyVerified(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithEmailVerification(t)
	SeedTestUser(t, l, "verified@test.com")

	verification, err := l.RequestEmailVerification(context.Background(), &User{Email: "verified@test.com"}, false)
	require.NoError(t, err)

	err = l.VerifyEmail(context.Background(), verification.Value)
	require.NoError(t, err)

	_, err = l.RequestEmailVerification(context.Background(), &User{Email: "verified@test.com"}, false)
	assert.ErrorIs(t, err, ErrEmailAlreadyVerified)
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithEmailVerification(t)
	err := l.VerifyEmail(context.Background(), "bad-token")
	assert.ErrorIs(t, err, ErrEmailVerificationTokenInvalid)
}

func TestVerifyEmail_TokenConsumed(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithEmailVerification(t)
	SeedTestUser(t, l, "consumed@test.com")

	verification, err := l.RequestEmailVerification(context.Background(), &User{Email: "consumed@test.com"}, false)
	require.NoError(t, err)

	err = l.VerifyEmail(context.Background(), verification.Value)
	require.NoError(t, err)

	err = l.VerifyEmail(context.Background(), verification.Value)
	assert.ErrorIs(t, err, ErrEmailVerificationTokenInvalid, "reusing a consumed token should fail")
}

func TestRequestEmailVerification_SendsEmail(t *testing.T) {
	t.Parallel()

	var sentEmail, sentToken string
	l := newTestLimenWithEmailVerification(t,
		WithSendEmailVerificationMail(func(email, token string) {
			sentEmail = email
			sentToken = token
		}),
	)
	SeedTestUser(t, l, "send@test.com")

	_, err := l.RequestEmailVerification(context.Background(), &User{Email: "send@test.com"}, true)
	require.NoError(t, err)
	assert.Equal(t, "send@test.com", sentEmail)
	assert.NotEmpty(t, sentToken)
}

func TestRequestEmailVerification_SkipsSendWhenFlagFalse(t *testing.T) {
	t.Parallel()

	called := false
	l := newTestLimenWithEmailVerification(t,
		WithSendEmailVerificationMail(func(_, _ string) {
			called = true
		}),
	)
	SeedTestUser(t, l, "nosend@test.com")

	_, err := l.RequestEmailVerification(context.Background(), &User{Email: "nosend@test.com"}, false)
	require.NoError(t, err)
	assert.False(t, called, "send callback should not be invoked when shouldSendEmail is false")
}
