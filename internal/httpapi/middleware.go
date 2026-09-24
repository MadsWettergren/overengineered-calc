package httpapi

import (
	"net/http"
	"strings"

	"github.com/MadsWettergren/over-engineered-calculator/internal/auth"
)

// requireAuth wraps a handler so it only runs when the request carries a
// valid bearer token.
//
// This is the one place in the package that knows about the Authorization
// header. Everything it wraps just runs normally once this returns — it
// doesn't need to know auth exists.
func requireAuth(authService *auth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		const prefix = "Bearer "

		header := request.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			writeError(
				responseWriter,
				http.StatusUnauthorized,
				"unauthorized",
				"missing or malformed Authorization header",
			)
			return
		}

		token := strings.TrimPrefix(header, prefix)

		if _, err := authService.Authenticate(token); err != nil {
			writeError(
				responseWriter,
				http.StatusUnauthorized,
				"unauthorized",
				"invalid or expired token",
			)
			return
		}

		next(responseWriter, request)
	}
}
