package credentialpassword

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestSignUp(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	pw := "Password1"

	result, err := plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    "new@test.com",
		Password: &pw,
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, "new@test.com", result.User.Email)
	assert.NotNil(t, result.User.Password, "password should be hashed and stored")
	assert.NotEqual(t, pw, *result.User.Password, "stored password should be hashed")
}

func TestSignUp_DuplicateEmail(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "dup@test.com", "Password1")

	pw := "Password1"
	_, err := plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    "dup@test.com",
		Password: &pw,
	}, nil)

	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestSignUp_WithUsername(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t, WithUsernameSupport(true))
	pw := "Password1"

	result, err := plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    "user@test.com",
		Password: &pw,
	}, map[string]any{"username": "johndoe"})

	require.NoError(t, err)
	assert.Equal(t, "user@test.com", result.User.Email)
}

func TestSignUp_DuplicateUsername(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t, WithUsernameSupport(true))
	pw := "Password1"

	_, err := plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    "first@test.com",
		Password: &pw,
	}, map[string]any{"username": "taken"})
	require.NoError(t, err)

	_, err = plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    "second@test.com",
		Password: &pw,
	}, map[string]any{"username": "taken"})
	assert.ErrorIs(t, err, ErrUsernameAlreadyExists)
}

func TestSignUp_ValidationErrors(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	pw := "Password1"
	weakPw := "abc"

	tests := []struct {
		name    string
		email   string
		pw      *string
		wantErr error
	}{
		{"missing email", "", &pw, ErrEmailRequired},
		{"nil password", "a@test.com", nil, ErrPasswordRequired},
		{"weak password", "a@test.com", &weakPw, ErrPasswordTooShort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
				Email:    tt.email,
				Password: tt.pw,
			}, nil)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSignIn(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "signin@test.com", "Password1")

	result, err := plugin.SignInWithCredentialAndPassword(context.Background(), "signin@test.com", "Password1")
	require.NoError(t, err)
	assert.Equal(t, "signin@test.com", result.User.Email)
}

func TestSignIn_Errors(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	seedTestUser(t, plugin, "exists@test.com", "Password1")

	tests := []struct {
		name       string
		credential string
		password   string
		wantErr    error
	}{
		{"wrong password", "exists@test.com", "WrongPassword1", ErrInvalidPassword},
		{"non-existent user", "ghost@test.com", "Password1", limen.ErrRecordNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := plugin.SignInWithCredentialAndPassword(context.Background(), tt.credential, tt.password)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSignIn_WithUsername(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t, WithUsernameSupport(true))
	pw := "Password1"

	_, err := plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    "uname@test.com",
		Password: &pw,
	}, map[string]any{"username": "myuser"})
	require.NoError(t, err)

	result, err := plugin.SignInWithCredentialAndPassword(context.Background(), "myuser", "Password1")
	require.NoError(t, err)
	assert.Equal(t, "uname@test.com", result.User.Email)
}

func TestCheckUsernameAvailability(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t, WithUsernameSupport(true))
	pw := "Password1"

	_, err := plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    "avail@test.com",
		Password: &pw,
	}, map[string]any{"username": "taken_user"})
	require.NoError(t, err)

	available, err := plugin.CheckUsernameAvailability(context.Background(), "fresh_user")
	require.NoError(t, err)
	assert.True(t, available)

	available, err = plugin.CheckUsernameAvailability(context.Background(), "taken_user")
	require.NoError(t, err)
	assert.False(t, available)
}

func TestCheckUsernameAvailability_Disabled(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	_, err := plugin.CheckUsernameAvailability(context.Background(), "any")
	assert.ErrorIs(t, err, ErrUsernameNotEnabled)
}

func TestCheckUsernameAvailability_InvalidFormat(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t, WithUsernameSupport(true))

	tests := []struct {
		name     string
		username string
		wantErr  error
	}{
		{"too short", "ab", ErrUsernameTooShort},
		{"invalid chars", "bad user!", ErrUsernameInvalidFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := plugin.CheckUsernameAvailability(context.Background(), tt.username)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestFindUserByUsername(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t, WithUsernameSupport(true))
	pw := "Password1"
	_, err := plugin.SignUpWithCredentialAndPassword(context.Background(), &limen.User{
		Email:    "findme@test.com",
		Password: &pw,
	}, map[string]any{"username": "findable"})
	require.NoError(t, err)

	user, err := plugin.FindUserByUsername(context.Background(), "findable")
	require.NoError(t, err)
	assert.Equal(t, "findme@test.com", user.Email)
}

func TestFindUserByUsername_Disabled(t *testing.T) {
	t.Parallel()

	plugin := newTestLimenWithPlugin(t)
	_, err := plugin.FindUserByUsername(context.Background(), "any")
	assert.ErrorIs(t, err, ErrUsernameNotEnabled)
}
