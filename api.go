package aegis

import (
	"net/http"
)

type AegisSession struct {
	User    *User
	Session *Session
	Raw     map[string]any
}

func (a *Aegis) GetSession(req *http.Request) (*AegisSession, error) {
	sessionValidateResult, err := a.sessionManager.ValidateSession(req.Context(), req)
	if err != nil {
		return nil, err
	}
	return &AegisSession{
		User:    sessionValidateResult.User,
		Session: sessionValidateResult.Session,
	}, nil
}
