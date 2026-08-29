package options

import (
	"context"
	"testing"

	"goGL/internal/domain/audit"
	"goGL/internal/domain/options"
)

// mockOptionRepo implements options.Repository for testing.
type mockOptionRepo struct {
	opts map[string]*options.Option
}

func newMockOptionRepo() *mockOptionRepo {
	return &mockOptionRepo{opts: make(map[string]*options.Option)}
}

func (m *mockOptionRepo) Create(_ context.Context, o *options.Option) error {
	m.opts[o.Key] = o
	return nil
}
func (m *mockOptionRepo) FindByKey(_ context.Context, key string) (*options.Option, error) {
	o, ok := m.opts[key]
	if !ok {
		return nil, options.ErrNotFound
	}
	return o, nil
}
func (m *mockOptionRepo) Update(_ context.Context, o *options.Option) error {
	m.opts[o.Key] = o
	return nil
}
func (m *mockOptionRepo) List(_ context.Context) ([]*options.Option, error) {
	var out []*options.Option
	for _, o := range m.opts {
		out = append(out, o)
	}
	return out, nil
}

// mockAuditor implements Auditor for testing.
type mockOptionAuditor struct {
	logs []*audit.AuditLog
}

func (m *mockOptionAuditor) Record(_ context.Context, l *audit.AuditLog) error {
	m.logs = append(m.logs, l)
	return nil
}

func TestValidateTaxCode_Valid10Digits(t *testing.T) {
	err := ValidateOptionValue("company.tax_code", "0123456789", options.TypeString)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateTaxCode_Valid13Digits(t *testing.T) {
	err := ValidateOptionValue("company.tax_code", "0123456789012", options.TypeString)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateTaxCode_Invalid(t *testing.T) {
	err := ValidateOptionValue("company.tax_code", "123", options.TypeString)
	if err == nil {
		t.Fatal("expected error for invalid tax code")
	}
}

func TestValidateEmail_Valid(t *testing.T) {
	err := ValidateOptionValue("company.email", "test@example.com", options.TypeString)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateEmail_Invalid(t *testing.T) {
	err := ValidateOptionValue("company.email", "not-an-email", options.TypeString)
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestValidatePhoneNumber_Valid(t *testing.T) {
	err := ValidateOptionValue("company.phone", "0912345678", options.TypeString)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidatePhoneNumber_Invalid(t *testing.T) {
	err := ValidateOptionValue("company.phone", "abc", options.TypeString)
	if err == nil {
		t.Fatal("expected error for invalid phone")
	}
}

func TestValidatePasswordExpiryDays_Valid(t *testing.T) {
	err := ValidateOptionValue("password.expiry_days", "90", options.TypeInt)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidatePasswordExpiryDays_TooShort(t *testing.T) {
	err := ValidateOptionValue("password.expiry_days", "5", options.TypeInt)
	if err == nil {
		t.Fatal("expected error for password expiry too short")
	}
}

func TestValidateFiscalYearStart_Valid(t *testing.T) {
	err := ValidateOptionValue("company.fiscal_year_start", "01-01", options.TypeString)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateFiscalYearStart_Invalid(t *testing.T) {
	err := ValidateOptionValue("company.fiscal_year_start", "13-01", options.TypeString)
	if err == nil {
		t.Fatal("expected error for invalid fiscal year start")
	}
}

func TestBulkUpdate(t *testing.T) {
	repo := newMockOptionRepo()
	auditor := &mockOptionAuditor{}
	svc := NewService(repo, auditor)

	// Create initial options
	for _, opt := range []options.Option{
		{Key: "company.name", Type: options.TypeString, Value: "Old Name"},
		{Key: "company.email", Type: options.TypeString, Value: "old@example.com"},
	} {
		_ = svc.SetOption(context.Background(), "admin", &opt)
	}

	// Bulk update
	updates := map[string]string{
		"company.name":  "New Name",
		"company.email": "new@example.com",
	}
	err := svc.BulkUpdate(context.Background(), "admin", updates)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify
	name, _ := svc.GetOption(context.Background(), "company.name")
	if name.Value != "New Name" {
		t.Errorf("expected 'New Name', got '%s'", name.Value)
	}
	email, _ := svc.GetOption(context.Background(), "company.email")
	if email.Value != "new@example.com" {
		t.Errorf("expected 'new@example.com', got '%s'", email.Value)
	}
}

func TestGetOptionsByCategory(t *testing.T) {
	repo := newMockOptionRepo()
	auditor := &mockOptionAuditor{}
	svc := NewService(repo, auditor)

	// Create options in different categories
	for _, opt := range []options.Option{
		{Key: "company.name", Category: "company", Type: options.TypeString, Value: "Test"},
		{Key: "company.email", Category: "company", Type: options.TypeString, Value: "test@example.com"},
		{Key: "tax.vat_rate", Category: "tax", Type: options.TypeInt, Value: "10"},
	} {
		_ = svc.SetOption(context.Background(), "admin", &opt)
	}

	// Get company options
	opts, err := svc.ListOptionsByCategory(context.Background(), "company")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
}
