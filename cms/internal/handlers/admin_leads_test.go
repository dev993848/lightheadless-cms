package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lightcms/cms/internal/db"
	"github.com/lightcms/cms/internal/middleware"
	"github.com/lightcms/cms/internal/models"
)

// injectSession injects a mock session into the request context.
func injectSession(r *http.Request, sessionID string) *http.Request {
	session := &models.Session{
		ID:        sessionID,
		AdminID:   1,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	ctx := context.WithValue(r.Context(), middleware.SessionKey, session)
	return r.WithContext(ctx)
}

func TestHandleDashboard(t *testing.T) {
	s, _ := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)

	req := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	s.HandleDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String()[:min(200, rr.Body.Len())])
	}
	if !strings.Contains(rr.Body.String(), "<html") {
		t.Error("expected HTML in dashboard")
	}
}

func TestHandleLeads_Full(t *testing.T) {
	s, database := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)

	_, _ = db.CreateLead(database, &models.Lead{Name: "Test", Phone: "123", Status: models.StatusNew})

	req := httptest.NewRequest(http.MethodGet, "/admin/leads", nil)
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	s.HandleLeads(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleLeads_HTMX(t *testing.T) {
	s, _ := setupTestServer(t)
	s.renderer = NewRenderer(templatesDir)

	req := httptest.NewRequest(http.MethodGet, "/admin/leads?status=new", nil)
	req.Header.Set("HX-Request", "true")
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	s.HandleLeads(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestHandleLeadDelete_Valid(t *testing.T) {
	s, database := setupTestServer(t)

	id, _ := db.CreateLead(database, &models.Lead{Name: "To Delete", Phone: "123", Status: models.StatusNew})

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /admin/leads/{id}", s.HandleLeadDelete)

	req := httptest.NewRequest(http.MethodDelete, "/admin/leads/"+itoa(int(id)), nil)
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// Lead should be deleted.
	lead, _ := db.GetLead(database, int(id))
	if lead != nil {
		t.Error("lead should be deleted")
	}
}

func TestHandleLeadDelete_InvalidID(t *testing.T) {
	s, _ := setupTestServer(t)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /admin/leads/{id}", s.HandleLeadDelete)

	req := httptest.NewRequest(http.MethodDelete, "/admin/leads/abc", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleLeadStatusUpdate_Valid(t *testing.T) {
	s, database := setupTestServer(t)

	id, _ := db.CreateLead(database, &models.Lead{Name: "Test", Phone: "123", Status: models.StatusNew})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/leads/{id}", s.HandleLeadStatusUpdate)

	body := "status=sent"
	req := httptest.NewRequest(http.MethodPost, "/admin/leads/"+itoa(int(id)), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected redirect 302, got %d", rr.Code)
	}

	lead, _ := db.GetLead(database, int(id))
	if lead.Status != models.StatusSent {
		t.Errorf("expected status 'sent', got %q", lead.Status)
	}
}

func TestHandleLeadStatusUpdate_InvalidStatus(t *testing.T) {
	s, database := setupTestServer(t)
	id, _ := db.CreateLead(database, &models.Lead{Name: "Test", Phone: "123", Status: models.StatusNew})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /admin/leads/{id}", s.HandleLeadStatusUpdate)

	body := "status=invalid"
	req := httptest.NewRequest(http.MethodPost, "/admin/leads/"+itoa(int(id)), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status, got %d", rr.Code)
	}
}

func TestHandleLeadsExport(t *testing.T) {
	s, database := setupTestServer(t)

	_, _ = db.CreateLead(database, &models.Lead{Name: "Export Test", Phone: "+7900", Email: "e@test.com", Status: models.StatusNew})

	req := httptest.NewRequest(http.MethodGet, "/admin/leads/export.csv", nil)
	req = injectSession(req, "test-session")
	rr := httptest.NewRecorder()
	s.HandleLeadsExport(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("expected CSV content-type, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "Export Test") {
		t.Errorf("expected lead name in CSV output")
	}
}

func TestParseLeadIDFromPath(t *testing.T) {
	id, suffix := parseLeadIDFromPath("/admin/leads/42")
	if id != 42 || suffix != "" {
		t.Errorf("expected (42, ''), got (%d, %q)", id, suffix)
	}

	id, suffix = parseLeadIDFromPath("/admin/leads/7/resend")
	if id != 7 || suffix != "resend" {
		t.Errorf("expected (7, 'resend'), got (%d, %q)", id, suffix)
	}

	id, _ = parseLeadIDFromPath("/admin/leads/abc")
	if id != 0 {
		t.Errorf("expected id=0 for invalid, got %d", id)
	}
}
