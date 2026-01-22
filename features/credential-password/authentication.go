package credentialpassword

import (
	"context"
	"fmt"
	"strings"

	"github.com/thecodearcher/aegis"
)

// SignInWithCredentialAndPassword authenticates a user with either email or username (if enabled) and password.
// The credential parameter can be either an email address or a username.
// Returns an AuthenticationResult on success, or an error if authentication fails.
func (p *credentialPasswordFeature) SignInWithCredentialAndPassword(ctx context.Context, credential string, password string) (*aegis.AuthenticationResult, error) {
	isUsername := !strings.Contains(credential, "@") && p.config.enableUsername

	var user *aegis.User
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

// authenticateUser validates the password and returns the authentication result.
func (p *credentialPasswordFeature) authenticateUser(user *aegis.User, password string) (*aegis.AuthenticationResult, error) {
	isValid, err := p.ComparePassword(password, user.Password)
	if err != nil {
		return nil, err
	}

	if !isValid {
		return nil, ErrInvalidPassword
	}

	pendingActions := []aegis.PendingAction{}
	if p.config.requireEmailVerification && user.EmailVerifiedAt == nil {
		pendingActions = append(pendingActions, aegis.PendingActionEmailVerification)
	}

	return &aegis.AuthenticationResult{
		User:           user,
		PendingActions: pendingActions,
	}, nil
}

// FindUserByUsername finds a user by their username.
// Returns an error if username support is not enabled or if the user is not found.
func (p *credentialPasswordFeature) FindUserByUsername(ctx context.Context, username string) (*aegis.User, error) {
	if !p.config.enableUsername {
		return nil, fmt.Errorf("username support is not enabled")
	}

	user, err := p.core.FindOne(ctx, p.userSchema, []aegis.Where{
		aegis.Eq(p.getUsernameField(), username),
	}, nil)

	if err != nil {
		return nil, err
	}

	return user.(*aegis.User), nil
}

// SignUpWithCredentialAndPassword creates a new user account with email and password.
// If username support is enabled, a username can be provided in additionalFields.
// Returns an AuthenticationResult on success, or an error if signup fails.
func (p *credentialPasswordFeature) SignUpWithCredentialAndPassword(ctx context.Context, user *aegis.User, additionalFields map[string]any) (*aegis.AuthenticationResult, error) {
	if err := p.validateUser(user, additionalFields); err != nil {
		return nil, err
	}

	username := strings.TrimSpace(aegis.GetFromMap[string](additionalFields, "username"))

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

	hashedPassword, err := p.HashPassword(user.Password)
	if err != nil {
		return nil, err
	}

	var pendingActions []aegis.PendingAction
	var verification *aegis.Verification

	err = p.core.WithTransaction(ctx, func(ctx context.Context) error {
		if err := p.dbAction.CreateUser(ctx, &aegis.User{
			Email:    user.Email,
			Password: hashedPassword,
		}, additionalFields); err != nil {
			return err
		}

		if p.config.requireEmailVerification {
			if verification, err = p.CreateEmailVerification(ctx, user); err != nil {
				return err
			}
			pendingActions = append(pendingActions, aegis.PendingActionEmailVerification)
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

	return &aegis.AuthenticationResult{
		User:           user,
		PendingActions: pendingActions,
	}, nil
}

func (p *credentialPasswordFeature) checkUsernameExists(ctx context.Context, username string) (bool, error) {
	if !p.config.enableUsername {
		return false, nil
	}

	exists, err := p.core.Exists(ctx, p.userSchema, []aegis.Where{
		aegis.Eq(p.getUsernameField(), username),
	})
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (p *credentialPasswordFeature) checkEmailExists(ctx context.Context, email string) (bool, error) {
	exists, err := p.core.Exists(ctx, p.userSchema, []aegis.Where{
		aegis.Eq(p.userSchema.GetEmailField(), email),
	})
	if err != nil {
		return false, err
	}
	return exists, nil
}
