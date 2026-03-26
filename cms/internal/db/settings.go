package db

import (
	"database/sql"
	"fmt"

	"github.com/lightcms/cms/internal/models"
)

// Get returns the settings row (id=1). If not found, returns default values.
func Get(db *sql.DB) (*models.Settings, error) {
	row := db.QueryRow(
		`SELECT id, site_name, admin_email, admin_password, bitrix24_webhook, bitrix24_enabled
		 FROM settings WHERE id = 1`,
	)

	var s models.Settings
	var bitrixEnabled int
	err := row.Scan(
		&s.ID, &s.SiteName, &s.AdminEmail, &s.AdminPassword,
		&s.Bitrix24Webhook, &bitrixEnabled,
	)
	if err == sql.ErrNoRows {
		// Return defaults.
		return &models.Settings{
			ID:       1,
			SiteName: "My CMS",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	s.Bitrix24Enabled = bitrixEnabled != 0
	return &s, nil
}

// Upsert saves settings using INSERT OR REPLACE.
func Upsert(db *sql.DB, s *models.Settings) error {
	bitrixEnabled := 0
	if s.Bitrix24Enabled {
		bitrixEnabled = 1
	}

	_, err := db.Exec(
		`INSERT OR REPLACE INTO settings (id, site_name, admin_email, admin_password, bitrix24_webhook, bitrix24_enabled)
		 VALUES (1, ?, ?, ?, ?, ?)`,
		s.SiteName, s.AdminEmail, s.AdminPassword, s.Bitrix24Webhook, bitrixEnabled,
	)
	if err != nil {
		return fmt.Errorf("upsert settings: %w", err)
	}
	return nil
}
