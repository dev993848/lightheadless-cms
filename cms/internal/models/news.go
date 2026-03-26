package models

import "time"

// News represents a news article.
type News struct {
	ID          int
	Date        time.Time
	Title       string
	Image       string
	Announce    string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DateForInput returns the date in YYYY-MM-DD format for HTML date input.
func (n News) DateForInput() string {
	if n.Date.IsZero() {
		return ""
	}
	return n.Date.Format("2006-01-02")
}
