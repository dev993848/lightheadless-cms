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

func TestHandleSettings_Renders(t *testing.T) {
	s, database := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)

	db.Upsert(database, &models.Settings{SiteName: "Test CMS", AdminEmail: "admin@test.com"})

	req := httptest.NewRequest(http.MethodGet, "/admin/settings", nil)
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	s.HandleSettings(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String()[:min(200, rr.Body.Len())])
	}
	if !strings.Contains(rr.Body.String(), "<html") {
		t.Error("expected HTML response")
	}
}

func TestHandleSettingsSave_General(t *testing.T) {
	s, database := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)

	hash, _ := bcrypt.GenerateFromPassword([]byte("existing"), bcrypt.MinCost)
	db.Upsert(database, &models.Settings{
		SiteName:      "Old Name",
		AdminEmail:    "admin@test.com",
		AdminPassword: string(hash),
	})

	body := "_section=general&site_name=New+CMS&admin_email=newemail%40test.com"
	req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	s.HandleSettingsSave(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected redirect 302, got %d", rr.Code)
	}

	settings, _ := db.Get(database)
	if settings.SiteName != "New CMS" {
		t.Errorf("expected SiteName='New CMS', got %q", settings.SiteName)
	}
	if settings.AdminEmail != "newemail@test.com" {
		t.Errorf("expected email updated, got %q", settings.AdminEmail)
	}
}

func TestHandleSettingsSave_Bitrix24(t *testing.T) {
	s, database := setupTestServer(t)

	db.Upsert(database, &models.Settings{SiteName: "CMS"})

	body := "_section=bitrix24&bitrix24_webhook=https%3A%2F%2Fexample.com%2Fhook&bitrix24_enabled=1"
	req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	s.HandleSettingsSave(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected redirect 302, got %d: %s", rr.Code, rr.Body.String())
	}

	settings, _ := db.Get(database)
	if !settings.Bitrix24Enabled {
		t.Error("expected Bitrix24Enabled=true")
	}
	if settings.Bitrix24Webhook != "https://example.com/hook" {
		t.Errorf("expected webhook URL updated, got %q", settings.Bitrix24Webhook)
	}
}

func TestHandleSettingsSave_PasswordMismatch(t *testing.T) {
	s, database := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)

	db.Upsert(database, &models.Settings{SiteName: "CMS", AdminEmail: "a@b.com"})

	body := "_section=general&admin_password=newpass&admin_password_confirm=different"
	req := httptest.NewRequest(http.MethodPost, "/admin/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	s.HandleSettingsSave(rr, req)

	// Should re-render the form with an error (not redirect).
	if rr.Code == http.StatusFound {
		t.Error("should not redirect when passwords don't match")
	}
}

func TestHandleSettingsTestBitrix_NoWebhook(t *testing.T) {
	s, database := setupTestServer(t)

	// No webhook configured.
	db.Upsert(database, &models.Settings{SiteName: "CMS", Bitrix24Webhook: ""})

	req := httptest.NewRequest(http.MethodPost, "/admin/settings/test-bitrix", nil)
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	s.HandleSettingsTestBitrix(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "не задан") {
		t.Errorf("expected 'не задан' in response, got %q", rr.Body.String())
	}
}

func TestHtmlEscape(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"<script>", "&lt;script&gt;"},
		{"A & B", "A &amp; B"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"it's", "it&#39;s"},
		{"safe", "safe"},
	}
	for _, c := range cases {
		got := htmlEscape(c.input)
		if got != c.expected {
			t.Errorf("htmlEscape(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}
