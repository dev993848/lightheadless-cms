package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lightcms/cms/internal/db"
	"github.com/lightcms/cms/internal/models"
)

func TestHandleAPINews_Empty(t *testing.T) {
	s, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/news", nil)
	rr := httptest.NewRecorder()
	s.HandleAPINews(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected data field")
	}
}

func TestHandleAPINews_WithItems(t *testing.T) {
	s, database := setupTestServer(t)

	_, _ = db.CreateNews(database, &models.News{
		Date:        time.Now(),
		Title:       "Тест новости",
		Announce:    "Краткое описание",
		Description: "Полное **описание**",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/news", nil)
	rr := httptest.NewRecorder()
	s.HandleAPINews(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	pagination := resp["pagination"].(map[string]interface{})
	if pagination["total"].(float64) != 1 {
		t.Errorf("expected total=1, got %v", pagination["total"])
	}
}

func TestHandleAPINews_Pagination(t *testing.T) {
	s, database := setupTestServer(t)

	for i := 0; i < 5; i++ {
		db.CreateNews(database, &models.News{
			Date:  time.Now(),
			Title: "Новость",
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/news?page=2&limit=2", nil)
	rr := httptest.NewRecorder()
	s.HandleAPINews(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	pagination := resp["pagination"].(map[string]interface{})
	if pagination["total"].(float64) != 5 {
		t.Errorf("expected total=5, got %v", pagination["total"])
	}
	if pagination["pages"].(float64) != 3 {
		t.Errorf("expected pages=3, got %v", pagination["pages"])
	}
}

func TestHandleAPINewsItem_Found(t *testing.T) {
	s, database := setupTestServer(t)

	id, _ := db.CreateNews(database, &models.News{
		Date:        time.Now(),
		Title:       "Заголовок новости",
		Description: "# Markdown заголовок",
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/news/{id}", s.HandleAPINewsItem)

	req := httptest.NewRequest(http.MethodGet, "/api/news/"+itoa(int(id)), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["title"] != "Заголовок новости" {
		t.Errorf("expected title match, got %v", resp["title"])
	}
	if resp["description_html"] == "" {
		t.Error("expected non-empty description_html")
	}
}

func TestHandleAPINewsItem_NotFound(t *testing.T) {
	s, _ := setupTestServer(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/news/{id}", s.HandleAPINewsItem)

	req := httptest.NewRequest(http.MethodGet, "/api/news/9999", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestHandleAPINewsItem_InvalidID(t *testing.T) {
	s, _ := setupTestServer(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/news/{id}", s.HandleAPINewsItem)

	req := httptest.NewRequest(http.MethodGet, "/api/news/abc", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// itoa converts int to string (avoids strconv import conflict).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}
