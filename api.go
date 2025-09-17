package aegis

import "net/http"

type API interface {
	SignInWithEmailAndPassword(r *http.Request, w *http.ResponseWriter)
}
