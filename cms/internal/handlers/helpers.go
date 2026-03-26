package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/lightcms/cms/internal/db"
	"github.com/lightcms/cms/internal/middleware"
)

// baseData builds the common data map needed by all admin pages.
// It queries settings, new lead count, CSRF token, and flash messages.
func baseData(r *http.Request, w http.ResponseWriter, database *sql.DB, page, title string) map[string]any {
	session := middleware.GetSession(r.Context())

	csrfToken := ""
	if session != nil {
		csrfToken = middleware.GenerateToken(session.ID)
	}

	settings, _ := db.Get(database)
	newLeadsCount, _ := db.CountByStatus(database, "new")
	flash := GetFlash(r, w)

	data := map[string]any{
		"Page":          page,
		"Title":         title,
		"Session":       session,
		"CSRFToken":     csrfToken,
		"Settings":      settings,
		"NewLeadsCount": newLeadsCount,
		"Flash":         flash,
	}
	return data
}

// PaginationData holds computed pagination metadata.
type PaginationData struct {
	Page  int
	Pages int
	Total int
	From  int
	To    int
	query url.Values
}

// QueryWithPage returns the current URL query string with the page parameter replaced.
func (p PaginationData) QueryWithPage(pg int) string {
	q := cloneValues(p.query)
	q.Set("page", strconv.Itoa(pg))
	return "?" + q.Encode()
}

// PageNumbers returns a slice of page numbers to display, with -1 representing ellipsis.
func (p PaginationData) PageNumbers() []int {
	if p.Pages <= 7 {
		pages := make([]int, p.Pages)
		for i := range pages {
			pages[i] = i + 1
		}
		return pages
	}

	var result []int
	result = append(result, 1)

	if p.Page > 3 {
		result = append(result, -1) // ellipsis
	}

	start := p.Page - 1
	if start < 2 {
		start = 2
	}
	end := p.Page + 1
	if end > p.Pages-1 {
		end = p.Pages - 1
	}

	for i := start; i <= end; i++ {
		result = append(result, i)
	}

	if p.Page < p.Pages-2 {
		result = append(result, -1) // ellipsis
	}

	result = append(result, p.Pages)
	return result
}

// newPagination constructs a PaginationData from total count, current page, limit per page,
// and the current URL query values.
func newPagination(total, page, limit int, q url.Values) PaginationData {
	if limit <= 0 {
		limit = 20
	}
	pages := total / limit
	if total%limit != 0 {
		pages++
	}
	if pages < 1 {
		pages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}

	from := (page-1)*limit + 1
	to := page * limit
	if to > total {
		to = total
	}
	if total == 0 {
		from = 0
	}

	return PaginationData{
		Page:  page,
		Pages: pages,
		Total: total,
		From:  from,
		To:    to,
		query: q,
	}
}

// cloneValues creates a deep copy of url.Values.
func cloneValues(v url.Values) url.Values {
	out := make(url.Values)
	for k, vals := range v {
		out[k] = append([]string(nil), vals...)
	}
	return out
}

// methodOverride returns the effective HTTP method.
// On POST, checks the "_method" form field for overrides (PUT, PATCH, DELETE).
func methodOverride(r *http.Request) string {
	method := r.Method
	if method == http.MethodPost {
		override := strings.ToUpper(r.FormValue("_method"))
		switch override {
		case http.MethodPut, http.MethodPatch, http.MethodDelete:
			return override
		}
	}
	return method
}

// parseIDFromPath extracts an integer ID segment from a URL path.
// It trims the given prefix and any trailing suffix (like "/edit", "/resend").
func parseIDFromPath(path, prefix string) (int, error) {
	s := strings.TrimPrefix(path, prefix)
	// Remove any trailing path segments.
	if idx := strings.Index(s, "/"); idx != -1 {
		s = s[:idx]
	}
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q: %w", s, err)
	}
	return id, nil
}
