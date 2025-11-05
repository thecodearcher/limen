package emailpassword

import (
	"net/http"

	"github.com/thecodearcher/aegis"
)

var (
	ErrInvalidCredentials = aegis.NewAegisError("invalid credentials", http.StatusUnauthorized, nil)
)
