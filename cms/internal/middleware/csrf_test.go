package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateToken_Deterministic(t *testing.T) {
	t1 := GenerateToken("session-abc")
	t2 := GenerateToken("session-abc")
	if t1 != t2 {
		t.Error("GenerateToken should be deterministic for the same session ID")
	}
}

func TestGenerateToken_DifferentInputs(t *testing.T) {
	t1 := GenerateToken("session-aaa")
	t2 := GenerateToken("session-bbb")
	if t1 == t2 {
		t.Error("different session IDs should produce different tokens")
	}
}

func TestVerify_Valid(t *testing.T) {
	sessionID := "test-session-id"
	token := GenerateToken(sessionID)
	if !Verify(sessionID, token) {
		t.Error("Verify should return true for valid token")
	}
}

func TestVerify_Invalid(t *testing.T) {
	if Verify("session-id", "wrongtoken") {
		t.Error("Verify should return false for wrong token")
	}
}

func TestVerify_EmptyToken(t *testing.T) {
	if Verify("session-id", "") {
		t.Error("Verify should return false for empty token")
	}
}

func TestVerify_WrongSession(t *testing.T) {
	token := GenerateToken("correct-session")
	if Verify("wrong-session", token) {
		t.Error("Verify should return false for wrong session")
	}
}

func TestCSRFMiddleware_GET_Passes(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("GET request should pass, got %d", rr.Code)
	}
}

func TestCSRFMiddleware_POST_NoSession(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/admin/settings", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("POST without session should get 403, got %d", rr.Code)
	}
}

func TestCSRFMiddleware_POST_ValidToken(t *testing.T) {
	sessionID := "valid-session-123"
	token := GenerateToken(sessionID)

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := "_csrf=" + token
	req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "cms_session", Value: sessionID})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("POST with valid CSRF should pass, got %d", rr.Code)
	}
}

func TestCSRFMiddleware_POST_InvalidToken(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := "_csrf=invalid-token"
	req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "cms_session", Value: "some-session"})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("POST with invalid CSRF should get 403, got %d", rr.Code)
	}
}

func TestCSRFMiddleware_DELETE_Header(t *testing.T) {
	sessionID := "del-session-456"
	token := GenerateToken(sessionID)

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/admin/leads/1", nil)
	req.Header.Set("X-CSRF-Token", token)
	req.AddCookie(&http.Cookie{Name: "cms_session", Value: sessionID})

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("DELETE with valid CSRF header should pass, got %d", rr.Code)
	}
}
