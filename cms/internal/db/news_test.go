package db

import (
	"testing"
	"time"

	"github.com/lightcms/cms/internal/models"
)

func newNewsItem(title string) *models.News {
	return &models.News{
		Date:        time.Now().UTC(),
		Title:       title,
		Image:       "",
		Announce:    "Краткое описание " + title,
		Description: "Полное описание " + title,
	}
}

func TestCreateAndGetNews(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	id, err := CreateNews(d, newNewsItem("Тест новости"))
	if err != nil {
		t.Fatalf("CreateNews: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero ID")
	}

	got, err := GetNews(d, int(id))
	if err != nil {
		t.Fatalf("GetNews: %v", err)
	}
	if got == nil {
		t.Fatal("GetNews returned nil")
	}
	if got.Title != "Тест новости" {
		t.Errorf("Title mismatch: %q", got.Title)
	}
}

func TestGetNews_NotFound(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	got, err := GetNews(d, 9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing news")
	}
}

func TestListNews_All(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = CreateNews(d, newNewsItem("Новость 1"))
	_, _ = CreateNews(d, newNewsItem("Новость 2"))
	_, _ = CreateNews(d, newNewsItem("Новость 3"))

	res, err := ListNews(d, NewsFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListNews: %v", err)
	}
	if res.Total != 3 {
		t.Errorf("expected total=3, got %d", res.Total)
	}
}

func TestListNews_Pagination(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	for i := 0; i < 5; i++ {
		_, _ = CreateNews(d, newNewsItem("Новость"))
	}

	res, err := ListNews(d, NewsFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("ListNews: %v", err)
	}
	if res.Total != 5 {
		t.Errorf("expected total=5, got %d", res.Total)
	}
	if len(res.News) != 2 {
		t.Errorf("expected 2 news on page, got %d", len(res.News))
	}
}

func TestUpdateNews(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	id, _ := CreateNews(d, newNewsItem("Оригинальный заголовок"))

	item, _ := GetNews(d, int(id))
	item.Title = "Обновлённый заголовок"
	if err := UpdateNews(d, item); err != nil {
		t.Fatalf("UpdateNews: %v", err)
	}

	got, _ := GetNews(d, int(id))
	if got.Title != "Обновлённый заголовок" {
		t.Errorf("expected updated title, got %q", got.Title)
	}
}

func TestDeleteNews(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	id, _ := CreateNews(d, newNewsItem("К удалению"))

	deleted, err := DeleteNews(d, int(id))
	if err != nil {
		t.Fatalf("DeleteNews: %v", err)
	}
	if deleted == nil {
		t.Error("expected deleted news item to be returned")
	}
	if deleted.Title != "К удалению" {
		t.Errorf("unexpected deleted title: %q", deleted.Title)
	}

	got, _ := GetNews(d, int(id))
	if got != nil {
		t.Error("expected nil for deleted news")
	}
}

func TestDeleteNews_NotFound(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	deleted, err := DeleteNews(d, 9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleted != nil {
		t.Error("expected nil for missing news")
	}
}

func TestCountNews(t *testing.T) {
	d := openTestDB(t)
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, _ = CreateNews(d, newNewsItem("A"))
	_, _ = CreateNews(d, newNewsItem("B"))

	count, err := CountNews(d)
	if err != nil {
		t.Fatalf("CountNews: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}
