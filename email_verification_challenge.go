package limen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const emailVerificationChallengeType = "email_verification"

type emailVerificationChallenge struct {
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
	Type  string `json:"type"`
}

// EmailVerificationChallengeEnabled reports whether waiting-room cookies may be issued.
// Callers should only issue one when a session is withheld until the email is verified.
func (c *LimenCore) EmailVerificationChallengeEnabled() bool {
	return c.EmailVerificationEnabled() && c.config.Email.verification.challengeEnabled
}

// IssueEmailVerificationChallenge sets an encrypted cookie that can resend verification
// without a full session.
func (c *LimenCore) IssueEmailVerificationChallenge(w http.ResponseWriter, email string) error {
	cfg := c.config.Email.verification
	if !cfg.enabled {
		return fmt.Errorf("email verification is not enabled")
	}

	payload, err := json.Marshal(emailVerificationChallenge{
		Email: email,
		Exp:   time.Now().Add(cfg.challengeTTL).Unix(),
		Type:  emailVerificationChallengeType,
	})
	if err != nil {
		return err
	}

	return c.cookies.SetSignedCookie(w, cfg.challengeCookieName, string(payload), int(cfg.challengeTTL.Seconds()))
}

// ClearEmailVerificationChallenge removes the waiting-room cookie.
func (c *LimenCore) ClearEmailVerificationChallenge(w http.ResponseWriter) {
	c.cookies.Delete(w, c.config.Email.verification.challengeCookieName)
}

// ResolveEmailVerificationChallenge loads the user bound to a valid waiting-room cookie.
func (c *LimenCore) ResolveEmailVerificationChallenge(r *http.Request) (*User, error) {
	raw, err := c.cookies.GetSignedCookie(r, c.config.Email.verification.challengeCookieName)
	if err != nil {
		return nil, ErrUnauthorized
	}

	var payload emailVerificationChallenge
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, ErrUnauthorized
	}
	if payload.Type != emailVerificationChallengeType || time.Now().Unix() >= payload.Exp {
		return nil, ErrUnauthorized
	}

	return c.DBAction.FindUserByEmail(r.Context(), payload.Email)
}
