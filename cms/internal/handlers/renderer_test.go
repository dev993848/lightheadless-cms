package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// templatesDir is the relative path to templates from the handlers package.
const templatesDir = "../../internal/templates"

func TestJSON_WritesJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	JSON(rr, 200, map[string]string{"key": "value"})

	if rr.Code != 200 {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON content-type, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), `"key"`) {
		t.Errorf("expected key in response, got %q", rr.Body.String())
	}
}

func TestNewRenderer_ReturnsRenderer(t *testing.T) {
	r := NewRenderer(templatesDir)
	if r == nil {
		t.Error("expected non-nil renderer")
	}
	if r.dir != templatesDir {
		t.Errorf("expected dir %q, got %q", templatesDir, r.dir)
	}
}

func TestRenderer_Standalone_LoginPage(t *testing.T) {
	r := NewRenderer(templatesDir)
	rr := httptest.NewRecorder()

	r.Standalone(rr, 200, "admin/login.html", map[string]any{
		"Error": "",
	})

	if rr.Code != 200 {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "<html") {
		t.Errorf("expected HTML response, body starts with: %q", rr.Body.String()[:min(100, rr.Body.Len())])
	}
}

func TestRenderer_Standalone_404(t *testing.T) {
	r := NewRenderer(templatesDir)
	rr := httptest.NewRecorder()

	r.Standalone(rr, 404, "errors/404.html", nil)

	if rr.Code != 404 {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestRenderer_Page_Renders(t *testing.T) {
	r := NewRenderer(templatesDir)
	rr := httptest.NewRecorder()

	data := map[string]any{
		"Page":          "dashboard",
		"Title":         "Dashboard",
		"Session":       nil,
		"CSRFToken":     "",
		"Settings":      nil,
		"NewLeadsCount": 0,
		"Flash":         nil,
		// Dashboard-specific
		"TotalLeads":    0,
		"NewLeadsToday": 0,
		"TotalNews":     0,
		"BitrixEnabled": false,
		"RecentLeads":   nil,
	}

	r.Page(rr, "dashboard", data)

	if rr.Code != 200 {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String()[:min(200, rr.Body.Len())])
	}
	if !strings.Contains(rr.Body.String(), "<html") {
		t.Errorf("expected HTML response")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
