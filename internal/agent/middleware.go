package agent

import (
	"net/http"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/auth"
)

func (a *Agent) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			writeError(w, http.StatusUnauthorized, errUnauthorized)
			return
		}
		token := strings.TrimPrefix(h, "Bearer ")
		if !auth.ValidateToken(token, a.Token) {
			writeError(w, http.StatusUnauthorized, errUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
