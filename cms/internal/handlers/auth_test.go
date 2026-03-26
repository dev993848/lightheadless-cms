package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lightcms/cms/internal/db"
	"github.com/lightcms/cms/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func setupAuthServer(t *testing.T) *Server {
	t.Helper()
	s, database := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)
	_ = database
	return s
}

func TestHandleLoginPage_Renders(t *testing.T) {
	s := setupAuthServer(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rr := httptest.NewRecorder()
	s.HandleLoginPage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String()[:min(200, rr.Body.Len())])
	}
	if !strings.Contains(rr.Body.String(), "<html") {
		t.Error("expected HTML in login page")
	}
}

func TestHandleLogin_WrongEmail(t *testing.T) {
	s, database := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)

	// Seed with known settings.
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	db.Upsert(database, &models.Settings{
		AdminEmail:    "admin@test.com",
		AdminPassword: string(hash),
	})

	body := "email=wrong%40test.com&password=password123"
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.HandleLogin(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong email, got %d", rr.Code)
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	s, database := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.MinCost)
	db.Upsert(database, &models.Settings{
		AdminEmail:    "admin@test.com",
		AdminPassword: string(hash),
	})

	body := "email=admin%40test.com&password=wrongpassword"
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.HandleLogin(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", rr.Code)
	}
}

func TestHandleLogin_Success(t *testing.T) {
	s, database := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	db.Upsert(database, &models.Settings{
		AdminEmail:    "admin@test.com",
		AdminPassword: string(hash),
	})

	body := "email=admin%40test.com&password=secret"
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.HandleLogin(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected redirect 302 on success, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/admin/" {
		t.Errorf("expected redirect to /admin/, got %q", rr.Header().Get("Location"))
	}

	// Check session cookie was set.
	var sessionCookie string
	for _, c := range rr.Result().Cookies() {
		if c.Name == "cms_session" {
			sessionCookie = c.Value
		}
	}
	if sessionCookie == "" {
		t.Error("expected session cookie to be set on login")
	}
}

func TestHandleLogout_ClearsSession(t *testing.T) {
	s, database := setupTestServer(t)

	sessionID, _ := db.CreateSession(database, 1)

	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	req.AddCookie(&http.Cookie{Name: "cms_session", Value: sessionID})
	rr := httptest.NewRecorder()
	s.HandleLogout(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/admin/login" {
		t.Errorf("expected redirect to /admin/login")
	}

	// Session should be deleted from DB.
	session, err := db.GetSession(database, sessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if session != nil {
		t.Error("session should be deleted after logout")
	}
}

func TestHandleLogout_NoCookie(t *testing.T) {
	s, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/admin/logout", nil)
	rr := httptest.NewRecorder()
	s.HandleLogout(rr, req)

	// Should redirect to login even without a session cookie.
	if rr.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rr.Code)
	}
}
