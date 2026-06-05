package emailotp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestSendOTP_RequiresEmail(t *testing.T) {
	t.Parallel()

	_, plugin := newTestLimenAndPlugin(t)
	_, err := plugin.SendOTP(context.Background(), "   ")
	assert.ErrorIs(t, err, ErrEmailRequired)
}

func TestSendOTP_RejectsUnknownType(t *testing.T) {
	t.Parallel()

	_, plugin := newTestLimenAndPlugin(t)
	_, err := plugin.SendOTP(context.Background(), "user@test.com", &SendOTPOptions{Type: OTPType("bogus")})
	assert.ErrorIs(t, err, ErrInvalidOTPType)
}

func TestSendOTP_GeneratesNumericCodeOfConfiguredLength(t *testing.T) {
	t.Parallel()

	var captured string
	_, plugin := newTestLimenAndPlugin(t, WithOTPLength(8), captureOTP(&captured))

	msg, err := plugin.SendOTP(context.Background(), "len@test.com")
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Len(t, captured, 8)
	for _, r := range captured {
		assert.True(t, r >= '0' && r <= '9', "OTP must contain only digits")
	}
}

func TestSendOTP_DisableSignUp_DoesNotSendWhenUserMissing(t *testing.T) {
	t.Parallel()

	var captured string
	_, plugin := newTestLimenAndPlugin(t, WithDisableSignUp(true), captureOTP(&captured))

	msg, err := plugin.SendOTP(context.Background(), "missing@test.com")
	require.NoError(t, err)
	assert.Nil(t, msg, "sign-in for unknown email should silently no-op")
	assert.Empty(t, captured, "no OTP should be delivered when sign-up is disabled and user is unknown")
}

func TestSendOTP_EmailVerification_DoesNotSendWhenUserMissing(t *testing.T) {
	t.Parallel()

	var captured string
	_, plugin := newTestLimenAndPlugin(t, captureOTP(&captured))

	msg, err := plugin.SendOTP(context.Background(), "no-user@test.com", &SendOTPOptions{Type: TypeEmailVerification})
	require.NoError(t, err)
	assert.Nil(t, msg)
	assert.Empty(t, captured)
}

func TestSendOTP_NormalizesEmailToLowerCase(t *testing.T) {
	t.Parallel()

	var captured EmailOTPMessage
	_, plugin := newTestLimenAndPlugin(t, WithSendOTP(func(m EmailOTPMessage) { captured = m }))

	_, err := plugin.SendOTP(context.Background(), "  Mixed@TEST.com  ")
	require.NoError(t, err)
	assert.Equal(t, "mixed@test.com", captured.Email)
}

func TestSendOTP_ReplacesPriorVerificationForSameIdentifier(t *testing.T) {
	t.Parallel()

	var otp string
	_, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	_, err := plugin.SendOTP(context.Background(), "rot@test.com")
	require.NoError(t, err)
	firstOTP := otp

	_, err = plugin.SendOTP(context.Background(), "rot@test.com")
	require.NoError(t, err)
	require.NotEqual(t, firstOTP, otp, "a fresh OTP should be generated on every send")

	// The prior OTP must no longer be valid.
	_, err = plugin.SignInWithOTP(context.Background(), "rot@test.com", firstOTP)
	assert.ErrorIs(t, err, ErrInvalidOTP)

	// The latest OTP works.
	_, err = plugin.SignInWithOTP(context.Background(), "rot@test.com", otp)
	require.NoError(t, err)
}

func TestSignInWithOTP_AutoCreatesUser(t *testing.T) {
	t.Parallel()

	var otp string
	_, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	_, err := plugin.SendOTP(context.Background(), "fresh@test.com")
	require.NoError(t, err)

	_, err = plugin.dbAction.FindUserByEmail(context.Background(), "fresh@test.com")
	require.ErrorIs(t, err, limen.ErrRecordNotFound)

	result, err := plugin.SignInWithOTP(context.Background(), "fresh@test.com", otp)
	require.NoError(t, err)
	require.NotNil(t, result.User)
	assert.Equal(t, "fresh@test.com", result.User.Email)
	require.NotNil(t, result.User.EmailVerifiedAt)

	user, err := plugin.dbAction.FindUserByEmail(context.Background(), "fresh@test.com")
	require.NoError(t, err)
	assert.Equal(t, result.User.ID, user.ID)
}

func TestSignInWithOTP_PropagatesAdditionalDataOnSignUp(t *testing.T) {
	t.Parallel()

	var otp string
	_, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	_, err := plugin.SendOTP(context.Background(), "meta@test.com")
	require.NoError(t, err)

	_, err = plugin.SignInWithOTP(context.Background(), "meta@test.com", otp, &SignInOptions{
		AdditionalData: map[string]any{
			"first_name": "Ada",
			"role":       "founder",
		},
	})
	require.NoError(t, err)

	user, err := plugin.dbAction.FindUserByEmail(context.Background(), "meta@test.com")
	require.NoError(t, err)
	assert.Equal(t, "Ada", user.Raw()["first_name"])
	assert.Equal(t, "founder", user.Raw()["role"])
}

func TestSignInWithOTP_DisableSignUp_ReturnsError(t *testing.T) {
	t.Parallel()

	// Generate an OTP for a user that doesn't exist, then re-init with
	// disableSignUp to confirm verification still rejects the sign-up.
	// We do this by generating then directly inserting the verification.
	var otp string
	_, plugin := newTestLimenAndPlugin(t, WithDisableSignUp(false), captureOTP(&otp))

	// Pre-create the verification by going through send (sign-up enabled).
	_, err := plugin.SendOTP(context.Background(), "blocked@test.com")
	require.NoError(t, err)
	require.NotEmpty(t, otp)

	// Flip the flag at the plugin level so verify-time policy applies.
	plugin.config.disableSignUp = true

	_, err = plugin.SignInWithOTP(context.Background(), "blocked@test.com", otp)
	assert.ErrorIs(t, err, ErrSignUpDisabled)
}

func TestSignInWithOTP_MarksExistingUnverifiedUserAsVerified(t *testing.T) {
	t.Parallel()

	var otp string
	l, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	seeded := limen.SeedTestUser(t, l, "existing@test.com")
	require.Nil(t, seeded.EmailVerifiedAt)

	_, err := plugin.SendOTP(context.Background(), "existing@test.com")
	require.NoError(t, err)

	result, err := plugin.SignInWithOTP(context.Background(), "existing@test.com", otp)
	require.NoError(t, err)
	require.NotNil(t, result.User.EmailVerifiedAt)
	assert.Equal(t, seeded.ID, result.User.ID)
}

func TestSignInWithOTP_WrongCodeIncrementsAttemptsThenLocksOut(t *testing.T) {
	t.Parallel()

	var otp string
	_, plugin := newTestLimenAndPlugin(t, WithAllowedAttempts(3), captureOTP(&otp))

	_, err := plugin.SendOTP(context.Background(), "attempts@test.com")
	require.NoError(t, err)
	require.NotEmpty(t, otp)

	wrong := wrongOTP(otp)

	_, err = plugin.SignInWithOTP(context.Background(), "attempts@test.com", wrong)
	assert.ErrorIs(t, err, ErrInvalidOTP)

	_, err = plugin.SignInWithOTP(context.Background(), "attempts@test.com", wrong)
	assert.ErrorIs(t, err, ErrInvalidOTP)

	// Third failed attempt should both fail *and* invalidate the OTP.
	_, err = plugin.SignInWithOTP(context.Background(), "attempts@test.com", wrong)
	assert.ErrorIs(t, err, ErrTooManyAttempts)

	// Now even the correct OTP should fail because the row is gone.
	_, err = plugin.SignInWithOTP(context.Background(), "attempts@test.com", otp)
	assert.ErrorIs(t, err, ErrInvalidOTP)
}

func TestSignInWithOTP_ConsumesOTPOnSuccess(t *testing.T) {
	t.Parallel()

	var otp string
	_, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	_, err := plugin.SendOTP(context.Background(), "once@test.com")
	require.NoError(t, err)

	_, err = plugin.SignInWithOTP(context.Background(), "once@test.com", otp)
	require.NoError(t, err)

	_, err = plugin.SignInWithOTP(context.Background(), "once@test.com", otp)
	assert.ErrorIs(t, err, ErrInvalidOTP, "OTP must be single-use")
}

func TestSignInWithOTP_ExpiredOTP(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var otp string
		_, plugin := newTestLimenAndPlugin(t,
			WithOTPExpiration(5*time.Minute),
			captureOTP(&otp),
		)

		_, err := plugin.SendOTP(context.Background(), "expired@test.com")
		require.NoError(t, err)
		require.NotEmpty(t, otp)

		time.Sleep(6 * time.Minute)

		_, err = plugin.SignInWithOTP(context.Background(), "expired@test.com", otp)
		assert.ErrorIs(t, err, ErrOTPExpired)
	})
}

func TestVerifyOTP_MarksEmailAsVerified(t *testing.T) {
	t.Parallel()

	var otp string
	l, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	seeded := limen.SeedTestUser(t, l, "verify-flow@test.com")
	require.Nil(t, seeded.EmailVerifiedAt)

	_, err := plugin.SendOTP(context.Background(), "verify-flow@test.com", &SendOTPOptions{Type: TypeEmailVerification})
	require.NoError(t, err)

	result, err := plugin.VerifyOTP(context.Background(), "verify-flow@test.com", otp)
	require.NoError(t, err)
	require.NotNil(t, result.User.EmailVerifiedAt)
	assert.Equal(t, seeded.ID, result.User.ID)
}

func TestVerifyOTP_UserNotFound(t *testing.T) {
	t.Parallel()

	// Generate a verification row by sending under email-verification flow
	// then deleting the user — exercise the "valid OTP, missing user" branch.
	var otp string
	l, plugin := newTestLimenAndPlugin(t, captureOTP(&otp))

	limen.SeedTestUser(t, l, "deleteme@test.com")

	_, err := plugin.SendOTP(context.Background(), "deleteme@test.com", &SendOTPOptions{Type: TypeEmailVerification})
	require.NoError(t, err)

	// Remove the user out from under the verification.
	user, err := plugin.dbAction.FindUserByEmail(context.Background(), "deleteme@test.com")
	require.NoError(t, err)
	require.NoError(t, plugin.core.Delete(context.Background(), plugin.userSchema, []limen.Where{
		limen.Eq(plugin.userSchema.GetIDField(), user.ID),
	}))

	_, err = plugin.VerifyOTP(context.Background(), "deleteme@test.com", otp)
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

func TestVerifyOTP_RequiresEmailAndOTP(t *testing.T) {
	t.Parallel()

	_, plugin := newTestLimenAndPlugin(t)
	_, err := plugin.VerifyOTP(context.Background(), "", "123456")
	assert.ErrorIs(t, err, ErrEmailRequired)
	_, err = plugin.VerifyOTP(context.Background(), "x@test.com", "  ")
	assert.ErrorIs(t, err, ErrOTPRequired)
}

func TestSendOTP_SignInAndEmailVerificationDoNotShareVerification(t *testing.T) {
	t.Parallel()

	var captured []EmailOTPMessage
	l, plugin := newTestLimenAndPlugin(t, WithSendOTP(func(m EmailOTPMessage) {
		captured = append(captured, m)
	}))

	limen.SeedTestUser(t, l, "two-types@test.com")

	_, err := plugin.SendOTP(context.Background(), "two-types@test.com", &SendOTPOptions{Type: TypeSignIn})
	require.NoError(t, err)
	_, err = plugin.SendOTP(context.Background(), "two-types@test.com", &SendOTPOptions{Type: TypeEmailVerification})
	require.NoError(t, err)

	require.Len(t, captured, 2)
	signInOTP := captured[0].OTP
	verifyOTP := captured[1].OTP

	// Sign-in OTP must NOT work for email-verification flow, and vice versa.
	_, err = plugin.VerifyOTP(context.Background(), "two-types@test.com", signInOTP)
	assert.ErrorIs(t, err, ErrInvalidOTP)

	_, err = plugin.SignInWithOTP(context.Background(), "two-types@test.com", verifyOTP)
	assert.ErrorIs(t, err, ErrInvalidOTP)

	// Each type's OTP works for its own flow.
	_, err = plugin.VerifyOTP(context.Background(), "two-types@test.com", verifyOTP)
	require.NoError(t, err)
	_, err = plugin.SignInWithOTP(context.Background(), "two-types@test.com", signInOTP)
	require.NoError(t, err)
}

func TestHashStoredOTP_StoresHashAndStillVerifies(t *testing.T) {
	t.Parallel()

	var otp string
	_, plugin := newTestLimenAndPlugin(t, WithHashStoredOTP(true), captureOTP(&otp))

	_, err := plugin.SendOTP(context.Background(), "hashed@test.com")
	require.NoError(t, err)
	require.NotEmpty(t, otp)

	// Verify that the stored value is NOT the plain OTP.
	verification, err := plugin.dbAction.FindVerificationByAction(context.Background(), EmailOTPAction, toOTPIdentifier(TypeSignIn, "hashed@test.com"))
	require.NoError(t, err)
	assert.False(t, strings.Contains(verification.Value, otp), "stored verification must not contain the plain OTP when hashing is enabled")

	_, err = plugin.SignInWithOTP(context.Background(), "hashed@test.com", otp)
	require.NoError(t, err, "hashed OTP must still verify against presented plaintext")
}

// wrongOTP returns an OTP of the same length and digit set as otp but
// guaranteed to differ from it.
func wrongOTP(otp string) string {
	if otp == "" {
		return "111111"
	}
	b := []byte(otp)
	for i := range b {
		if b[i] == '0' {
			b[i] = '1'
		} else {
			b[i] = '0'
		}
	}
	return string(b)
}
