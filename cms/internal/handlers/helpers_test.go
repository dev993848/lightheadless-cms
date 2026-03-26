package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// ── PageNumbers ────────────────────────────────────────────────────────────

func TestPageNumbers_FewPages(t *testing.T) {
	p := PaginationData{Page: 1, Pages: 5}
	nums := p.PageNumbers()
	if len(nums) != 5 {
		t.Errorf("expected 5 page numbers, got %d", len(nums))
	}
	for i, n := range nums {
		if n != i+1 {
			t.Errorf("expected page %d, got %d", i+1, n)
		}
	}
}

func TestPageNumbers_ManyPages_Start(t *testing.T) {
	p := PaginationData{Page: 1, Pages: 20}
	nums := p.PageNumbers()
	// Should start with 1 and end with 20, contain -1 for ellipsis.
	if nums[0] != 1 {
		t.Error("first page should be 1")
	}
	if nums[len(nums)-1] != 20 {
		t.Error("last page should be 20")
	}
	hasEllipsis := false
	for _, n := range nums {
		if n == -1 {
			hasEllipsis = true
		}
	}
	if !hasEllipsis {
		t.Error("expected ellipsis (-1) in page numbers")
	}
}

func TestPageNumbers_ManyPages_Middle(t *testing.T) {
	p := PaginationData{Page: 10, Pages: 20}
	nums := p.PageNumbers()
	if nums[0] != 1 {
		t.Error("first should always be 1")
	}
	if nums[len(nums)-1] != 20 {
		t.Error("last should always be 20")
	}
	// Should have two ellipses.
	ellipses := 0
	for _, n := range nums {
		if n == -1 {
			ellipses++
		}
	}
	if ellipses != 2 {
		t.Errorf("expected 2 ellipses for middle page, got %d", ellipses)
	}
}

func TestPageNumbers_ManyPages_End(t *testing.T) {
	p := PaginationData{Page: 20, Pages: 20}
	nums := p.PageNumbers()
	if nums[0] != 1 {
		t.Error("first page should be 1")
	}
	if nums[len(nums)-1] != 20 {
		t.Error("last page should be 20")
	}
}

// ── QueryWithPage ──────────────────────────────────────────────────────────

func TestQueryWithPage(t *testing.T) {
	q := url.Values{}
	q.Set("status", "new")
	q.Set("page", "1")
	p := PaginationData{Page: 1, Pages: 5, query: q}

	result := p.QueryWithPage(3)
	if !strings.Contains(result, "page=3") {
		t.Errorf("expected page=3 in query, got %q", result)
	}
	if !strings.Contains(result, "status=new") {
		t.Errorf("expected status=new preserved in query, got %q", result)
	}
}

// ── newPagination ──────────────────────────────────────────────────────────

func TestNewPagination_Basic(t *testing.T) {
	p := newPagination(100, 2, 10, nil)
	if p.Pages != 10 {
		t.Errorf("expected 10 pages, got %d", p.Pages)
	}
	if p.From != 11 {
		t.Errorf("expected from=11, got %d", p.From)
	}
	if p.To != 20 {
		t.Errorf("expected to=20, got %d", p.To)
	}
}

func TestNewPagination_Empty(t *testing.T) {
	p := newPagination(0, 1, 20, nil)
	if p.From != 0 {
		t.Errorf("expected from=0 for empty result, got %d", p.From)
	}
	if p.Pages != 1 {
		t.Errorf("expected pages=1 for empty result, got %d", p.Pages)
	}
}

func TestNewPagination_ClampPage(t *testing.T) {
	p := newPagination(50, 100, 10, nil) // page 100 > pages 5
	if p.Page != 5 {
		t.Errorf("expected page clamped to 5, got %d", p.Page)
	}
}

func TestNewPagination_DefaultLimit(t *testing.T) {
	p := newPagination(100, 1, 0, nil) // limit=0 should default to 20
	if p.Pages != 5 {
		t.Errorf("expected 5 pages with default limit 20, got %d", p.Pages)
	}
}

// ── methodOverride ─────────────────────────────────────────────────────────

func TestMethodOverride_GET_Unchanged(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if m := methodOverride(req); m != http.MethodGet {
		t.Errorf("GET should not be overridden, got %q", m)
	}
}

func TestMethodOverride_POST_Delete(t *testing.T) {
	body := "_method=DELETE"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if m := methodOverride(req); m != http.MethodDelete {
		t.Errorf("expected DELETE, got %q", m)
	}
}

func TestMethodOverride_POST_Put(t *testing.T) {
	body := "_method=PUT"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if m := methodOverride(req); m != http.MethodPut {
		t.Errorf("expected PUT, got %q", m)
	}
}

func TestMethodOverride_POST_Invalid(t *testing.T) {
	body := "_method=HACK"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if m := methodOverride(req); m != http.MethodPost {
		t.Errorf("invalid override should return POST, got %q", m)
	}
}

// ── parseIDFromPath ────────────────────────────────────────────────────────

func TestParseIDFromPath_Valid(t *testing.T) {
	id, err := parseIDFromPath("/admin/leads/42", "/admin/leads/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("expected 42, got %d", id)
	}
}

func TestParseIDFromPath_WithSuffix(t *testing.T) {
	id, err := parseIDFromPath("/admin/leads/7/resend", "/admin/leads/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 7 {
		t.Errorf("expected 7, got %d", id)
	}
}

func TestParseIDFromPath_Invalid(t *testing.T) {
	_, err := parseIDFromPath("/admin/leads/abc", "/admin/leads/")
	if err == nil {
		t.Error("expected error for non-numeric ID")
	}
}

// ── cloneValues ────────────────────────────────────────────────────────────

func TestCloneValues_Independence(t *testing.T) {
	orig := url.Values{"key": {"value1", "value2"}}
	clone := cloneValues(orig)
	clone["key"][0] = "modified"
	if orig["key"][0] == "modified" {
		t.Error("clone should not share slices with original")
	}
}
