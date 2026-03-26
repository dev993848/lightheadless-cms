package middleware

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/lightcms/cms/internal/db"
	"github.com/lightcms/cms/internal/models"
)

// ctxKey is a private key type for context values to avoid collisions.
type ctxKey int

// SessionKey is the context key for storing the authenticated session.
const SessionKey ctxKey = 0

// Auth returns a middleware that validates the session cookie and stores the
// session in the request context. Redirects to /admin/login if invalid.
func Auth(database *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("cms_session")
			if err != nil || cookie.Value == "" {
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}

			session, err := db.GetSession(database, cookie.Value)
			if err != nil || session == nil {
				// Clear stale cookie.
				http.SetCookie(w, &http.Cookie{
					Name:   "cms_session",
					Value:  "",
					Path:   "/",
					MaxAge: -1,
				})
				http.Redirect(w, r, "/admin/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), SessionKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetSession retrieves the authenticated session from the context.
// Returns nil if no session is stored.
func GetSession(ctx context.Context) *models.Session {
	v := ctx.Value(SessionKey)
	if v == nil {
		return nil
	}
	s, _ := v.(*models.Session)
	return s
}
