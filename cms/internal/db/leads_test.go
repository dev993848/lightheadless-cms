package db

import (
	"testing"
	"time"

	"github.com/lightcms/cms/internal/models"
)

func TestCreateAndGetLead(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	in := &models.Lead{
		Name:    "Иван",
		Phone:   "+79001234567",
		Email:   "ivan@example.com",
		Comment: "Тест",
		Status:  models.StatusNew,
	}
	id, err := CreateLead(d, in)
	if err != nil {
		t.Fatalf("CreateLead: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero ID")
	}

	got, err := GetLead(d, int(id))
	if err != nil {
		t.Fatalf("GetLead: %v", err)
	}
	if got == nil {
		t.Fatal("GetLead returned nil")
	}
	if got.Name != "Иван" {
		t.Errorf("Name mismatch: %q", got.Name)
	}
	if got.Phone != "+79001234567" {
		t.Errorf("Phone mismatch: %q", got.Phone)
	}
	if got.Status != models.StatusNew {
		t.Errorf("expected status 'new', got %q", got.Status)
	}
}

func TestGetLead_NotFound(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	got, err := GetLead(d, 9999)
	if err != nil {
		t.Fatalf("GetLead unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing lead")
	}
}

func TestListLeads_EmptyFilters(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = CreateLead(d, &models.Lead{Name: "A", Phone: "111", Status: models.StatusNew})
	_, _ = CreateLead(d, &models.Lead{Name: "B", Phone: "222", Status: models.StatusNew})

	res, err := ListLeads(d, LeadsFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListLeads: %v", err)
	}
	if res.Total != 2 {
		t.Errorf("expected total=2, got %d", res.Total)
	}
	if len(res.Leads) != 2 {
		t.Errorf("expected 2 leads, got %d", len(res.Leads))
	}
}

func TestListLeads_StatusFilter(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	id, _ := CreateLead(d, &models.Lead{Name: "A", Phone: "111", Status: models.StatusNew})
	_ = UpdateLeadStatus(d, int(id), models.StatusSent)
	_, _ = CreateLead(d, &models.Lead{Name: "B", Phone: "222", Status: models.StatusNew})

	res, err := ListLeads(d, LeadsFilter{Status: models.StatusSent, Limit: 10})
	if err != nil {
		t.Fatalf("ListLeads: %v", err)
	}
	if res.Total != 1 || len(res.Leads) != 1 {
		t.Errorf("expected 1 sent lead, got total=%d, len=%d", res.Total, len(res.Leads))
	}
}

func TestListLeads_SearchFilter(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = CreateLead(d, &models.Lead{Name: "Иван Петров", Phone: "111", Status: models.StatusNew})
	_, _ = CreateLead(d, &models.Lead{Name: "Мария Сидорова", Phone: "222", Status: models.StatusNew})

	res, err := ListLeads(d, LeadsFilter{Search: "Иван", Limit: 10})
	if err != nil {
		t.Fatalf("ListLeads: %v", err)
	}
	if res.Total != 1 || len(res.Leads) != 1 {
		t.Errorf("expected 1 search result, got total=%d", res.Total)
	}
}

func TestUpdateLeadStatus(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	id, _ := CreateLead(d, &models.Lead{Name: "Test", Phone: "123", Status: models.StatusNew})

	if err := UpdateLeadStatus(d, int(id), models.StatusSent); err != nil {
		t.Fatalf("UpdateLeadStatus: %v", err)
	}
	got, _ := GetLead(d, int(id))
	if got.Status != models.StatusSent {
		t.Errorf("expected status 'sent', got %q", got.Status)
	}
}

func TestUpdateLeadBitrix(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	id, _ := CreateLead(d, &models.Lead{Name: "Test", Phone: "123", Status: models.StatusNew})

	sentAt := time.Now().UTC().Truncate(time.Second)
	if err := UpdateLeadBitrix(d, int(id), models.StatusSent, "ok", sentAt); err != nil {
		t.Fatalf("UpdateLeadBitrix: %v", err)
	}
	got, _ := GetLead(d, int(id))
	if got.BitrixResponse != "ok" {
		t.Errorf("expected BitrixResponse 'ok', got %q", got.BitrixResponse)
	}
	if got.Status != models.StatusSent {
		t.Errorf("expected status 'sent', got %q", got.Status)
	}
}

func TestDeleteLead(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	id, _ := CreateLead(d, &models.Lead{Name: "Test", Phone: "123", Status: models.StatusNew})

	if err := DeleteLead(d, int(id)); err != nil {
		t.Fatalf("DeleteLead: %v", err)
	}
	got, err := GetLead(d, int(id))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for deleted lead")
	}
}

func TestCountByStatus(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = CreateLead(d, &models.Lead{Name: "A", Phone: "1", Status: models.StatusNew})
	_, _ = CreateLead(d, &models.Lead{Name: "B", Phone: "2", Status: models.StatusNew})
	id, _ := CreateLead(d, &models.Lead{Name: "C", Phone: "3", Status: models.StatusNew})
	_ = UpdateLeadStatus(d, int(id), models.StatusSent)

	newCount, err := CountByStatus(d, models.StatusNew)
	if err != nil {
		t.Fatalf("CountByStatus new: %v", err)
	}
	if newCount != 2 {
		t.Errorf("expected 2 new, got %d", newCount)
	}

	sentCount, err := CountByStatus(d, models.StatusSent)
	if err != nil {
		t.Fatalf("CountByStatus sent: %v", err)
	}
	if sentCount != 1 {
		t.Errorf("expected 1 sent, got %d", sentCount)
	}
}

func TestListRecentLeads(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	for i := 0; i < 5; i++ {
		_, _ = CreateLead(d, &models.Lead{Name: "Lead", Phone: "123", Status: models.StatusNew})
	}

	leads, err := ListRecentLeads(d, 3)
	if err != nil {
		t.Fatalf("ListRecentLeads: %v", err)
	}
	if len(leads) != 3 {
		t.Errorf("expected 3 recent leads, got %d", len(leads))
	}
}
