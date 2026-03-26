package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lightcms/cms/internal/models"
)

// LeadsFilter defines filtering options for listing leads.
type LeadsFilter struct {
	Search string
	Status string
	Limit  int
	Offset int
}

// QueryString returns the filter parameters as a URL-encoded query string.
func (f LeadsFilter) QueryString() string {
	params := url.Values{}
	if f.Search != "" {
		params.Set("search", f.Search)
	}
	if f.Status != "" {
		params.Set("status", f.Status)
	}
	if f.Limit > 0 && f.Limit != 20 {
		params.Set("limit", strconv.Itoa(f.Limit))
	}
	return params.Encode()
}

// HasFilters returns true if any filter is active.
func (f LeadsFilter) HasFilters() bool {
	return f.Search != "" || f.Status != ""
}

// LeadsResult holds a page of leads and the total count matching the filter.
type LeadsResult struct {
	Leads []models.Lead
	Total int
}

// scanLead scans a lead row from a *sql.Rows or *sql.Row.
func scanLead(scan func(...any) error) (*models.Lead, error) {
	var l models.Lead
	var bitrixSentAt sql.NullTime
	err := scan(
		&l.ID, &l.Name, &l.Phone, &l.Email, &l.Comment,
		&l.Status, &l.BitrixResponse, &bitrixSentAt, &l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if bitrixSentAt.Valid {
		t := bitrixSentAt.Time
		l.BitrixSentAt = &t
	}
	return &l, nil
}

// CreateLead inserts a new lead and returns its ID.
func CreateLead(db *sql.DB, lead *models.Lead) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO leads (name, phone, email, comment, status, bitrix_response)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		lead.Name, lead.Phone, lead.Email, lead.Comment, lead.Status, lead.BitrixResponse,
	)
	if err != nil {
		return 0, fmt.Errorf("create lead: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get lead id: %w", err)
	}
	return id, nil
}

// GetLead returns a lead by its ID.
func GetLead(db *sql.DB, id int) (*models.Lead, error) {
	row := db.QueryRow(
		`SELECT id, name, phone, email, comment, status, bitrix_response, bitrix_sent_at, created_at
		 FROM leads WHERE id = ?`,
		id,
	)
	l, err := scanLead(row.Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get lead %d: %w", id, err)
	}
	return l, nil
}

// ListLeads returns filtered, paginated leads and the total matching count.
func ListLeads(db *sql.DB, f LeadsFilter) (*LeadsResult, error) {
	var conditions []string
	var args []any

	if f.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, f.Status)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		conditions = append(conditions, "(name LIKE ? OR phone LIKE ? OR email LIKE ?)")
		args = append(args, like, like, like)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total.
	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := db.QueryRow("SELECT COUNT(*) FROM leads"+where, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count leads: %w", err)
	}

	// Fetch page.
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	pageArgs := append(args, limit, offset)
	rows, err := db.Query(
		`SELECT id, name, phone, email, comment, status, bitrix_response, bitrix_sent_at, created_at
		 FROM leads`+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		pageArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close()

	leads := make([]models.Lead, 0)
	for rows.Next() {
		l, err := scanLead(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}
		leads = append(leads, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &LeadsResult{Leads: leads, Total: total}, nil
}

// UpdateLeadStatus updates only the status field of a lead.
func UpdateLeadStatus(db *sql.DB, id int, status string) error {
	_, err := db.Exec(`UPDATE leads SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		return fmt.Errorf("update lead status: %w", err)
	}
	return nil
}

// UpdateLeadBitrix updates the status, bitrix_response, and bitrix_sent_at fields.
func UpdateLeadBitrix(db *sql.DB, id int, status, response string, sentAt time.Time) error {
	_, err := db.Exec(
		`UPDATE leads SET status = ?, bitrix_response = ?, bitrix_sent_at = ? WHERE id = ?`,
		status, response, sentAt.UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("update lead bitrix: %w", err)
	}
	return nil
}

// DeleteLead removes a lead by ID.
func DeleteLead(db *sql.DB, id int) error {
	_, err := db.Exec(`DELETE FROM leads WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete lead: %w", err)
	}
	return nil
}

// CountByStatus returns the count of leads with the given status.
func CountByStatus(db *sql.DB, status string) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM leads WHERE status = ?`, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count by status: %w", err)
	}
	return count, nil
}

// CountCreatedToday returns the count of leads created today (UTC).
func CountCreatedToday(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM leads WHERE date(created_at) = date('now')`,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count today: %w", err)
	}
	return count, nil
}

// ListRecentLeads returns the N most recently created leads.
func ListRecentLeads(db *sql.DB, n int) ([]models.Lead, error) {
	rows, err := db.Query(
		`SELECT id, name, phone, email, comment, status, bitrix_response, bitrix_sent_at, created_at
		 FROM leads ORDER BY created_at DESC LIMIT ?`,
		n,
	)
	if err != nil {
		return nil, fmt.Errorf("list recent leads: %w", err)
	}
	defer rows.Close()

	leads := make([]models.Lead, 0, n)
	for rows.Next() {
		l, err := scanLead(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}
		leads = append(leads, *l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return leads, nil
}
