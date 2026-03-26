package db

import (
	"testing"
)

func TestGenerateID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id, err := GenerateID()
		if err != nil {
			t.Fatalf("GenerateID: %v", err)
		}
		if len(id) != 64 {
			t.Errorf("expected 64-char hex ID, got len=%d", len(id))
		}
		if ids[id] {
			t.Errorf("duplicate session ID generated")
		}
		ids[id] = true
	}
}

func TestCreateAndGetSession(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	sessionID, err := CreateSession(d, 1)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sessionID == "" {
		t.Error("expected non-empty session ID")
	}

	session, err := GetSession(d, sessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if session == nil {
		t.Fatal("GetSession returned nil for valid session")
	}
	if session.ID != sessionID {
		t.Errorf("session ID mismatch: %q vs %q", session.ID, sessionID)
	}
	if session.AdminID != 1 {
		t.Errorf("expected admin_id=1, got %d", session.AdminID)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	session, err := GetSession(d, "nonexistent")
	if err != nil {
		t.Fatalf("GetSession unexpected error: %v", err)
	}
	if session != nil {
		t.Error("expected nil for missing session")
	}
}

func TestDeleteSession(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	sessionID, _ := CreateSession(d, 1)
	if err := DeleteSession(d, sessionID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	session, err := GetSession(d, sessionID)
	if err != nil {
		t.Fatalf("GetSession after delete: %v", err)
	}
	if session != nil {
		t.Error("expected nil after deleting session")
	}
}
