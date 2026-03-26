package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/lightcms/cms/internal/db"
	"github.com/lightcms/cms/internal/models"
	"github.com/lightcms/cms/internal/validate"
)

// setCORSHeaders sets permissive CORS headers for API endpoints.
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// HandleAPILeads handles POST /api/leads and OPTIONS /api/leads.
func (s *Server) HandleAPILeads(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var input validate.LeadInput

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
		if err != nil {
			JSON(w, http.StatusBadRequest, map[string]any{
				"status":  "error",
				"message": "Failed to read request body",
			})
			return
		}
		if err := json.Unmarshal(body, &input); err != nil {
			JSON(w, http.StatusBadRequest, map[string]any{
				"status":  "error",
				"message": "Invalid JSON body",
			})
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			JSON(w, http.StatusBadRequest, map[string]any{
				"status":  "error",
				"message": "Failed to parse form",
			})
			return
		}
		input = validate.LeadInput{
			Name:    r.FormValue("name"),
			Phone:   r.FormValue("phone"),
			Email:   r.FormValue("email"),
			Comment: r.FormValue("comment"),
		}
	}

	sanitized, errs := validate.Lead(input)
	if len(errs) > 0 {
		errList := make([]map[string]string, len(errs))
		for i, e := range errs {
			errList[i] = map[string]string{"field": e.Field, "message": e.Message}
		}
		JSON(w, http.StatusUnprocessableEntity, map[string]any{
			"status":  "error",
			"message": "Validation failed",
			"errors":  errList,
		})
		return
	}

	lead := &models.Lead{
		Name:    sanitized.Name,
		Phone:   sanitized.Phone,
		Email:   sanitized.Email,
		Comment: sanitized.Comment,
		Status:  models.StatusNew,
	}

	id, err := db.CreateLead(s.db, lead)
	if err != nil {
		JSON(w, http.StatusInternalServerError, map[string]any{
			"status":  "error",
			"message": "Failed to save lead",
		})
		return
	}
	lead.ID = int(id)

	// Submit to Bitrix24 asynchronously if pool is available.
	if s.bitrix != nil {
		s.bitrix.Submit(*lead)
	}

	JSON(w, http.StatusCreated, map[string]any{
		"status":  "success",
		"message": "Заявка принята",
		"id":      id,
	})
}
