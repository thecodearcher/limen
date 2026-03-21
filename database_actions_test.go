package limen

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseActionHelper_VerificationLifecycle(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	dbAction := l.core.DBAction

	verification, err := dbAction.CreateVerification(ctx, "password_reset", "user@test.com", "token-123", 24*time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "token-123", verification.Value)
	assert.Equal(t, "password_reset::user@test.com", verification.Subject)

	found, err := dbAction.FindVerificationByAction(ctx, "password_reset", "user@test.com")
	require.NoError(t, err)
	assert.Equal(t, "token-123", found.Value)

	valid, err := dbAction.FindValidVerificationByToken(ctx, "token-123")
	require.NoError(t, err)
	assert.Equal(t, "password_reset::user@test.com", valid.Subject)

	err = dbAction.VerifyVerificationToken(ctx, "token-123", "password_reset", "user@test.com")
	require.NoError(t, err)

	_, err = dbAction.FindValidVerificationByToken(ctx, "token-123")
	assert.ErrorIs(t, err, ErrRecordNotFound, "token should be deleted after verification")
}

func TestDatabaseActionHelper_VerifyVerificationToken_InvalidToken(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()

	err := l.core.DBAction.VerifyVerificationToken(ctx, "bad-token", "password_reset", "user@test.com")
	assert.ErrorIs(t, err, ErrVerificationTokenInvalid)
}

func TestDatabaseActionHelper_VerifyVerificationToken_WrongAction(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	dbAction := l.core.DBAction

	_, err := dbAction.CreateVerification(ctx, "password_reset", "user@test.com", "token-456", 24*time.Hour)
	require.NoError(t, err)

	err = dbAction.VerifyVerificationToken(ctx, "token-456", "email_verify", "user@test.com")
	assert.ErrorIs(t, err, ErrVerificationTokenInvalid)
}

func TestDatabaseActionHelper_CreateVerification_EmptyIdentifier(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()

	_, err := l.core.DBAction.CreateVerification(ctx, "action", "", "token", time.Hour)
	assert.Error(t, err)
}

func TestDatabaseActionHelper_SessionLifecycle(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	dbAction := l.core.DBAction

	userID := seedUser(t, l, "session-lifecycle@test.com")

	now := time.Now()
	err := dbAction.CreateSession(ctx, &Session{
		Token:      "sess-token-1",
		UserID:     userID,
		CreatedAt:  now,
		ExpiresAt:  now.Add(7 * 24 * time.Hour),
		LastAccess: now,
	}, nil)
	require.NoError(t, err)

	sess, err := dbAction.FindSessionByToken(ctx, "sess-token-1")
	require.NoError(t, err)
	assert.Equal(t, "sess-token-1", sess.Token)
	assert.Equal(t, userID, sess.UserID)

	sessions, err := dbAction.ListSessionsByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	err = dbAction.DeleteSessionByToken(ctx, "sess-token-1")
	require.NoError(t, err)

	_, err = dbAction.FindSessionByToken(ctx, "sess-token-1")
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

func TestDatabaseActionHelper_DeleteSessionByUserID(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	ctx := context.Background()
	dbAction := l.core.DBAction

	userID := seedUser(t, l, "del-all@test.com")

	now := time.Now()
	for range 3 {
		err := dbAction.CreateSession(ctx, &Session{
			Token:      GenerateRandomString(32),
			UserID:     userID,
			CreatedAt:  now,
			ExpiresAt:  now.Add(7 * 24 * time.Hour),
			LastAccess: now,
		}, nil)
		require.NoError(t, err)
	}

	err := dbAction.DeleteSessionByUserID(ctx, userID)
	require.NoError(t, err)

	sessions, err := dbAction.ListSessionsByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}
