package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/lightcms/cms/internal/models"
)

// GenerateID returns a cryptographically random 32-byte hex string (64 characters).
func GenerateID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateSession inserts a new session for the given adminID, expiring in 24 hours.
// Returns the new session ID.
func CreateSession(db *sql.DB, adminID int) (string, error) {
	id, err := GenerateID()
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)

	_, err = db.Exec(
		`INSERT INTO sessions (id, admin_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		id, adminID, now, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return id, nil
}

// GetSession retrieves a valid (not expired) session by ID.
// Returns nil, nil if the session does not exist or has expired.
func GetSession(db *sql.DB, id string) (*models.Session, error) {
	row := db.QueryRow(
		`SELECT id, admin_id, created_at, expires_at FROM sessions WHERE id = ? AND expires_at > datetime('now')`,
		id,
	)

	var s models.Session
	err := row.Scan(&s.ID, &s.AdminID, &s.CreatedAt, &s.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	return &s, nil
}

// DeleteSession removes a session by ID.
func DeleteSession(db *sql.DB, id string) error {
	if _, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
