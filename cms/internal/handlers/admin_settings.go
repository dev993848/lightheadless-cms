package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lightcms/cms/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// HandleSettings handles GET /admin/settings.
func (s *Server) HandleSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := db.Get(s.db)
	if err != nil {
		http.Error(w, "Failed to load settings", http.StatusInternalServerError)
		return
	}

	data := baseData(r, w, s.db, "settings", "Настройки")
	data["Settings"] = settings

	s.renderer.Page(w, "settings", data)
}

// HandleSettingsSave handles POST /admin/settings.
// Uses _section field to determine which section is being saved: "general" or "bitrix24".
func (s *Server) HandleSettingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	settings, err := db.Get(s.db)
	if err != nil {
		http.Error(w, "Failed to load settings", http.StatusInternalServerError)
		return
	}

	section := r.FormValue("_section")

	switch section {
	case "general":
		siteName := strings.TrimSpace(r.FormValue("site_name"))
		adminEmail := strings.TrimSpace(r.FormValue("admin_email"))
		newPassword := r.FormValue("admin_password")
		confirmPassword := r.FormValue("admin_password_confirm")

		if siteName != "" {
			settings.SiteName = siteName
		}
		if adminEmail != "" {
			settings.AdminEmail = adminEmail
		}

		if newPassword != "" {
			if newPassword != confirmPassword {
				data := baseData(r, w, s.db, "settings", "Настройки")
				data["Settings"] = settings
				data["Error"] = "Пароли не совпадают"
				s.renderer.Page(w, "settings", data)
				return
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				data := baseData(r, w, s.db, "settings", "Настройки")
				data["Settings"] = settings
				data["Error"] = "Не удалось хешировать пароль"
				s.renderer.Page(w, "settings", data)
				return
			}
			settings.AdminPassword = string(hash)
		}

	case "bitrix24":
		webhook := strings.TrimSpace(r.FormValue("bitrix24_webhook"))
		enabled := r.FormValue("bitrix24_enabled") == "1" || r.FormValue("bitrix24_enabled") == "on"

		settings.Bitrix24Webhook = webhook
		settings.Bitrix24Enabled = enabled

	default:
		// Save all fields if no section specified.
		siteName := strings.TrimSpace(r.FormValue("site_name"))
		adminEmail := strings.TrimSpace(r.FormValue("admin_email"))
		if siteName != "" {
			settings.SiteName = siteName
		}
		if adminEmail != "" {
			settings.AdminEmail = adminEmail
		}
	}

	if err := db.Upsert(s.db, settings); err != nil {
		data := baseData(r, w, s.db, "settings", "Настройки")
		data["Settings"] = settings
		data["Error"] = "Не удалось сохранить настройки: " + err.Error()
		s.renderer.Page(w, "settings", data)
		return
	}

	SetFlash(w, Flash{Type: "success", Message: "Настройки сохранены"})
	http.Redirect(w, r, "/admin/settings", http.StatusFound)
}

// HandleSettingsTestBitrix handles POST /admin/settings/test-bitrix.
// Sends a test request to the configured Bitrix24 webhook and returns an HTML fragment.
func (s *Server) HandleSettingsTestBitrix(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div class="alert alert-danger">Ошибка обработки формы</div>`)
		return
	}

	webhookURL := strings.TrimSpace(r.FormValue("bitrix24_webhook"))
	if webhookURL == "" {
		settings, _ := db.Get(s.db)
		if settings != nil {
			webhookURL = settings.Bitrix24Webhook
		}
	}

	if webhookURL == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div class="alert alert-warning">URL webhook не задан</div>`)
		return
	}

	testPayload := map[string]any{
		"fields": map[string]any{
			"NAME":     "Тестовая заявка",
			"PHONE":    []map[string]string{{"VALUE": "+70000000000", "VALUE_TYPE": "WORK"}},
			"COMMENTS": "Тест подключения Bitrix24 из CMS",
			"TITLE":    "Тест " + time.Now().Format("02.01.2006 15:04"),
		},
	}

	body, _ := json.Marshal(testPayload)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div class="alert alert-danger">Ошибка создания запроса: %s</div>`, htmlEscape(err.Error()))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<div class="alert alert-danger">Ошибка подключения: %s</div>`, htmlEscape(err.Error()))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if resp.StatusCode >= 400 {
		fmt.Fprintf(w,
			`<div class="alert alert-danger">Ошибка %d: %s</div>`,
			resp.StatusCode, htmlEscape(string(respBody)),
		)
	} else {
		fmt.Fprintf(w, `<div class="alert alert-success">Успешно! Ответ: %s</div>`, htmlEscape(string(respBody)))
	}
}

// htmlEscape escapes HTML special characters for safe output in templates.
func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

