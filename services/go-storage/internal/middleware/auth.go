package middleware

import (
	"crypto/subtle"
	"net/http"
)

// ServiceToken returns middleware that validates the X-Service-Token header.
// If token is empty (dev mode), all requests are allowed through.
func ServiceToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			provided := r.Header.Get("X-Service-Token")
			// Constant-time compare to prevent timing attacks.
			if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
