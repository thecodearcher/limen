package passkey

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"

	"github.com/go-webauthn/webauthn/webauthn"
)

// challengeState is what we persist in the verifications.value column
// for each in-flight ceremony. It carries the WebAuthn library's
// SessionData (challenge, allowed credentials, user verification
// requirement, etc.) plus identifying information that lets the
// "finish" endpoint look up the user without trusting the client.
type challengeState struct {
	Ceremony  CeremonyKind         `json:"k"`
	UserID    any                  `json:"u,omitempty"`
	UserName  string               `json:"un,omitempty"`
	Session   webauthn.SessionData `json:"s"`
}

func encodeChallengeState(s *challengeState) (string, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeChallengeState(value string) (*challengeState, error) {
	var s challengeState
	if err := json.Unmarshal([]byte(value), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// newVerificationToken returns a random URL-safe identifier used both
// as the signed cookie value handed to the client and as the lookup
// key for the persisted challenge row.
func newVerificationToken() (string, error) {
	b := make([]byte, verificationTokenByteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
