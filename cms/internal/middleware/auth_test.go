package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lightcms/cms/internal/db"
	_ "modernc.org/sqlite"
)

func openAuthDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func TestAuth_NoCookie_Redirects(t *testing.T) {
	database := openAuthDB(t)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		Auth(database),
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected redirect 302, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/admin/login" {
		t.Errorf("expected redirect to /admin/login, got %q", rr.Header().Get("Location"))
	}
}

func TestAuth_InvalidSession_Redirects(t *testing.T) {
	database := openAuthDB(t)
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
		Auth(database),
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(&http.Cookie{Name: "cms_session", Value: "non-existent-session"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected redirect 302, got %d", rr.Code)
	}
}

func TestAuth_ValidSession_Passes(t *testing.T) {
	database := openAuthDB(t)

	sessionID, err := db.CreateSession(database, 1)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	handlerCalled := false
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}),
		Auth(database),
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(&http.Cookie{Name: "cms_session", Value: sessionID})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for valid session, got %d", rr.Code)
	}
	if !handlerCalled {
		t.Error("handler should have been called")
	}
}

func TestAuth_ValidSession_StoresInContext(t *testing.T) {
	database := openAuthDB(t)
	sessionID, _ := db.CreateSession(database, 1)

	var ctxSessionID string
	handler := Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := GetSession(r.Context())
			if s != nil {
				ctxSessionID = s.ID
			}
			w.WriteHeader(http.StatusOK)
		}),
		Auth(database),
	)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req.AddCookie(&http.Cookie{Name: "cms_session", Value: sessionID})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if ctxSessionID != sessionID {
		t.Errorf("expected session in context, got %q", ctxSessionID)
	}
}

func TestGetSession_NoValue(t *testing.T) {
	s := GetSession(context.Background())
	if s != nil {
		t.Error("expected nil session from empty context")
	}
}
