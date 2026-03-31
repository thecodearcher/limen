package credentialpassword

import (
	"context"
	"strings"

	"github.com/thecodearcher/limen"
)

// SignInWithCredentialAndPassword authenticates a user with either email or username (if enabled) and password.
// The credential parameter can be either an email address or a username.
// Returns an AuthenticationResult on success, or an error if authentication fails.
func (p *credentialPasswordPlugin) SignInWithCredentialAndPassword(ctx context.Context, credential string, password string) (*limen.AuthenticationResult, error) {
	isUsername := !strings.Contains(credential, "@") && p.config.enableUsername

	var user *limen.User
	var err error

	if isUsername {
		user, err = p.FindUserByUsername(ctx, credential)
	} else {
		user, err = p.dbAction.FindUserByEmail(ctx, credential)
	}

	if err != nil {
		// hash the password to avoid timing attacks when the user is not found
		// this allows constant response time for both valid and invalid credentials
		_, _ = p.HashPassword(password)
		return nil, err
	}

	return p.authenticateUser(user, password)
}

func (p *credentialPasswordPlugin) authenticateUser(user *limen.User, password string) (*limen.AuthenticationResult, error) {
	isValid, err := p.ComparePassword(password, user.Password)
	if err != nil {
		return nil, err
	}

	if !isValid {
		return nil, ErrInvalidPassword
	}

	return &limen.AuthenticationResult{User: user}, nil
}

// FindUserByUsername finds a user by their username.
// Returns an error if username support is not enabled or if the user is not found.
func (p *credentialPasswordPlugin) FindUserByUsername(ctx context.Context, username string) (*limen.User, error) {
	if !p.config.enableUsername {
		return nil, ErrUsernameNotEnabled
	}

	user, err := p.core.FindOne(ctx, p.userSchema, []limen.Where{
		limen.Eq(p.getUsernameField(), username),
	}, nil)

	if err != nil {
		return nil, err
	}

	return user.(*limen.User), nil
}

// SignUpWithCredentialAndPassword creates a new user account with email and password.
// If username support is enabled, a username can be provided in additionalFields.
// Returns an AuthenticationResult on success, or an error if signup fails.
func (p *credentialPasswordPlugin) SignUpWithCredentialAndPassword(ctx context.Context, user *limen.User, additionalFields map[string]any) (*limen.AuthenticationResult, error) {
	if err := p.validateUser(user, additionalFields); err != nil {
		return nil, err
	}

	username := strings.TrimSpace(limen.GetFromMap[string](additionalFields, "username"))

	if p.config.enableUsername && username != "" {
		usernameExists, err := p.checkUsernameExists(ctx, username)
		if err != nil {
			return nil, err
		}

		if usernameExists {
			return nil, ErrUsernameAlreadyExists
		}

		additionalFields[p.getUsernameField()] = username
	}

	userExists, err := p.checkEmailExists(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	if userExists {
		return nil, ErrEmailAlreadyExists
	}

	hashedPassword, err := p.HashPassword(*user.Password)
	if err != nil {
		return nil, err
	}

	var verification *limen.Verification

	err = p.core.WithTransaction(ctx, func(ctx context.Context) error {
		if err := p.dbAction.CreateUser(ctx, &limen.User{
			Email:    user.Email,
			Password: &hashedPassword,
		}, additionalFields); err != nil {
			return err
		}

		if p.config.requireEmailVerification {
			if verification, err = p.CreateEmailVerification(ctx, user); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	user, err = p.dbAction.FindUserByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	if verification != nil {
		p.SendVerificationEmail(ctx, user, verification)
	}

	return &limen.AuthenticationResult{User: user}, nil
}

func (p *credentialPasswordPlugin) checkUsernameExists(ctx context.Context, username string) (bool, error) {
	if !p.config.enableUsername {
		return false, nil
	}

	exists, err := p.core.Exists(ctx, p.userSchema, []limen.Where{
		limen.Eq(p.getUsernameField(), username),
	})
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (p *credentialPasswordPlugin) checkEmailExists(ctx context.Context, email string) (bool, error) {
	exists, err := p.core.Exists(ctx, p.userSchema, []limen.Where{
		limen.Eq(p.userSchema.GetEmailField(), email),
	})
	if err != nil {
		return false, err
	}
	return exists, nil
}

// CheckUsernameAvailability validates the username format and checks if it's available.
// Returns true if the username is available, false if it's already taken or invalid.
func (p *credentialPasswordPlugin) CheckUsernameAvailability(ctx context.Context, username string) (bool, error) {
	if !p.config.enableUsername {
		return false, ErrUsernameNotEnabled
	}

	trimmedUsername := strings.TrimSpace(username)
	if err := p.validateUsername(trimmedUsername); err != nil {
		return false, err
	}

	exists, err := p.checkUsernameExists(ctx, trimmedUsername)
	if err != nil {
		return false, err
	}

	return !exists, nil
}
