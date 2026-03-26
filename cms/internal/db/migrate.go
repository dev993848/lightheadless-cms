package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

const schema = `
CREATE TABLE IF NOT EXISTS leads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    phone TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    comment TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'new',
    bitrix_response TEXT NOT NULL DEFAULT '',
    bitrix_sent_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_leads_status ON leads(status);
CREATE INDEX IF NOT EXISTS idx_leads_created ON leads(created_at DESC);

CREATE TABLE IF NOT EXISTS news (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date DATETIME NOT NULL,
    title TEXT NOT NULL,
    image TEXT NOT NULL DEFAULT '',
    announce TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_news_date ON news(date DESC);

CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY DEFAULT 1,
    site_name TEXT NOT NULL DEFAULT 'My CMS',
    admin_email TEXT NOT NULL DEFAULT '',
    admin_password TEXT NOT NULL DEFAULT '',
    bitrix24_webhook TEXT NOT NULL DEFAULT '',
    bitrix24_enabled INTEGER NOT NULL DEFAULT 0,
    CHECK(id = 1)
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    admin_id INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
`

// Migrate runs the schema migrations and cleans up expired sessions.
func Migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}

	// Clean up expired sessions.
	if _, err := db.Exec(`DELETE FROM sessions WHERE expires_at < datetime('now')`); err != nil {
		return fmt.Errorf("cleanup sessions: %w", err)
	}

	return nil
}

const passwordCharset = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"

// generatePassword creates a random password of length n from the charset.
func generatePassword(n int) (string, error) {
	charset := []rune(passwordCharset)
	result := make([]rune, n)
	for i := range result {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[idx.Int64()]
	}
	return string(result), nil
}

// SeedAdmin checks if the settings table is empty; if so, generates a random
// 12-character password, bcrypt-hashes it, and inserts the default settings row.
// Returns (true, plaintext password, nil) if seeded, (false, "", nil) otherwise.
func SeedAdmin(db *sql.DB) (created bool, password string, err error) {
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&count); err != nil {
		return false, "", fmt.Errorf("count settings: %w", err)
	}
	if count > 0 {
		return false, "", nil
	}

	password, err = generatePassword(12)
	if err != nil {
		return false, "", fmt.Errorf("generate password: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, "", fmt.Errorf("bcrypt hash: %w", err)
	}

	_, err = db.Exec(
		`INSERT INTO settings (id, site_name, admin_email, admin_password, bitrix24_webhook, bitrix24_enabled)
		 VALUES (1, 'My CMS', 'admin@example.com', ?, '', 0)`,
		string(hash),
	)
	if err != nil {
		return false, "", fmt.Errorf("insert settings: %w", err)
	}

	return true, password, nil
}
