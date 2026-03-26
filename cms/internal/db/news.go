package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lightcms/cms/internal/models"
)

// NewsFilter defines filtering options for listing news.
type NewsFilter struct {
	Search string
	Limit  int
	Offset int
}

// NewsResult holds a page of news items and the total count matching the filter.
type NewsResult struct {
	News  []models.News
	Total int
}

// scanNews scans a news row.
func scanNews(scan func(...any) error) (*models.News, error) {
	var n models.News
	err := scan(
		&n.ID, &n.Date, &n.Title, &n.Image, &n.Announce,
		&n.Description, &n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// CreateNews inserts a new news item and returns its ID.
func CreateNews(db *sql.DB, n *models.News) (int64, error) {
	log.Printf("Creating news: title=%q, date=%v", n.Title, n.Date)
	res, err := db.Exec(
		`INSERT INTO news (date, title, image, announce, description)
		 VALUES (?, ?, ?, ?, ?)`,
		n.Date.UTC(), n.Title, n.Image, n.Announce, n.Description,
	)
	if err != nil {
		return 0, fmt.Errorf("create news: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get news id: %w", err)
	}
	log.Printf("News created with ID=%d", id)
	return id, nil
}

// GetNews returns a news item by its ID.
func GetNews(db *sql.DB, id int) (*models.News, error) {
	row := db.QueryRow(
		`SELECT id, date, title, image, announce, description, created_at, updated_at
		 FROM news WHERE id = ?`,
		id,
	)
	n, err := scanNews(row.Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get news %d: %w", id, err)
	}
	return n, nil
}

// ListNews returns filtered, paginated news and the total matching count.
func ListNews(db *sql.DB, f NewsFilter) (*NewsResult, error) {
	var conditions []string
	var args []any

	if f.Search != "" {
		like := "%" + f.Search + "%"
		conditions = append(conditions, "(title LIKE ? OR announce LIKE ?)")
		args = append(args, like, like)
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total.
	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := db.QueryRow("SELECT COUNT(*) FROM news"+where, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count news: %w", err)
	}

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
		`SELECT id, date, title, image, announce, description, created_at, updated_at
		 FROM news`+where+` ORDER BY date DESC LIMIT ? OFFSET ?`,
		pageArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("list news: %w", err)
	}
	defer rows.Close()

	newsList := make([]models.News, 0)
	for rows.Next() {
		n, err := scanNews(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan news: %w", err)
		}
		newsList = append(newsList, *n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &NewsResult{News: newsList, Total: total}, nil
}

// UpdateNews updates all fields of a news item and sets updated_at to now.
func UpdateNews(db *sql.DB, n *models.News) error {
	_, err := db.Exec(
		`UPDATE news SET date = ?, title = ?, image = ?, announce = ?, description = ?, updated_at = ?
		 WHERE id = ?`,
		n.Date.UTC(), n.Title, n.Image, n.Announce, n.Description, time.Now().UTC(), n.ID,
	)
	if err != nil {
		return fmt.Errorf("update news: %w", err)
	}
	return nil
}

// DeleteNews removes a news item by ID and returns the deleted item (for image cleanup).
func DeleteNews(db *sql.DB, id int) (*models.News, error) {
	// Fetch first so we can return info for cleanup.
	n, err := GetNews(db, id)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}

	if _, err := db.Exec(`DELETE FROM news WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("delete news: %w", err)
	}
	return n, nil
}

// CountNews returns the total number of news items.
func CountNews(db *sql.DB) (int, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM news`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count news: %w", err)
	}
	return count, nil
}
