package emailotp

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/thecodearcher/limen"
)

// SendOTP generates an OTP for the given email, persists it, and invokes the
// configured send callback. The plugin always returns success when the email
// is well-formed so callers cannot enumerate registered addresses; for the
// sign-in flow with sign-up disabled, no OTP is delivered when the user does
// not exist.
func (p *emailOTPPlugin) SendOTP(ctx context.Context, email string, opts ...*SendOTPOptions) (*EmailOTPMessage, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, ErrEmailRequired
	}

	otpType := TypeSignIn
	if len(opts) > 0 && opts[0] != nil && opts[0].Type != "" {
		otpType = opts[0].Type
	}
	if !otpType.valid() {
		return nil, ErrInvalidOTPType
	}

	// Sign-in with sign-up disabled: silently no-op for unknown emails so we
	// don't leak account existence.
	if otpType == TypeSignIn && p.config.disableSignUp {
		_, err := p.dbAction.FindUserByEmail(ctx, email)
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}

	// Email-verification requires an existing user; silently no-op otherwise.
	if otpType == TypeEmailVerification {
		_, err := p.dbAction.FindUserByEmail(ctx, email)
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}

	otp, err := p.generateOTP(email, otpType)
	if err != nil {
		return nil, err
	}

	if err := p.persistOTP(ctx, email, otpType, otp); err != nil {
		return nil, err
	}

	msg := EmailOTPMessage{Email: email, OTP: otp, Type: otpType}
	if p.config.sendOTP != nil {
		p.config.sendOTP(msg)
	}
	return &msg, nil
}

// VerifyOTP consumes an OTP for the email-verification flow and marks the
// matching user's email as verified. It returns the refreshed user so a
// caller can hand it to CreateSession if desired.
func (p *emailOTPPlugin) VerifyOTP(ctx context.Context, email, otp string) (*limen.AuthenticationResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, ErrEmailRequired
	}
	if strings.TrimSpace(otp) == "" {
		return nil, ErrOTPRequired
	}

	if _, err := p.consumeOTP(ctx, email, TypeEmailVerification, otp); err != nil {
		return nil, err
	}

	user, err := p.dbAction.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if user.EmailVerifiedAt == nil {
		now := time.Now()
		if err := p.dbAction.UpdateUser(ctx, &limen.User{EmailVerifiedAt: &now}, []limen.Where{
			limen.Eq(p.userSchema.GetIDField(), user.ID),
		}); err != nil {
			return nil, err
		}
		user.EmailVerifiedAt = &now
	}

	return &limen.AuthenticationResult{User: user}, nil
}

// SignInWithOTP consumes a sign-in OTP, returning the authenticated user.
// When the email has no corresponding user and sign-up is enabled, a new
// user is created using opts.AdditionalData.
func (p *emailOTPPlugin) SignInWithOTP(ctx context.Context, email, otp string, opts ...*SignInOptions) (*limen.AuthenticationResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, ErrEmailRequired
	}
	if strings.TrimSpace(otp) == "" {
		return nil, ErrOTPRequired
	}

	if _, err := p.consumeOTP(ctx, email, TypeSignIn, otp); err != nil {
		return nil, err
	}

	var signInOpts *SignInOptions
	if len(opts) > 0 {
		signInOpts = opts[0]
	}

	existing, err := p.dbAction.FindUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, limen.ErrRecordNotFound) {
		return nil, err
	}

	if existing == nil {
		if p.config.disableSignUp {
			return nil, ErrSignUpDisabled
		}
		additional := map[string]any{}
		if signInOpts != nil && signInOpts.AdditionalData != nil {
			additional = signInOpts.AdditionalData
		}
		now := time.Now()
		user := &limen.User{Email: email, EmailVerifiedAt: &now}
		if err := p.dbAction.CreateUser(ctx, user, additional); err != nil {
			return nil, err
		}
		// Re-read to obtain the persisted user including generated id and
		// any additional columns.
		refreshed, err := p.dbAction.FindUserByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		return &limen.AuthenticationResult{User: refreshed}, nil
	}

	if existing.EmailVerifiedAt == nil {
		now := time.Now()
		if err := p.dbAction.UpdateUser(ctx, &limen.User{EmailVerifiedAt: &now}, []limen.Where{
			limen.Eq(p.userSchema.GetIDField(), existing.ID),
		}); err != nil {
			return nil, err
		}
		existing.EmailVerifiedAt = &now
	}

	return &limen.AuthenticationResult{User: existing}, nil
}

// persistOTP writes a fresh verification row for (email, type), replacing any
// prior unconsumed verification for the same identifier so the table never
// accumulates duplicate live OTPs.
func (p *emailOTPPlugin) persistOTP(ctx context.Context, email string, otpType OTPType, otp string) error {
	identifier := toOTPIdentifier(otpType, email)

	if existing, err := p.dbAction.FindVerificationByAction(ctx, EmailOTPAction, identifier); err == nil {
		if err := p.dbAction.DeleteVerification(ctx, existing.ID); err != nil {
			return err
		}
	} else if !errors.Is(err, limen.ErrRecordNotFound) {
		return err
	}

	state := &otpState{
		Email:    email,
		Type:     otpType,
		OTP:      p.storeValue(otp),
		Attempts: 0,
	}
	encoded, err := encodeOTPState(state)
	if err != nil {
		return err
	}
	_, err = p.dbAction.CreateVerification(ctx, EmailOTPAction, identifier, encoded, p.config.otpExpiration)
	return err
}

// consumeOTP atomically validates a presented OTP. On success the
// verification row is deleted. On a wrong code the attempt counter is
// incremented; once it reaches allowedAttempts the row is removed and any
// further verification attempts fail with ErrTooManyAttempts.
func (p *emailOTPPlugin) consumeOTP(ctx context.Context, email string, otpType OTPType, presented string) (*otpState, error) {
	identifier := toOTPIdentifier(otpType, email)

	var resultState *otpState
	err := p.core.WithTransaction(ctx, func(ctx context.Context) error {
		verification, err := p.dbAction.FindVerificationByAction(ctx, EmailOTPAction, identifier)
		if err != nil {
			if errors.Is(err, limen.ErrRecordNotFound) {
				return ErrInvalidOTP
			}
			return err
		}

		if verification.ExpiresAt.Before(time.Now()) {
			if delErr := p.dbAction.DeleteVerification(ctx, verification.ID); delErr != nil {
				return delErr
			}
			return ErrOTPExpired
		}

		state, err := decodeOTPState(verification.Value)
		if err != nil {
			return err
		}

		if state.Attempts >= p.config.allowedAttempts {
			if delErr := p.dbAction.DeleteVerification(ctx, verification.ID); delErr != nil {
				return delErr
			}
			return ErrTooManyAttempts
		}

		if !p.matchOTP(state.OTP, presented) {
			state.Attempts++
			if state.Attempts >= p.config.allowedAttempts {
				if delErr := p.dbAction.DeleteVerification(ctx, verification.ID); delErr != nil {
					return delErr
				}
				return ErrTooManyAttempts
			}
			encoded, encodeErr := encodeOTPState(state)
			if encodeErr != nil {
				return encodeErr
			}
			verification.Value = encoded
			if updateErr := p.dbAction.UpdateVerification(ctx, verification, []limen.Where{
				limen.Eq(p.verificationSchema.GetIDField(), verification.ID),
			}); updateErr != nil {
				return updateErr
			}
			return ErrInvalidOTP
		}

		if delErr := p.dbAction.DeleteVerification(ctx, verification.ID); delErr != nil {
			return delErr
		}
		resultState = state
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resultState, nil
}
