package credentialpassword

import (
	"strings"

	"github.com/thecodearcher/aegis"
)

func (p *credentialPasswordPlugin) validatePassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}
	if len(password) < p.config.passwordMinLength {
		return ErrPasswordTooShort
	}
	if p.config.passwordRequireUppercase && !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return ErrPasswordRequiresUppercase
	}
	if p.config.passwordRequireNumbers && !strings.ContainsAny(password, "0123456789") {
		return ErrPasswordRequiresNumbers
	}
	if p.config.passwordRequireSymbols && !strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?") {
		return ErrPasswordRequiresSymbols
	}
	return nil
}

func (p *credentialPasswordPlugin) validateUsername(username string) error {
	if !p.config.enableUsername || (!p.config.usernameRequiredOnSignup && username == "") {
		return nil
	}

	if username == "" && p.config.usernameRequiredOnSignup {
		return ErrUsernameRequired
	}

	if len(username) < p.config.usernameMinLength {
		return ErrUsernameTooShort
	}

	if len(username) > p.config.usernameMaxLength {
		return ErrUsernameTooLong
	}

	if p.config.usernameValidationRegex != nil && !p.config.usernameValidationRegex.MatchString(username) {
		return ErrUsernameInvalidFormat
	}

	if p.config.usernameValidationFunc != nil {
		return p.config.usernameValidationFunc(username)
	}

	return nil
}

func (p *credentialPasswordPlugin) validateUser(user *aegis.User, additionalFields map[string]any) error {
	if user.Email == "" {
		return ErrEmailRequired
	}
	if user.Password == nil {
		return ErrPasswordRequired
	}
	if err := p.validatePassword(*user.Password); err != nil {
		return err
	}

	if additionalFields != nil {
		if usernameVal, ok := additionalFields["username"].(string); ok {
			if err := p.validateUsername(strings.TrimSpace(usernameVal)); err != nil {
				return err
			}
		}
	}

	return nil
}
