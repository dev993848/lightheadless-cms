package models

import "time"

// LeadStatus type alias for string.
type LeadStatus = string

const (
	StatusNew   LeadStatus = "new"
	StatusSent  LeadStatus = "sent"
	StatusError LeadStatus = "error"
)

// Lead represents a contact form submission.
type Lead struct {
	ID             int
	Name           string
	Phone          string
	Email          string
	Comment        string
	Status         LeadStatus
	BitrixResponse string
	BitrixSentAt   *time.Time
	CreatedAt      time.Time
}
