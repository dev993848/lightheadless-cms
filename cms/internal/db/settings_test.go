package db

import (
	"testing"

	"github.com/lightcms/cms/internal/models"
)

func TestGetSettings_NoRow(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// No settings row yet — should return defaults.
	s, err := Get(d)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s == nil {
		t.Fatal("Get returned nil")
	}
	if s.SiteName != "My CMS" {
		t.Errorf("expected default SiteName='My CMS', got %q", s.SiteName)
	}
}

func TestUpsertAndGetSettings(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	settings := &models.Settings{
		SiteName:        "Test CMS",
		AdminEmail:      "admin@test.com",
		AdminPassword:   "hashed_password",
		Bitrix24Webhook: "https://example.com/hook",
		Bitrix24Enabled: true,
	}
	if err := Upsert(d, settings); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := Get(d)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SiteName != "Test CMS" {
		t.Errorf("SiteName mismatch: %q", got.SiteName)
	}
	if got.AdminEmail != "admin@test.com" {
		t.Errorf("AdminEmail mismatch: %q", got.AdminEmail)
	}
	if !got.Bitrix24Enabled {
		t.Error("expected Bitrix24Enabled=true")
	}
	if got.Bitrix24Webhook != "https://example.com/hook" {
		t.Errorf("Bitrix24Webhook mismatch: %q", got.Bitrix24Webhook)
	}
}

func TestUpsert_Update(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Insert first.
	if err := Upsert(d, &models.Settings{SiteName: "Original"}); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	// Update.
	if err := Upsert(d, &models.Settings{SiteName: "Updated", AdminEmail: "new@example.com"}); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	got, _ := Get(d)
	if got.SiteName != "Updated" {
		t.Errorf("expected Updated, got %q", got.SiteName)
	}
	if got.AdminEmail != "new@example.com" {
		t.Errorf("expected new email, got %q", got.AdminEmail)
	}
}

func TestBitrix24Enabled_False(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if err := Upsert(d, &models.Settings{Bitrix24Enabled: false}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, _ := Get(d)
	if got.Bitrix24Enabled {
		t.Error("expected Bitrix24Enabled=false")
	}
}
