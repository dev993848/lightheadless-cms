package handlers

import (
	"database/sql"
	"net/http"

	"github.com/lightcms/cms/internal/config"
	"github.com/lightcms/cms/internal/middleware"
	"github.com/lightcms/cms/internal/services"
)

// Server holds the application dependencies for HTTP handlers.
type Server struct {
	db        *sql.DB
	cfg       *config.Config
	renderer  *Renderer
	bitrix    *services.BitrixPool
	uploader  *services.Uploader
	version   string
	buildTime string
}

// NewRouter creates the Server, configures routes, and returns the root http.Handler.
func NewRouter(
	db *sql.DB,
	cfg *config.Config,
	bitrix *services.BitrixPool,
	version, buildTime string,
) http.Handler {
	s := &Server{
		db:        db,
		cfg:       cfg,
		renderer:  NewRenderer("./ui/templates"),
		bitrix:    bitrix,
		uploader:  services.NewUploader(cfg.UploadPath),
		version:   version,
		buildTime: buildTime,
	}

	mux := http.NewServeMux()

	// Static assets.
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	
	// Redirect /static/ to /static/index.html
	mux.HandleFunc("GET /static/index", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/static/" || r.URL.Path == "/static/index" {
			http.Redirect(w, r, "/static/index.html", http.StatusMovedPermanently)
			return
		}
		http.NotFound(w, r)
	})
	
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadPath))))

	// Root redirect.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/admin/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// ── Public API ────────────────────────────────────────────────────────────
	rl := middleware.NewRateLimiter()

	mux.HandleFunc("GET /api/news", s.HandleAPINews)
	mux.HandleFunc("GET /api/news/{id}", s.HandleAPINewsItem)
	mux.HandleFunc("POST /api/leads", func(w http.ResponseWriter, r *http.Request) {
		rl.Middleware(http.HandlerFunc(s.HandleAPILeads)).ServeHTTP(w, r)
	})
	mux.HandleFunc("OPTIONS /api/leads", s.HandleAPILeads)

	// ── Auth ──────────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /admin/login", s.HandleLoginPage)
	mux.HandleFunc("POST /admin/login", s.HandleLogin)
	mux.HandleFunc("POST /admin/logout", middleware.Chain(
		http.HandlerFunc(s.HandleLogout),
		middleware.Auth(db),
	).ServeHTTP)

	// ── Admin: Dashboard ──────────────────────────────────────────────────────
	mux.Handle("GET /admin/", middleware.Chain(
		http.HandlerFunc(s.HandleDashboard),
		middleware.Auth(db),
	))

	// ── Admin: Leads ──────────────────────────────────────────────────────────
	mux.Handle("GET /admin/leads", middleware.Chain(
		http.HandlerFunc(s.HandleLeads),
		middleware.Auth(db),
	))
	mux.Handle("GET /admin/leads/export.csv", middleware.Chain(
		http.HandlerFunc(s.HandleLeadsExport),
		middleware.Auth(db),
	))
	mux.Handle("GET /admin/leads/{id}", middleware.Chain(
		http.HandlerFunc(s.HandleLeadDetail),
		middleware.Auth(db),
	))
	mux.Handle("DELETE /admin/leads/{id}", middleware.Chain(
		http.HandlerFunc(s.HandleLeadDelete),
		middleware.Auth(db),
		middleware.Middleware,
	))
	mux.Handle("POST /admin/leads/{id}/resend", middleware.Chain(
		http.HandlerFunc(s.HandleLeadResend),
		middleware.Auth(db),
		middleware.Middleware,
	))
	mux.Handle("POST /admin/leads/{id}", middleware.Chain(
		http.HandlerFunc(s.HandleLeadStatusUpdate),
		middleware.Auth(db),
		middleware.Middleware,
	))

	// ── Admin: News ───────────────────────────────────────────────────────────
	mux.Handle("GET /admin/news", middleware.Chain(
		http.HandlerFunc(s.HandleNews),
		middleware.Auth(db),
	))
	mux.Handle("GET /admin/news/new", middleware.Chain(
		http.HandlerFunc(s.HandleNewsNew),
		middleware.Auth(db),
	))
	mux.Handle("POST /admin/news", middleware.Chain(
		http.HandlerFunc(s.HandleNewsCreate),
		middleware.Auth(db),
		middleware.Middleware,
	))
	mux.Handle("POST /admin/news/preview", middleware.Chain(
		http.HandlerFunc(s.HandleNewsPreview),
		middleware.Auth(db),
	))
	mux.Handle("GET /admin/news/{id}/edit", middleware.Chain(
		http.HandlerFunc(s.HandleNewsEdit),
		middleware.Auth(db),
	))
	mux.Handle("POST /admin/news/{id}", middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Dispatch based on _method override.
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				r.ParseForm()
			}
			switch methodOverride(r) {
			case http.MethodDelete:
				s.HandleNewsDelete(w, r)
			default:
				s.HandleNewsUpdate(w, r)
			}
		}),
		middleware.Auth(db),
		middleware.Middleware,
	))
	mux.Handle("DELETE /admin/news/{id}", middleware.Chain(
		http.HandlerFunc(s.HandleNewsDelete),
		middleware.Auth(db),
		middleware.Middleware,
	))

	// ── Admin: Settings ───────────────────────────────────────────────────────
	mux.Handle("GET /admin/settings", middleware.Chain(
		http.HandlerFunc(s.HandleSettings),
		middleware.Auth(db),
	))
	mux.Handle("POST /admin/settings", middleware.Chain(
		http.HandlerFunc(s.HandleSettingsSave),
		middleware.Auth(db),
		middleware.Middleware,
	))
	mux.Handle("POST /admin/settings/test-bitrix", middleware.Chain(
		http.HandlerFunc(s.HandleSettingsTestBitrix),
		middleware.Auth(db),
		middleware.Middleware,
	))

	// Wrap the entire mux with the request logger.
	return middleware.Logger(mux)
}
