package options

import (
	"context"
	"strconv"
	"strings"
	"time"

	"goGL/internal/domain/audit"
	"goGL/internal/domain/options"
)

// Auditor records setting changes. Satisfied by the audit service.
type Auditor interface {
	Record(ctx context.Context, l *audit.AuditLog) error
}

type Service interface {
	// SetOption validates and persists a setting. Value is validated against
	// the option's declared type; an int/bool mismatch is rejected.
	SetOption(ctx context.Context, by string, o *options.Option) error
	// GetOption returns the stored value, or the default when unset.
	GetOption(ctx context.Context, key string) (*options.Option, error)
	// ListOptions returns all options, stored value or default.
	ListOptions(ctx context.Context) ([]*options.Option, error)
	// ResetOption restores the default value.
	ResetOption(ctx context.Context, by, key string) error
	// BulkUpdate updates multiple options at once.
	BulkUpdate(ctx context.Context, by string, updates map[string]string) error
	// ListOptionsByCategory returns options filtered by category.
	ListOptionsByCategory(ctx context.Context, category string) ([]*options.Option, error)
}

type service struct {
	repo  options.Repository
	audit Auditor
	now   func() time.Time
}

func NewService(repo options.Repository, auditor Auditor) Service {
	return &service{repo: repo, audit: auditor, now: time.Now}
}

func (s *service) SetOption(ctx context.Context, by string, o *options.Option) error {
	key := strings.TrimSpace(o.Key)
	if key == "" {
		return options.ErrInvalidValue
	}
	if o.Type == "" {
		o.Type = options.TypeString
	}
	if err := validateValue(o.Type, o.Value); err != nil {
		return err
	}

	now := s.now().UTC()
	existing, err := s.repo.FindByKey(ctx, key)
	switch {
	case err == nil:
		existing.Value = o.Value
		existing.Type = o.Type
		existing.UpdatedAt = now
		existing.UpdatedBy = by
		if err := s.repo.Update(ctx, existing); err != nil {
			return err
		}
	case err == options.ErrNotFound:
		o.ID = "opt_" + key
		o.UpdatedAt = now
		o.UpdatedBy = by
		if err := s.repo.Create(ctx, o); err != nil {
			return err
		}
	default:
		return err
	}

	return s.audit.Record(ctx, &audit.AuditLog{
		UserCode: by, Module: "options", Action: "option.set", TargetID: key,
		Timestamp: now.Format(time.RFC3339),
	})
}

func (s *service) GetOption(ctx context.Context, key string) (*options.Option, error) {
	o, err := s.repo.FindByKey(ctx, key)
	if err == options.ErrNotFound {
		// Unknown key: no default known, so return a NotFound-style error.
		return nil, options.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if o.Value == "" {
		o.Value = o.DefaultValue
	}
	return o, nil
}

func (s *service) ListOptions(ctx context.Context) ([]*options.Option, error) {
	opts, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, o := range opts {
		if o.Value == "" {
			o.Value = o.DefaultValue
		}
	}
	return opts, nil
}

func (s *service) ResetOption(ctx context.Context, by, key string) error {
	o, err := s.repo.FindByKey(ctx, key)
	if err != nil {
		return err
	}
	o.Value = ""
	o.UpdatedAt = s.now().UTC()
	o.UpdatedBy = by
	if err := s.repo.Update(ctx, o); err != nil {
		return err
	}
	return s.audit.Record(ctx, &audit.AuditLog{
		UserCode: by, Module: "options", Action: "option.reset", TargetID: key,
		Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) BulkUpdate(ctx context.Context, by string, updates map[string]string) error {
	for key, value := range updates {
		existing, err := s.repo.FindByKey(ctx, key)
		if err != nil {
			if err == options.ErrNotFound {
				continue
			}
			return err
		}
		existing.Value = value
		existing.UpdatedAt = s.now().UTC()
		existing.UpdatedBy = by
		if err := s.repo.Update(ctx, existing); err != nil {
			return err
		}
	}
	return s.audit.Record(ctx, &audit.AuditLog{
		UserCode: by, Module: "options", Action: "option.bulk_update", TargetID: "multiple",
		Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) ListOptionsByCategory(ctx context.Context, category string) ([]*options.Option, error) {
	opts, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	var result []*options.Option
	for _, o := range opts {
		if o.Category == category {
			if o.Value == "" {
				o.Value = o.DefaultValue
			}
			result = append(result, o)
		}
	}
	return result, nil
}

// validateValue checks a value against its declared type.
func validateValue(t options.ValueType, v string) error {
	switch t {
	case options.TypeInt:
		if _, err := strconv.ParseInt(v, 10, 64); err != nil {
			return options.ErrInvalidValue
		}
	case options.TypeBool:
		if v != "true" && v != "false" {
			return options.ErrInvalidValue
		}
	case options.TypeString:
		// Anything is a valid string.
	default:
		return options.ErrInvalidValue
	}
	return nil
}

// ValidateOptionValue validates a value for a specific option key.
func ValidateOptionValue(key, value string, valueType options.ValueType) error {
	// First validate the type
	if err := validateValue(valueType, value); err != nil {
		return err
	}

	// Then validate key-specific rules
	switch key {
	case "company.tax_code":
		return validateTaxCode(value)
	case "company.email":
		return validateEmail(value)
	case "company.phone":
		return validatePhone(value)
	case "password.expiry_days":
		return validatePasswordExpiry(value)
	case "company.fiscal_year_start":
		return validateFiscalYearStart(value)
	}
	return nil
}

func validateTaxCode(code string) error {
	if len(code) != 10 && len(code) != 13 {
		return options.ErrInvalidValue
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return options.ErrInvalidValue
		}
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return nil
	}
	at := false
	for _, c := range email {
		if c == '@' {
			at = true
		}
	}
	if !at {
		return options.ErrInvalidValue
	}
	return nil
}

func validatePhone(phone string) error {
	if phone == "" {
		return nil
	}
	for _, c := range phone {
		if c < '0' || c > '9' {
			return options.ErrInvalidValue
		}
	}
	return nil
}

func validatePasswordExpiry(days string) error {
	n, err := strconv.ParseInt(days, 10, 64)
	if err != nil {
		return options.ErrInvalidValue
	}
	if n < 30 {
		return options.ErrInvalidValue
	}
	return nil
}

func validateFiscalYearStart(date string) error {
	if len(date) != 5 || date[2] != '-' {
		return options.ErrInvalidValue
	}
	month, err := strconv.ParseInt(date[:2], 10, 64)
	if err != nil || month < 1 || month > 12 {
		return options.ErrInvalidValue
	}
	day, err := strconv.ParseInt(date[3:], 10, 64)
	if err != nil || day < 1 || day > 31 {
		return options.ErrInvalidValue
	}
	return nil
}
