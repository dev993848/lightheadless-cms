package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

// csrfSecret is generated once at startup.
var csrfSecret []byte

func init() {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("csrf: failed to generate secret: " + err.Error())
	}
	csrfSecret = secret
}

// GenerateToken returns an HMAC-SHA256 of the sessionID using the CSRF secret.
func GenerateToken(sessionID string) string {
	mac := hmac.New(sha256.New, csrfSecret)
	mac.Write([]byte(sessionID))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks that the provided token matches the expected HMAC for sessionID.
// Uses constant-time comparison to prevent timing attacks.
func Verify(sessionID, token string) bool {
	expected := GenerateToken(sessionID)
	eBytes, err1 := hex.DecodeString(expected)
	tBytes, err2 := hex.DecodeString(token)
	if err1 != nil || err2 != nil {
		return false
	}
	return hmac.Equal(eBytes, tBytes)
}

// Middleware enforces CSRF token verification on mutating HTTP methods.
// Token is read from form field "_csrf" or header "X-CSRF-Token".
// Session ID is read from cookie "cms_session".
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			// Read session cookie.
			cookie, err := r.Cookie("cms_session")
			if err != nil || cookie.Value == "" {
				csrfForbidden(w)
				return
			}

			// Read token from form field or header.
			token := r.FormValue("_csrf")
			if token == "" {
				token = r.Header.Get("X-CSRF-Token")
			}

			if !Verify(cookie.Value, token) {
				csrfForbidden(w)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func csrfForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": "Invalid or missing CSRF token",
	})
}
