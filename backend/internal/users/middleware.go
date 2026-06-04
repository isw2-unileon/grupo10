package users

import (
	"context"
	"net/http"
	"strings"
)

// TokenParser validates a token string and returns its claims. JWTIssuer
// implements it; abstracting it keeps the middleware independent from the
// signing strategy and easy to test.
type TokenParser interface {
	Parse(token string) (*Claims, error)
}

// contextKey is unexported so other packages cannot collide with our keys.
type contextKey string

const userIDKey contextKey = "userID"

// RequireAuth returns middleware that rejects requests lacking a valid Bearer
// token and, on success, stores the authenticated user ID in the request
// context for downstream handlers to read via UserIDFromContext.
func RequireAuth(parser TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or malformed Authorization header"})
				return
			}
			claims, err := parser.Parse(token)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(h[len(prefix):]), true
}

// UserIDFromContext returns the authenticated user ID stored by RequireAuth.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok && id != ""
}
