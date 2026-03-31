package credentialpassword

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestComparePassword_NilHash(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	_, err := plugin.ComparePassword("anything", nil)
	assert.ErrorIs(t, err, ErrPasswordNotSet)
}

func TestRequestPasswordReset(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "reset@test.com", "Password1")

	verification, err := plugin.RequestPasswordReset(context.Background(), "reset@test.com")
	require.NoError(t, err)
	assert.NotEmpty(t, verification.Value, "should return a reset token")
}

func TestRequestPasswordReset_NonExistentEmail(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	_, err := plugin.RequestPasswordReset(context.Background(), "ghost@test.com")
	assert.ErrorIs(t, err, ErrEmailNotFound)
}

func TestResetPassword(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "doreset@test.com", "OldPassword1")

	verification, err := plugin.RequestPasswordReset(context.Background(), "doreset@test.com")
	require.NoError(t, err)

	err = plugin.ResetPassword(context.Background(), verification.Value, "NewPassword1")
	require.NoError(t, err)

	result, err := plugin.SignInWithCredentialAndPassword(context.Background(), "doreset@test.com", "NewPassword1")
	require.NoError(t, err)
	assert.Equal(t, "doreset@test.com", result.User.Email)

	_, err = plugin.SignInWithCredentialAndPassword(context.Background(), "doreset@test.com", "OldPassword1")
	assert.ErrorIs(t, err, ErrInvalidPassword, "old password should no longer work")
}

func TestResetPassword_InvalidToken(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	err := plugin.ResetPassword(context.Background(), "bad-token", "NewPassword1")
	assert.ErrorIs(t, err, ErrResetTokenInvalid)
}

func TestResetPassword_WeakNewPassword(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "weak@test.com", "Password1")

	verification, err := plugin.RequestPasswordReset(context.Background(), "weak@test.com")
	require.NoError(t, err)

	err = plugin.ResetPassword(context.Background(), verification.Value, "ab")
	assert.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestSetPassword(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	user := seedOAuthTestUser(t, plugin, "setpw@test.com")

	err := plugin.SetPassword(context.Background(), user, "NewPassword1", false)
	require.NoError(t, err)

	result, err := plugin.SignInWithCredentialAndPassword(context.Background(), "setpw@test.com", "NewPassword1")
	require.NoError(t, err)
	assert.Equal(t, "setpw@test.com", result.User.Email)
}

func TestSetPassword_AlreadySet(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	user := seedTestUser(t, plugin, "haspw@test.com", "Password1")

	err := plugin.SetPassword(context.Background(), user, "AnotherPassword1", false)
	assert.ErrorIs(t, err, ErrPasswordAlreadySet)
}

func TestSetPassword_RevokeSessions(t *testing.T) {
	t.Parallel()

	l, plugin := newTestLimenAndPlugin(t)
	user := seedOAuthTestUser(t, plugin, "revoke-set@test.com")
	session := limen.SeedTestSession(t, l, user.ID, user.Email)

	err := plugin.SetPassword(context.Background(), user, "NewPassword1", true)
	require.NoError(t, err)

	_, err = plugin.dbAction.FindSessionByToken(context.Background(), session.Token)
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)

	result, err := plugin.SignInWithCredentialAndPassword(context.Background(), "revoke-set@test.com", "NewPassword1")
	require.NoError(t, err)
	assert.Equal(t, "revoke-set@test.com", result.User.Email)
}

func TestSetPassword_WeakNewPassword(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	user := seedTestUser(t, plugin, "weakset@test.com", "Password1")

	oauthUser := &limen.User{ID: user.ID, Email: user.Email, Password: nil}
	err := plugin.SetPassword(context.Background(), oauthUser, "ab", false)
	assert.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestUpdatePassword(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	user := seedTestUser(t, plugin, "update@test.com", "OldPassword1")

	err := plugin.UpdatePassword(context.Background(), user, "OldPassword1", "NewPassword1", false)
	require.NoError(t, err)

	result, err := plugin.SignInWithCredentialAndPassword(context.Background(), "update@test.com", "NewPassword1")
	require.NoError(t, err)
	assert.Equal(t, "update@test.com", result.User.Email)

	_, err = plugin.SignInWithCredentialAndPassword(context.Background(), "update@test.com", "OldPassword1")
	assert.ErrorIs(t, err, ErrInvalidPassword)
}

func TestUpdatePassword_WrongCurrentPassword(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	user := seedTestUser(t, plugin, "wrongcur@test.com", "Password1")

	err := plugin.UpdatePassword(context.Background(), user, "WrongCurrent1", "NewPassword1", false)
	assert.ErrorIs(t, err, ErrInvalidCurrentPassword)
}

func TestUpdatePassword_RevokeSessions(t *testing.T) {
	t.Parallel()

	l, plugin := newTestLimenAndPlugin(t)
	user := seedTestUser(t, plugin, "revoke-upd@test.com", "OldPassword1")
	session := limen.SeedTestSession(t, l, user.ID, user.Email)

	err := plugin.UpdatePassword(context.Background(), user, "OldPassword1", "NewPassword1", true)
	require.NoError(t, err)

	_, err = plugin.dbAction.FindSessionByToken(context.Background(), session.Token)
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)

	result, err := plugin.SignInWithCredentialAndPassword(context.Background(), "revoke-upd@test.com", "NewPassword1")
	require.NoError(t, err)
	assert.Equal(t, "revoke-upd@test.com", result.User.Email)

	_, err = plugin.SignInWithCredentialAndPassword(context.Background(), "revoke-upd@test.com", "OldPassword1")
	assert.ErrorIs(t, err, ErrInvalidPassword)
}

func TestUpdatePassword_WeakNewPassword(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	user := seedTestUser(t, plugin, "weakupd@test.com", "Password1")

	err := plugin.UpdatePassword(context.Background(), user, "Password1", "ab", false)
	assert.ErrorIs(t, err, ErrPasswordTooShort)
}

func TestRequestPasswordReset_CallbackInvoked(t *testing.T) {
	t.Parallel()

	var sentEmail, sentToken string
	plugin := newTestLimenWithPlugin(t,
		WithSendPasswordResetEmail(func(email, token string) {
			sentEmail = email
			sentToken = token
		}),
	)
	seedTestUser(t, plugin, "callback@test.com", "Password1")

	_, err := plugin.RequestPasswordReset(context.Background(), "callback@test.com")
	require.NoError(t, err)

	assert.Equal(t, "callback@test.com", sentEmail)
	assert.NotEmpty(t, sentToken)
}

func TestResetPassword_OnSuccessCallback(t *testing.T) {
	t.Parallel()

	var callbackUser *limen.User
	plugin := newTestLimenWithPlugin(t,
		WithOnPasswordResetSuccess(func(ctx context.Context, user *limen.User) {
			callbackUser = user
		}),
	)
	seedTestUser(t, plugin, "onsuccess@test.com", "Password1")

	verification, err := plugin.RequestPasswordReset(context.Background(), "onsuccess@test.com")
	require.NoError(t, err)

	err = plugin.ResetPassword(context.Background(), verification.Value, "NewPassword1")
	require.NoError(t, err)

	require.NotNil(t, callbackUser)
	assert.Equal(t, "onsuccess@test.com", callbackUser.Email)
}

func TestRequestPasswordReset_CustomTokenGenerator(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t,
		WithGenerateResetToken(func(user *limen.User) (string, error) {
			return "custom-token-for-" + user.Email, nil
		}),
	)
	seedTestUser(t, plugin, "customtok@test.com", "Password1")

	verification, err := plugin.RequestPasswordReset(context.Background(), "customtok@test.com")
	require.NoError(t, err)
	assert.Equal(t, "custom-token-for-customtok@test.com", verification.Value)
}

func TestHashPassword_CustomHashFn(t *testing.T) {
	t.Parallel()

	customHash := func(pw string) (string, error) { return "custom:" + pw, nil }
	plugin := newTestLimenWithPlugin(t, WithHashFn(customHash))

	hash, err := plugin.HashPassword("test")
	require.NoError(t, err)
	assert.Equal(t, "custom:test", hash)
}

func TestComparePassword_CustomCompareFn(t *testing.T) {
	t.Parallel()

	customCompare := func(pw, hash string) (bool, error) { return hash == "custom:"+pw, nil }
	plugin := newTestLimenWithPlugin(t, WithCompareFn(customCompare))

	hash := "custom:mypassword"
	ok, err := plugin.ComparePassword("mypassword", &hash)
	require.NoError(t, err)
	assert.True(t, ok)
}
