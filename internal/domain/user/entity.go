package user

import (
	"context"
	"time"
)

// Status values for User.Status and Role.
type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
)

// Role is a named set of casbin permissions. Built-in roles (admin, cashier,
// …) are seeded with IsSystem true and cannot be deleted or renamed; custom
// roles are managed by the admin UI and persist their policies alongside the
// casbin rules via the application service.
type Role struct {
	Code        string       `json:"code"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"is_system"`
	Policies    []PolicyRule `json:"policies"`
}

// PolicyRule is one casbin permission: object (route pattern) + action
// (HTTP method). Enforced with keyMatch2, so "*" matches a single segment.
type PolicyRule struct {
	Object string `json:"object"`
	Action string `json:"action"`
}

// User is the authenticated principal. PasswordHash is argon2id and is never
// serialized to JSON. FailedAttempts/LockedUntil drive the login lockout
// policy (Điều 19). MustChangePassword forces a password reset on first login.
// PasswordChangedAt tracks when the password was last changed for expiry enforcement.
// MFAEnabled/MFASecret/BackupCodes support multi-factor authentication.
type User struct {
	ID                 string    `json:"id"`
	Username           string    `json:"username"`
	FullName           string    `json:"full_name"`
	PasswordHash       string    `json:"-"`
	RoleCodes          []string  `json:"role_codes"`
	Status             Status    `json:"status"`
	FailedAttempts     int       `json:"-"`
	LockedUntil        time.Time `json:"-"`
	MustChangePassword bool      `json:"must_change_password"`
	PasswordChangedAt  time.Time `json:"password_changed_at"`
	MFAEnabled         bool      `json:"mfa_enabled"`
	MFASecret          string    `json:"-"`
	BackupCodes        []BackupCode `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	CreatedBy          string    `json:"created_by"`
}

func (u *User) IsSuspended() bool { return u.Status == StatusSuspended }

func (u *User) IsLocked() bool {
	return u.Status != StatusSuspended && !u.LockedUntil.IsZero() && time.Now().Before(u.LockedUntil)
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*User, error)
	SaveRole(ctx context.Context, r *Role) error
	FindRoleByCode(ctx context.Context, code string) (*Role, error)
	ListRoles(ctx context.Context) ([]*Role, error)
	DeleteRole(ctx context.Context, code string) error
	// Password history for expiry enforcement (Circular 99/2025).
	AddPasswordHistory(ctx context.Context, userID, hash string) error
	GetPasswordHistory(ctx context.Context, userID string, limit int) ([]string, error)
	UpdatePasswordChangedAt(ctx context.Context, userID string, t time.Time) error
}
