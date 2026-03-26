package handlers

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/lightcms/cms/internal/db"
	"github.com/yuin/goldmark"
)

var md = goldmark.New()

// renderMarkdown converts a Markdown string to HTML.
func renderMarkdown(src string) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// HandleAPINews handles GET /api/news?page=1&limit=10
func (s *Server) HandleAPINews(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	result, err := db.ListNews(s.db, db.NewsFilter{
		Search: q.Get("search"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]any{
			"status":  "error",
			"message": "Failed to fetch news",
		})
		return
	}

	pages := result.Total / limit
	if result.Total%limit != 0 {
		pages++
	}

	JSON(w, http.StatusOK, map[string]any{
		"data": result.News,
		"pagination": map[string]any{
			"page":  page,
			"limit": limit,
			"total": result.Total,
			"pages": pages,
		},
	})
}

// HandleAPINewsItem handles GET /api/news/{id}
func (s *Server) HandleAPINewsItem(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	idStr := r.PathValue("id")
	if idStr == "" {
		JSON(w, http.StatusBadRequest, map[string]any{
			"status":  "error",
			"message": "Missing news ID",
		})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSON(w, http.StatusBadRequest, map[string]any{
			"status":  "error",
			"message": "Invalid news ID",
		})
		return
	}

	n, err := db.GetNews(s.db, id)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]any{
			"status":  "error",
			"message": "Failed to fetch news",
		})
		return
	}
	if n == nil {
		JSON(w, http.StatusNotFound, map[string]any{
			"status":  "error",
			"message": "News not found",
		})
		return
	}

	descHTML, err := renderMarkdown(n.Description)
	if err != nil {
		descHTML = n.Description
	}

	type newsItem struct {
		ID              int    `json:"id"`
		Date            string `json:"date"`
		Title           string `json:"title"`
		Image           string `json:"image"`
		Announce        string `json:"announce"`
		Description     string `json:"description"`
		DescriptionHTML string `json:"description_html"`
		CreatedAt       string `json:"created_at"`
		UpdatedAt       string `json:"updated_at"`
	}

	JSON(w, http.StatusOK, newsItem{
		ID:              n.ID,
		Date:            n.Date.Format("2006-01-02"),
		Title:           n.Title,
		Image:           n.Image,
		Announce:        n.Announce,
		Description:     n.Description,
		DescriptionHTML: descHTML,
		CreatedAt:       n.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       n.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}
