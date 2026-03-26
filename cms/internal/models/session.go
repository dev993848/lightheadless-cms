package models

import "time"

// Session represents an authenticated admin session.
type Session struct {
	ID        string
	AdminID   int
	CreatedAt time.Time
	ExpiresAt time.Time
}
