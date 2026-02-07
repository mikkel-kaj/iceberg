package agent

import (
	"net/http"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/auth"
)

func (a *Agent) isAuthorized(r *http.Request) bool {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(h, "Bearer ")
	return auth.ValidateToken(token, a.Token)
}

func (a *Agent) requireAuth(w http.ResponseWriter, r *http.Request) bool {
	if !a.isAuthorized(r) {
		writeError(w, http.StatusUnauthorized, errUnauthorized)
		return false
	}
	return true
}

func (a *Agent) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.requireAuth(w, r) {
			return
		}
		next.ServeHTTP(w, r)
	})
}
