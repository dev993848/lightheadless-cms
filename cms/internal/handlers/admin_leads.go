package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lightcms/cms/internal/db"
	"github.com/lightcms/cms/internal/models"
)

// HandleLeads handles GET /admin/leads with search, status filter, and pagination.
func (s *Server) HandleLeads(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	filter := db.LeadsFilter{
		Search: q.Get("search"),
		Status: q.Get("status"),
		Limit:  limit,
		Offset: offset,
	}

	result, err := db.ListLeads(s.db, filter)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	pagination := newPagination(result.Total, page, limit, q)

	data := baseData(r, w, s.db, "leads", "Заявки")
	data["Leads"] = result.Leads
	data["Pagination"] = pagination
	data["Filter"] = filter

	// HTMX partial refresh: only render the table fragment.
	if r.Header.Get("HX-Request") == "true" {
		s.renderer.Partial(w, "leads", "leads_table", data)
		return
	}

	s.renderer.Page(w, "leads", data)
}

// HandleLeadDetail handles GET /admin/leads/{id}.
func (s *Server) HandleLeadDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid lead ID", http.StatusBadRequest)
		return
	}

	lead, err := db.GetLead(s.db, id)
	if err != nil || lead == nil {
		http.NotFound(w, r)
		return
	}

	data := baseData(r, w, s.db, "lead_detail", "Заявка #"+strconv.Itoa(id))
	data["Lead"] = lead

	s.renderer.Page(w, "lead_detail", data)
}

// HandleLeadDelete handles DELETE /admin/leads/{id}.
// Returns 200 with an empty body (HTMX removes the row client-side).
func (s *Server) HandleLeadDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid lead ID", http.StatusBadRequest)
		return
	}

	if err := db.DeleteLead(s.db, id); err != nil {
		http.Error(w, "Failed to delete lead", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleLeadResend handles POST /admin/leads/{id}/resend.
// Re-submits the lead to the Bitrix24 queue and returns an HTMX partial.
func (s *Server) HandleLeadResend(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid lead ID", http.StatusBadRequest)
		return
	}

	lead, err := db.GetLead(s.db, id)
	if err != nil || lead == nil {
		http.NotFound(w, r)
		return
	}

	if s.bitrix != nil {
		s.bitrix.Submit(*lead)
	}

	// Update status to "new" so it gets picked up.
	_ = db.UpdateLeadStatus(s.db, id, models.StatusNew)

	// Return updated status badge fragment.
	data := baseData(r, w, s.db, "leads", "Заявки")
	data["Lead"] = lead
	data["Status"] = models.StatusNew

	s.renderer.Partial(w, "leads", "status_badge", data)
}

// HandleLeadStatusUpdate handles POST /admin/leads/{id} with _method=PATCH.
func (s *Server) HandleLeadStatusUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid lead ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")
	switch status {
	case models.StatusNew, models.StatusSent, models.StatusError:
	default:
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	if err := db.UpdateLeadStatus(s.db, id, status); err != nil {
		http.Error(w, "Failed to update status", http.StatusInternalServerError)
		return
	}

	SetFlash(w, Flash{Type: "success", Message: "Статус заявки обновлён"})
	http.Redirect(w, r, "/admin/leads/"+idStr, http.StatusFound)
}

// HandleLeadsExport handles GET /admin/leads/export.csv.
// Streams all leads matching current filters as a CSV file.
func (s *Server) HandleLeadsExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := db.LeadsFilter{
		Search: q.Get("search"),
		Status: q.Get("status"),
		Limit:  1<<31 - 1, // All rows.
		Offset: 0,
	}

	date := time.Now().Format("2006-01-02")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="leads-`+date+`.csv"`)

	cw := csv.NewWriter(w)

	// Write UTF-8 BOM for Excel compatibility.
	w.Write([]byte("\xef\xbb\xbf"))

	// Header row.
	_ = cw.Write([]string{"ID", "Имя", "Телефон", "Email", "Комментарий", "Статус", "Ответ Bitrix24", "Отправлено в Bitrix", "Дата создания"})

	result, err := db.ListLeads(s.db, filter)
	if err != nil {
		return
	}

	for _, lead := range result.Leads {
		bitrixSentAt := ""
		if lead.BitrixSentAt != nil {
			bitrixSentAt = lead.BitrixSentAt.Format("02.01.2006 15:04")
		}
		_ = cw.Write([]string{
			strconv.Itoa(lead.ID),
			lead.Name,
			lead.Phone,
			lead.Email,
			lead.Comment,
			lead.Status,
			lead.BitrixResponse,
			bitrixSentAt,
			lead.CreatedAt.Format("02.01.2006 15:04"),
		})
	}

	cw.Flush()
}

// parseLeadIDFromPath parses the lead ID from paths like /admin/leads/123 or /admin/leads/123/resend.
func parseLeadIDFromPath(path string) (int, string) {
	s := strings.TrimPrefix(path, "/admin/leads/")
	parts := strings.SplitN(s, "/", 2)
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, ""
	}
	suffix := ""
	if len(parts) > 1 {
		suffix = parts[1]
	}
	return id, suffix
}
