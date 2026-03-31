package credentialpassword

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestPlugin(opts ...ConfigOption) *credentialPasswordPlugin {
	return New(opts...)
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()

	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "valid", password: "Password1", wantErr: nil},
		{name: "empty", password: "", wantErr: ErrPasswordRequired},
		{name: "too short", password: "Ab1", wantErr: ErrPasswordTooShort},
		{name: "no uppercase", password: "password1", wantErr: ErrPasswordRequiresUppercase},
		{name: "no numbers", password: "Password", wantErr: ErrPasswordRequiresNumbers},
		{name: "at minimum length", password: "Ab1x", wantErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := plugin.validatePassword(tt.password)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePassword_WithSymbolsRequired(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin(WithPasswordRequireSymbols(true))

	err := plugin.validatePassword("Password1")
	assert.ErrorIs(t, err, ErrPasswordRequiresSymbols)

	err = plugin.validatePassword("Password1!")
	assert.NoError(t, err)
}

func TestValidateUsername(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin(WithUsernameSupport(true), WithRequireUsernameOnSignUp(true))

	tests := []struct {
		name     string
		username string
		wantErr  error
	}{
		{name: "valid", username: "john_doe", wantErr: nil},
		{name: "valid with hyphen", username: "john-doe", wantErr: nil},
		{name: "empty required", username: "", wantErr: ErrUsernameRequired},
		{name: "too short", username: "ab", wantErr: ErrUsernameTooShort},
		{name: "at min length", username: "abc", wantErr: nil},
		{name: "invalid chars", username: "john doe!", wantErr: ErrUsernameInvalidFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := plugin.validateUsername(tt.username)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUsername_NotEnabled(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	err := plugin.validateUsername("anything")
	assert.NoError(t, err)
}

func TestValidateUsername_OptionalWhenNotRequired(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin(WithUsernameSupport(true), WithRequireUsernameOnSignUp(false))
	err := plugin.validateUsername("")
	assert.NoError(t, err, "empty username should be ok when not required")
}

func TestValidateUsername_TooLong(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin(WithUsernameSupport(true))
	longUsername := strings.Repeat("a", defaultMaxUsernameLength+1)
	err := plugin.validateUsername(longUsername)
	assert.ErrorIs(t, err, ErrUsernameTooLong)
}

func TestValidateUsername_CustomRegex(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin(
		WithUsernameSupport(true),
		WithUsernameValidationRegex(regexp.MustCompile(`^[a-z]+$`)),
	)

	err := plugin.validateUsername("lowercase")
	assert.NoError(t, err)

	err = plugin.validateUsername("HasUpper")
	assert.ErrorIs(t, err, ErrUsernameInvalidFormat)
}

func TestValidateUsername_CustomValidationFunc(t *testing.T) {
	t.Parallel()

	blocked := errors.New("username is blocked")
	plugin := newTestPlugin(
		WithUsernameSupport(true),
		func(c *config) {
			c.usernameValidationFunc = func(username string) error {
				if username == "blocked" {
					return blocked
				}
				return nil
			}
		},
	)

	err := plugin.validateUsername("allowed")
	assert.NoError(t, err)

	err = plugin.validateUsername("blocked")
	assert.ErrorIs(t, err, blocked)
}
