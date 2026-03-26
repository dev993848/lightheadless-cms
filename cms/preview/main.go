// Preview server — renders admin templates with mock data.
// Run from cms/ directory: go run ./preview/
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

// ── Mock types ────────────────────────────────────────────────────────────────

type Flash struct {
	Type    string // success | error | warning
	Message string
}

type Lead struct {
	ID             int
	Name           string
	Phone          string
	Email          string
	Comment        string
	Status         string
	BitrixResponse string
	BitrixSentAt   time.Time
	CreatedAt      time.Time
}

type News struct {
	ID           int
	Date         time.Time
	Title        string
	Announce     string
	Description  string
	Image        string
	DateForInput string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Settings struct {
	SiteName        string
	AdminEmail      string
	Bitrix24Webhook string
	Bitrix24Enabled bool
}

type Filter struct {
	Search      string
	Status      string
	Limit       int
	HasFilters  bool
	QueryString string
}

type Pagination struct {
	Page  int
	Pages int
	Total int
	From  int
	To    int
}

func (p Pagination) QueryWithPage(page int) string { return fmt.Sprintf("page=%d", page) }
func (p Pagination) PageNumbers() []int {
	var out []int
	for i := 1; i <= p.Pages; i++ {
		out = append(out, i)
	}
	return out
}

// ── Template functions ────────────────────────────────────────────────────────

var funcMap = template.FuncMap{
	"formatDate": func(t time.Time) string {
		if t.IsZero() {
			return "—"
		}
		return t.Format("02.01.2006")
	},
	"formatDateTime": func(t time.Time) string {
		if t.IsZero() {
			return "—"
		}
		return t.Format("02.01.2006 15:04")
	},
	"dec":   func(i int) int { return i - 1 },
	"inc":   func(i int) int { return i + 1 },
	"slice": func(s string, i, j int) string {
		r := []rune(s)
		if i >= len(r) {
			return ""
		}
		if j > len(r) {
			j = len(r)
		}
		return string(r[i:j])
	},
}

// ── Template helpers ──────────────────────────────────────────────────────────

// loadPage parses base layout + partials + a specific page template.
// Returns a template that can be executed with name "base".
func loadPage(pageFile string) *template.Template {
	files := []string{
		"internal/templates/layouts/base.html",
		"internal/templates/partials/nav.html",
		"internal/templates/partials/flash.html",
		"internal/templates/partials/pagination.html",
		pageFile,
	}
	t, err := template.New("").Funcs(funcMap).ParseFiles(files...)
	if err != nil {
		log.Fatalf("template parse error (%s): %v", pageFile, err)
	}
	return t
}

func render(w http.ResponseWriter, t *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("render error: %v", err)
		http.Error(w, "render error:\n"+err.Error(), 500)
	}
}

// ── Mock data ─────────────────────────────────────────────────────────────────

var now = time.Now()

var mockLeads = []Lead{
	{ID: 1, Name: "Иван Петров", Phone: "+7 999 123-45-67", Email: "ivan@example.com", Comment: "Хочу узнать подробности о продукте", Status: "new", CreatedAt: now.Add(-2 * time.Hour)},
	{ID: 2, Name: "Мария Сидорова", Phone: "+7 916 987-65-43", Email: "maria@company.ru", Status: "sent", BitrixSentAt: now.Add(-1 * time.Hour), CreatedAt: now.Add(-5 * time.Hour)},
	{ID: 3, Name: "Алексей Козлов", Phone: "+7 903 555-11-22", Email: "", Comment: "Звоните после 18:00", Status: "error", CreatedAt: now.Add(-24 * time.Hour)},
	{ID: 4, Name: "Елена Новикова", Phone: "+7 812 333-44-55", Email: "elena@mail.ru", Status: "new", CreatedAt: now.Add(-48 * time.Hour)},
	{ID: 5, Name: "Дмитрий Волков", Phone: "+7 495 777-88-99", Email: "d.volkov@corp.com", Status: "sent", CreatedAt: now.Add(-72 * time.Hour)},
}

var mockNews = []News{
	{ID: 1, Date: now.Add(-24 * time.Hour), Title: "Запуск новой версии продукта", Announce: "Мы рады представить обновлённую версию с улучшенным интерфейсом и новыми функциями.", CreatedAt: now},
	{ID: 2, Date: now.Add(-48 * time.Hour), Title: "Партнёрство с ведущими компаниями рынка", Announce: "Заключены соглашения о сотрудничестве с крупнейшими игроками отрасли.", CreatedAt: now},
	{ID: 3, Date: now.Add(-72 * time.Hour), Title: "Итоги года: рост на 40%", Announce: "Компания завершила год с рекордными показателями роста выручки и клиентской базы.", CreatedAt: now},
}

var mockSettings = Settings{
	SiteName:        "LightCMS",
	AdminEmail:      "admin@example.com",
	Bitrix24Webhook: "https://domain.bitrix24.ru/rest/1/key/crm.lead.add.json",
	Bitrix24Enabled: true,
}

func pageBase(currentPage, title string, flash *Flash) map[string]any {
	return map[string]any{
		"SiteName":      mockSettings.SiteName,
		"PageTitle":     title,
		"CurrentPage":   currentPage,
		"Flash":         flash,
		"CSRFToken":     "preview-csrf-token",
		"AdminEmail":    mockSettings.AdminEmail,
		"NewLeadsCount": 2,
	}
}

func merge(base map[string]any, extra map[string]any) map[string]any {
	for k, v := range extra {
		base[k] = v
	}
	return base
}

// ── Page handlers ─────────────────────────────────────────────────────────────

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHTML)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	files := []string{"internal/templates/admin/login.html"}
	t, err := template.New("login.html").Funcs(funcMap).ParseFiles(files...)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, map[string]any{
		"SiteName":  mockSettings.SiteName,
		"CSRFToken": "preview-token",
		"Error":     "",
		"Email":     "admin@example.com",
	}); err != nil {
		log.Printf("login render error: %v", err)
	}
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	t := loadPage("internal/templates/admin/dashboard.html")
	render(w, t, merge(pageBase("dashboard", "Дашборд", nil), map[string]any{
		"TotalLeads":      42,
		"NewLeadsToday":   5,
		"TotalNews":       3,
		"Bitrix24Enabled": true,
		"RecentLeads":     mockLeads[:3],
	}))
}

func handleLeads(w http.ResponseWriter, r *http.Request) {
	flash := (*Flash)(nil)
	if r.URL.Query().Get("flash") == "1" {
		flash = &Flash{Type: "success", Message: "Заявка успешно удалена"}
	}
	t := loadPage("internal/templates/admin/leads.html")
	render(w, t, merge(pageBase("leads", "Заявки", flash), map[string]any{
		"Leads":      mockLeads,
		"Filter":     Filter{Status: "", Limit: 20, HasFilters: false},
		"Pagination": Pagination{Page: 1, Pages: 3, Total: 42, From: 1, To: 5},
	}))
}

func handleLeadDetail(w http.ResponseWriter, r *http.Request) {
	lead := mockLeads[2]
	lead.BitrixSentAt = now.Add(-30 * time.Minute)
	lead.BitrixResponse = `{"error":"INVALID_TOKEN","error_description":"Invalid access token"}`
	t := loadPage("internal/templates/admin/lead_detail.html")
	render(w, t, merge(pageBase("leads", "Заявка #3", nil), map[string]any{
		"Lead": lead,
	}))
}

func handleNews(w http.ResponseWriter, r *http.Request) {
	t := loadPage("internal/templates/admin/news.html")
	render(w, t, merge(pageBase("news", "Новости", nil), map[string]any{
		"News":       mockNews,
		"Pagination": Pagination{Page: 1, Pages: 1, Total: 3, From: 1, To: 3},
	}))
}

func handleNewsNew(w http.ResponseWriter, r *http.Request) {
	t := loadPage("internal/templates/admin/news_form.html")
	render(w, t, merge(pageBase("news", "Новая публикация", nil), map[string]any{
		"News":   News{DateForInput: now.Format("2006-01-02")},
		"Errors": map[string]string{},
	}))
}

func handleNewsEdit(w http.ResponseWriter, r *http.Request) {
	n := mockNews[0]
	n.DateForInput = n.Date.Format("2006-01-02")
	n.Description = "# Запуск новой версии\n\nМы рады представить **версию 2.0** нашего продукта.\n\n## Что нового\n\n- Улучшенный интерфейс\n- Быстрее на 40%\n- Новые интеграции"
	t := loadPage("internal/templates/admin/news_form.html")
	render(w, t, merge(pageBase("news", "Редактирование новости", nil), map[string]any{
		"News":   n,
		"Errors": map[string]string{},
	}))
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	t := loadPage("internal/templates/admin/settings.html")
	render(w, t, merge(pageBase("settings", "Настройки", nil), map[string]any{
		"Settings":   mockSettings,
		"Errors":     map[string]string{},
		"AppVersion": "v0.1.0-preview",
		"BuildTime":  now.Format("2006-01-02"),
	}))
}

// Redirect JS/CSS to CDN for preview
func handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "htmx.min.js"):
		http.Redirect(w, r, "https://unpkg.com/htmx.org@1.9.12/dist/htmx.min.js", http.StatusFound)
	case strings.HasSuffix(path, "alpine.min.js"):
		http.Redirect(w, r, "https://unpkg.com/alpinejs@3.14.1/dist/cdn.min.js", http.StatusFound)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	mux := http.NewServeMux()

	// Route all requests through one handler to avoid Go 1.22 routing quirks
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" || r.URL.Path == "":
			handleIndex(w, r)
		case r.URL.Path == "/login":
			handleLogin(w, r)
		case r.URL.Path == "/admin/" || r.URL.Path == "/admin":
			handleDashboard(w, r)
		case r.URL.Path == "/admin/leads":
			handleLeads(w, r)
		case r.URL.Path == "/admin/leads/detail":
			handleLeadDetail(w, r)
		case r.URL.Path == "/admin/news":
			handleNews(w, r)
		case r.URL.Path == "/admin/news/new":
			handleNewsNew(w, r)
		case r.URL.Path == "/admin/news/edit":
			handleNewsEdit(w, r)
		case r.URL.Path == "/admin/settings":
			handleSettings(w, r)
		case strings.HasPrefix(r.URL.Path, "/static/"):
			handleStatic(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	addr := "localhost:3000"
	log.Println("─────────────────────────────────────────")
	log.Printf("  Preview → http://%s/", addr)
	log.Println("─────────────────────────────────────────")
	log.Fatal(http.ListenAndServe(addr, mux))
}

// ── Index HTML ────────────────────────────────────────────────────────────────

const indexHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>LightCMS — Preview</title>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
<script src="https://cdn.tailwindcss.com"></script>
<script>tailwind.config={theme:{extend:{fontFamily:{sans:['Inter','sans-serif']}}}}</script>
</head>
<body class="bg-slate-950 font-sans min-h-screen flex items-center justify-center p-8">
<div class="w-full max-w-md">
  <div class="flex items-center gap-3 mb-8">
    <div class="w-10 h-10 rounded-xl bg-indigo-600 flex items-center justify-center shadow-lg shadow-indigo-600/40">
      <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 0 1 6 3.75h2.25A2.25 2.25 0 0 1 10.5 6v2.25a2.25 2.25 0 0 1-2.25 2.25H6a2.25 2.25 0 0 1-2.25-2.25V6ZM3.75 15.75A2.25 2.25 0 0 1 6 13.5h2.25a2.25 2.25 0 0 1 2.25 2.25V18a2.25 2.25 0 0 1-2.25 2.25H6A2.25 2.25 0 0 1 3.75 18v-2.25ZM13.5 6a2.25 2.25 0 0 1 2.25-2.25H18A2.25 2.25 0 0 1 20.25 6v2.25A2.25 2.25 0 0 1 18 10.5h-2.25a2.25 2.25 0 0 1-2.25-2.25V6ZM13.5 15.75a2.25 2.25 0 0 1 2.25-2.25H18a2.25 2.25 0 0 1 2.25 2.25V18A2.25 2.25 0 0 1 18 20.25h-2.25A2.25 2.25 0 0 1 13.5 18v-2.25Z"/>
      </svg>
    </div>
    <div>
      <h1 class="text-white font-semibold text-lg leading-tight">LightCMS Preview</h1>
      <p class="text-slate-500 text-sm">Выберите страницу</p>
    </div>
  </div>

  <div class="space-y-1.5">
    <p class="text-xs font-semibold uppercase tracking-wider text-slate-600 px-1 pb-1">Auth</p>
    <a href="/login" target="_blank" class="flex items-center justify-between w-full px-4 py-3 bg-slate-800/70 hover:bg-slate-800 rounded-xl transition-colors group">
      <span class="text-white text-sm font-medium">Страница входа</span>
      <span class="text-slate-500 group-hover:text-slate-400 font-mono text-xs">/login</span>
    </a>

    <p class="text-xs font-semibold uppercase tracking-wider text-slate-600 px-1 pt-3 pb-1">Admin</p>
    <a href="/admin/" target="_blank" class="flex items-center justify-between w-full px-4 py-3 bg-slate-800/70 hover:bg-slate-800 rounded-xl transition-colors group">
      <span class="text-white text-sm font-medium">Дашборд</span>
      <span class="text-slate-500 group-hover:text-slate-400 font-mono text-xs">/admin/</span>
    </a>
    <a href="/admin/leads" target="_blank" class="flex items-center justify-between w-full px-4 py-3 bg-slate-800/70 hover:bg-slate-800 rounded-xl transition-colors group">
      <span class="text-white text-sm font-medium">Заявки</span>
      <span class="text-slate-500 group-hover:text-slate-400 font-mono text-xs">/admin/leads</span>
    </a>
    <a href="/admin/leads?flash=1" target="_blank" class="flex items-center justify-between w-full px-4 py-3 bg-slate-800/70 hover:bg-slate-800 rounded-xl transition-colors group">
      <span class="text-white text-sm font-medium">Заявки + flash</span>
      <span class="text-slate-500 group-hover:text-slate-400 font-mono text-xs">?flash=1</span>
    </a>
    <a href="/admin/leads/detail" target="_blank" class="flex items-center justify-between w-full px-4 py-3 bg-slate-800/70 hover:bg-slate-800 rounded-xl transition-colors group">
      <span class="text-white text-sm font-medium">Детали заявки</span>
      <span class="text-slate-500 group-hover:text-slate-400 font-mono text-xs">/admin/leads/detail</span>
    </a>
    <a href="/admin/news" target="_blank" class="flex items-center justify-between w-full px-4 py-3 bg-slate-800/70 hover:bg-slate-800 rounded-xl transition-colors group">
      <span class="text-white text-sm font-medium">Новости</span>
      <span class="text-slate-500 group-hover:text-slate-400 font-mono text-xs">/admin/news</span>
    </a>
    <a href="/admin/news/new" target="_blank" class="flex items-center justify-between w-full px-4 py-3 bg-slate-800/70 hover:bg-slate-800 rounded-xl transition-colors group">
      <span class="text-white text-sm font-medium">Создание новости</span>
      <span class="text-slate-500 group-hover:text-slate-400 font-mono text-xs">/admin/news/new</span>
    </a>
    <a href="/admin/news/edit" target="_blank" class="flex items-center justify-between w-full px-4 py-3 bg-slate-800/70 hover:bg-slate-800 rounded-xl transition-colors group">
      <span class="text-white text-sm font-medium">Редактирование новости</span>
      <span class="text-slate-500 group-hover:text-slate-400 font-mono text-xs">/admin/news/edit</span>
    </a>
    <a href="/admin/settings" target="_blank" class="flex items-center justify-between w-full px-4 py-3 bg-slate-800/70 hover:bg-slate-800 rounded-xl transition-colors group">
      <span class="text-white text-sm font-medium">Настройки</span>
      <span class="text-slate-500 group-hover:text-slate-400 font-mono text-xs">/admin/settings</span>
    </a>
  </div>

  <p class="text-center text-slate-700 text-xs mt-8">Каждая страница открывается в новой вкладке</p>
</div>
</body>
</html>`
