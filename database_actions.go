package aegis

import (
	"context"
	"errors"
	"time"
)

// DatabaseActionHelper provides common database operations for plugins.
type DatabaseActionHelper struct {
	core *AegisCore
}

func newCommonDatabaseActionsHelper(core *AegisCore) *DatabaseActionHelper {
	return &DatabaseActionHelper{core: core}
}

func (h *DatabaseActionHelper) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	user, err := h.core.FindOne(ctx, h.core.Schema.User, []Where{
		Eq(h.core.Schema.User.GetEmailField(), email),
	}, nil)
	if err != nil {
		return nil, err
	}
	return user.(*User), nil
}

func (h *DatabaseActionHelper) FindUser(ctx context.Context, conditions []Where) (*User, error) {
	user, err := h.core.FindOne(ctx, h.core.Schema.User, conditions, nil)
	if err != nil {
		return nil, err
	}
	return user.(*User), nil
}

func (h *DatabaseActionHelper) FindUserByID(ctx context.Context, id any) (*User, error) {
	user, err := h.core.FindOne(ctx, h.core.Schema.User, []Where{
		Eq(h.core.Schema.User.GetIDField(), id),
	}, nil)

	if err != nil {
		return nil, err
	}
	return user.(*User), nil
}

func (h *DatabaseActionHelper) CreateUser(ctx context.Context, data *User, additionalFields map[string]any) error {
	if err := h.core.Create(ctx, h.core.Schema.User, data, additionalFields); err != nil {
		return err
	}

	return nil
}

func (h *DatabaseActionHelper) UpdateUser(ctx context.Context, updatedUser *User, conditions []Where) error {
	if err := h.core.Update(ctx, h.core.Schema.User, updatedUser, conditions); err != nil {
		return err
	}
	return nil
}

func (h *DatabaseActionHelper) CreateVerification(ctx context.Context, action string, identifier string, token string, expiresAt time.Duration) (*Verification, error) {
	if identifier == "" {
		return nil, errors.New("identifier is required")
	}
	verificationSchema := h.core.Schema.Verification
	actionValue := GenerateVerificationAction(action, identifier)

	if err := h.core.Create(ctx, verificationSchema, &Verification{
		Subject:   actionValue,
		Value:     token,
		ExpiresAt: time.Now().Add(expiresAt),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil); err != nil {
		return nil, err
	}

	return h.FindVerificationByAction(ctx, action, identifier)
}

func (h *DatabaseActionHelper) FindVerificationByAction(ctx context.Context, action string, identifier string) (*Verification, error) {
	verificationSchema := h.core.Schema.Verification
	actionValue := GenerateVerificationAction(action, identifier)
	verification, err := h.core.FindOne(ctx, verificationSchema,
		[]Where{
			Eq(verificationSchema.GetSubjectField(), actionValue),
		},
		[]OrderBy{
			{
				Column:    verificationSchema.GetCreatedAtField(),
				Direction: OrderByDesc,
			},
		})
	if err != nil {
		return nil, err
	}
	return verification.(*Verification), nil
}

func (h *DatabaseActionHelper) FindValidVerificationByToken(ctx context.Context, token string) (*Verification, error) {
	verificationSchema := h.core.Schema.Verification
	verification, err := h.core.FindOne(ctx, verificationSchema,
		[]Where{
			Eq(verificationSchema.GetValueField(), token),
			Gt(verificationSchema.GetExpiresAtField(), time.Now()),
		},
		[]OrderBy{
			{
				Column:    verificationSchema.GetCreatedAtField(),
				Direction: OrderByDesc,
			},
		})
	if err != nil {
		return nil, err
	}
	return verification.(*Verification), nil
}

// VerifyVerificationToken verifies a verification token for a given action and identifier.
// Returns an error if the token is invalid or the action and identifier do not match.
//
// Note: This function will delete the verification token after it is verified.
func (h *DatabaseActionHelper) VerifyVerificationToken(ctx context.Context, token string, action string, identifier string) error {
	verification, err := h.FindValidVerificationByToken(ctx, token)
	if err != nil {
		return ErrVerificationTokenInvalid
	}

	verificationAction, verificationIdentifier := ParseVerificationAction(verification.Subject)
	if action != verificationAction || identifier != verificationIdentifier {
		return ErrVerificationTokenInvalid
	}

	return h.DeleteVerificationToken(ctx, token)
}

func (h *DatabaseActionHelper) DeleteVerificationToken(ctx context.Context, token string) error {
	verificationSchema := h.core.Schema.Verification
	return h.core.Delete(ctx, verificationSchema, []Where{
		Eq(verificationSchema.GetValueField(), token),
	})
}

func (h *DatabaseActionHelper) CreateSession(ctx context.Context, data *Session, additionalFields map[string]any) error {
	if err := h.core.Create(ctx, h.core.Schema.Session, data, additionalFields); err != nil {
		return err
	}
	return nil
}

func (h *DatabaseActionHelper) UpdateSession(ctx context.Context, data *Session, conditions []Where) error {
	if err := h.core.Update(ctx, h.core.Schema.Session, data, conditions); err != nil {
		return err
	}
	return nil
}

func (h *DatabaseActionHelper) FindSessionByToken(ctx context.Context, sessionToken string) (*Session, error) {
	sessionSchema := h.core.Schema.Session
	session, err := h.core.FindOne(ctx, sessionSchema, []Where{
		Eq(sessionSchema.GetTokenField(), sessionToken),
	}, nil)
	if err != nil {
		return nil, err
	}
	return session.(*Session), nil
}

func (h *DatabaseActionHelper) DeleteSessionByToken(ctx context.Context, sessionToken string) error {
	sessionSchema := h.core.Schema.Session
	return h.core.Delete(ctx, sessionSchema, []Where{
		Eq(sessionSchema.GetTokenField(), sessionToken),
	})
}

func (h *DatabaseActionHelper) DeleteSessionByUserID(ctx context.Context, userID any) error {
	sessionSchema := h.core.Schema.Session
	return h.core.Delete(ctx, sessionSchema, []Where{
		Eq(sessionSchema.GetUserIDField(), userID),
	})
}
