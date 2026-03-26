package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lightcms/cms/internal/config"
	"github.com/lightcms/cms/internal/db"
	_ "modernc.org/sqlite"
)

func setupTestServer(t *testing.T) (*Server, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		Port:       8080,
		DBPath:     ":memory:",
		UploadPath: t.TempDir(),
	}

	s := &Server{
		db:  database,
		cfg: cfg,
	}
	return s, database
}

func TestHandleAPILeads_JSON_Valid(t *testing.T) {
	s, _ := setupTestServer(t)

	body := `{"name":"Иван Петров","phone":"+79001234567","email":"ivan@example.com","comment":"Тест"}`
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.HandleAPILeads(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["status"] != "success" {
		t.Errorf("expected status=success, got %v", resp["status"])
	}
}

func TestHandleAPILeads_Form_Valid(t *testing.T) {
	s, _ := setupTestServer(t)

	body := "name=Мария+Сидорова&phone=%2B79009999999"
	req := httptest.NewRequest(http.MethodPost, "/api/leads", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	s.HandleAPILeads(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleAPILeads_MissingFields(t *testing.T) {
	s, _ := setupTestServer(t)

	body := `{"name":"","phone":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.HandleAPILeads(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["status"] != "error" {
		t.Errorf("expected status=error, got %v", resp["status"])
	}
	if resp["errors"] == nil {
		t.Error("expected errors in response")
	}
}

func TestHandleAPILeads_InvalidJSON(t *testing.T) {
	s, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/leads", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.HandleAPILeads(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleAPILeads_OPTIONS(t *testing.T) {
	s, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/leads", nil)
	rr := httptest.NewRecorder()

	s.HandleAPILeads(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for OPTIONS, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header")
	}
}

func TestHandleAPILeads_CORS_Headers(t *testing.T) {
	s, _ := setupTestServer(t)

	body := `{"name":"Test","phone":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.HandleAPILeads(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header on POST response")
	}
}
