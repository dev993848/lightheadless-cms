package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetFlash_SetsCookie(t *testing.T) {
	rr := httptest.NewRecorder()
	SetFlash(rr, Flash{Type: "success", Message: "Saved!"})

	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected flash cookie to be set")
	}
	found := false
	for _, c := range cookies {
		if c.Name == flashCookieName {
			found = true
			if c.Value == "" {
				t.Error("flash cookie value should not be empty")
			}
		}
	}
	if !found {
		t.Errorf("cookie %q not found", flashCookieName)
	}
}

func TestGetFlash_NoCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	flash := GetFlash(req, rr)
	if flash != nil {
		t.Error("expected nil flash when no cookie")
	}
}

func TestGetFlash_ReadsAndClears(t *testing.T) {
	// First set the flash.
	setRR := httptest.NewRecorder()
	SetFlash(setRR, Flash{Type: "error", Message: "Something went wrong"})

	// Extract the cookie value.
	var cookieHeader string
	for _, c := range setRR.Result().Cookies() {
		if c.Name == flashCookieName {
			cookieHeader = c.Value
		}
	}
	if cookieHeader == "" {
		t.Fatal("flash cookie not set")
	}

	// Now read it back.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: flashCookieName, Value: cookieHeader})
	getRR := httptest.NewRecorder()

	flash := GetFlash(req, getRR)
	if flash == nil {
		t.Fatal("expected flash message")
	}
	if flash.Type != "error" {
		t.Errorf("expected type=error, got %q", flash.Type)
	}
	if flash.Message != "Something went wrong" {
		t.Errorf("unexpected message: %q", flash.Message)
	}

	// The cookie should be cleared (MaxAge=-1).
	cleared := false
	for _, c := range getRR.Result().Cookies() {
		if c.Name == flashCookieName && c.MaxAge == -1 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("expected flash cookie to be cleared after reading")
	}
}

func TestGetFlash_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: flashCookieName, Value: "not-valid-json"})
	rr := httptest.NewRecorder()

	flash := GetFlash(req, rr)
	if flash != nil {
		t.Error("expected nil flash for invalid JSON")
	}
}
