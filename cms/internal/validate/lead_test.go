package validate

import (
	"strings"
	"testing"
)

func TestLead_Valid(t *testing.T) {
	input := LeadInput{
		Name:    "  Иван Иванов  ",
		Phone:   "+7 (999) 123-45-67",
		Email:   "ivan@example.com",
		Comment: "Хочу заказать услугу",
	}
	out, errs := Lead(input)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if out.Name != "Иван Иванов" {
		t.Errorf("Name not trimmed: %q", out.Name)
	}
}

func TestLead_RequiredFields(t *testing.T) {
	_, errs := Lead(LeadInput{})
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors (name+phone), got %d: %v", len(errs), errs)
	}
	fields := map[string]bool{}
	for _, e := range errs {
		fields[e.Field] = true
	}
	if !fields["name"] {
		t.Error("expected error for field 'name'")
	}
	if !fields["phone"] {
		t.Error("expected error for field 'phone'")
	}
}

func TestLead_WhitespaceOnly(t *testing.T) {
	_, errs := Lead(LeadInput{Name: "   ", Phone: "\t"})
	if len(errs) != 2 {
		t.Errorf("expected 2 errors for whitespace-only fields, got %d", len(errs))
	}
}

func TestLead_EmailOptional(t *testing.T) {
	// Valid with no email.
	_, errs := Lead(LeadInput{Name: "Test", Phone: "123"})
	if len(errs) != 0 {
		t.Errorf("expected no errors when email is empty, got %v", errs)
	}
}

func TestLead_InvalidEmail(t *testing.T) {
	_, errs := Lead(LeadInput{Name: "Test", Phone: "123", Email: "notanemail"})
	if len(errs) != 1 || errs[0].Field != "email" {
		t.Errorf("expected email error, got %v", errs)
	}
}

func TestLead_ValidEmail(t *testing.T) {
	_, errs := Lead(LeadInput{Name: "Test", Phone: "123", Email: "test@domain.co.uk"})
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid email, got %v", errs)
	}
}

func TestLead_NameTooLong(t *testing.T) {
	longName := strings.Repeat("а", 256)
	_, errs := Lead(LeadInput{Name: longName, Phone: "123"})
	if len(errs) != 1 || errs[0].Field != "name" {
		t.Errorf("expected name length error, got %v", errs)
	}
}

func TestLead_PhoneTooLong(t *testing.T) {
	longPhone := strings.Repeat("1", 21)
	_, errs := Lead(LeadInput{Name: "Test", Phone: longPhone})
	if len(errs) != 1 || errs[0].Field != "phone" {
		t.Errorf("expected phone length error, got %v", errs)
	}
}

func TestLead_CommentTooLong(t *testing.T) {
	longComment := strings.Repeat("х", 1001)
	_, errs := Lead(LeadInput{Name: "Test", Phone: "123", Comment: longComment})
	if len(errs) != 1 || errs[0].Field != "comment" {
		t.Errorf("expected comment length error, got %v", errs)
	}
}

func TestLead_HTMLEscaping(t *testing.T) {
	out, errs := Lead(LeadInput{
		Name:    `<script>alert("xss")</script>`,
		Phone:   `+7&phone`,
		Comment: `<b>bold</b>`,
	})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if strings.Contains(out.Name, "<script>") {
		t.Errorf("Name not escaped: %q", out.Name)
	}
	if strings.Contains(out.Comment, "<b>") {
		t.Errorf("Comment not escaped: %q", out.Comment)
	}
}
