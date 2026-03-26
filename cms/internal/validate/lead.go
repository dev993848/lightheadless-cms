package validate

import (
	"html"
	"regexp"
	"strings"
)

// LeadInput holds the raw input for a lead form submission.
type LeadInput struct {
	Name    string
	Phone   string
	Email   string
	Comment string
}

// Error represents a field-level validation error.
type Error struct {
	Field   string
	Message string
}

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Lead validates and sanitizes the input. Returns the sanitized input and any errors.
// Rules:
//   - name:    required, max 255 chars, TrimSpace + html.EscapeString
//   - phone:   required, max 20 chars,  TrimSpace + html.EscapeString
//   - email:   optional; if given must match email regexp, max 255 chars, TrimSpace
//   - comment: optional, max 1000 chars, TrimSpace + html.EscapeString
func Lead(input LeadInput) (LeadInput, []Error) {
	var errs []Error

	// Name.
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		errs = append(errs, Error{Field: "name", Message: "Имя обязательно для заполнения"})
	} else if len([]rune(input.Name)) > 255 {
		errs = append(errs, Error{Field: "name", Message: "Имя не должно превышать 255 символов"})
	} else {
		input.Name = html.EscapeString(input.Name)
	}

	// Phone.
	input.Phone = strings.TrimSpace(input.Phone)
	if input.Phone == "" {
		errs = append(errs, Error{Field: "phone", Message: "Телефон обязателен для заполнения"})
	} else if len([]rune(input.Phone)) > 20 {
		errs = append(errs, Error{Field: "phone", Message: "Телефон не должен превышать 20 символов"})
	} else {
		input.Phone = html.EscapeString(input.Phone)
	}

	// Email (optional).
	input.Email = strings.TrimSpace(input.Email)
	if input.Email != "" {
		if len([]rune(input.Email)) > 255 {
			errs = append(errs, Error{Field: "email", Message: "Email не должен превышать 255 символов"})
		} else if !emailRegexp.MatchString(input.Email) {
			errs = append(errs, Error{Field: "email", Message: "Некорректный формат email"})
		}
	}

	// Comment (optional).
	input.Comment = strings.TrimSpace(input.Comment)
	if len([]rune(input.Comment)) > 1000 {
		errs = append(errs, Error{Field: "comment", Message: "Комментарий не должен превышать 1000 символов"})
	} else if input.Comment != "" {
		input.Comment = html.EscapeString(input.Comment)
	}

	return input, errs
}
