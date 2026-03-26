package handlers

import (
	"net/http"

	"github.com/lightcms/cms/internal/db"
)

// HandleDashboard handles GET /admin/ and renders the dashboard page.
func (s *Server) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	data := baseData(r, w, s.db, "dashboard", "Панель управления")

	totalLeads, _ := db.ListLeads(s.db, db.LeadsFilter{Limit: 1})
	newLeadsToday, _ := db.CountCreatedToday(s.db)
	totalNews, _ := db.CountNews(s.db)
	recentLeads, _ := db.ListRecentLeads(s.db, 5)

	settings, _ := db.Get(s.db)

	data["TotalLeads"] = 0
	if totalLeads != nil {
		data["TotalLeads"] = totalLeads.Total
	}
	data["NewLeadsToday"] = newLeadsToday
	data["TotalNews"] = totalNews
	data["RecentLeads"] = recentLeads
	data["Bitrix24Enabled"] = settings != nil && settings.Bitrix24Enabled

	s.renderer.Page(w, "dashboard", data)
}
