package db

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrate_CreatesAllTables(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	tables := []string{"leads", "news", "settings", "sessions"}
	for _, name := range tables {
		var cnt int
		err := db.QueryRow(
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name,
		).Scan(&cnt)
		if err != nil {
			t.Errorf("query table %q: %v", name, err)
		}
		if cnt != 1 {
			t.Errorf("table %q not found after Migrate", name)
		}
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db := openTestDB(t)
	for i := 0; i < 3; i++ {
		if err := Migrate(db); err != nil {
			t.Fatalf("Migrate run %d: %v", i+1, err)
		}
	}
}

func TestSeedAdmin_FirstRun(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	created, password, err := SeedAdmin(db)
	if err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	if !created {
		t.Error("expected created=true on first run")
	}
	if len(password) == 0 {
		t.Error("expected non-empty password")
	}
}

func TestSeedAdmin_SecondRun(t *testing.T) {
	db := openTestDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if _, _, err := SeedAdmin(db); err != nil {
		t.Fatalf("SeedAdmin first: %v", err)
	}

	created, password, err := SeedAdmin(db)
	if err != nil {
		t.Fatalf("SeedAdmin second: %v", err)
	}
	if created {
		t.Error("expected created=false on second run")
	}
	if password != "" {
		t.Errorf("expected empty password on second run, got %q", password)
	}
}

func TestGeneratePassword_Length(t *testing.T) {
	for _, n := range []int{6, 12, 24} {
		pw, err := generatePassword(n)
		if err != nil {
			t.Fatalf("generatePassword(%d): %v", n, err)
		}
		if len([]rune(pw)) != n {
			t.Errorf("generatePassword(%d) returned length %d", n, len([]rune(pw)))
		}
	}
}

func TestGeneratePassword_UsesCharset(t *testing.T) {
	charset := passwordCharset
	pw, err := generatePassword(100)
	if err != nil {
		t.Fatalf("generatePassword: %v", err)
	}
	for _, ch := range pw {
		if !containsRune(charset, ch) {
			t.Errorf("password contains invalid character: %q", ch)
		}
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
