package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

const flashCookieName = "cms_flash"

// Flash represents a flash notification message.
type Flash struct {
	Type    string `json:"type"`    // success | error | warning
	Message string `json:"message"`
}

// SetFlash stores a flash message in a cookie (base64-encoded JSON, 1 hour TTL).
func SetFlash(w http.ResponseWriter, flash Flash) {
	data, err := json.Marshal(flash)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(data),
		Path:     "/",
		MaxAge:   3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(time.Hour),
	})
}

// GetFlash reads and clears the flash cookie. Returns nil if none is present.
func GetFlash(r *http.Request, w http.ResponseWriter) *Flash {
	cookie, err := r.Cookie(flashCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	// Clear the cookie immediately.
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil
	}
	var flash Flash
	if err := json.Unmarshal(decoded, &flash); err != nil {
		return nil
	}
	return &flash
}
