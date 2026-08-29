package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"goGL/internal/domain/audit"
	"goGL/internal/domain/session"
	"goGL/internal/domain/user"
)

// Auditor records security-relevant events. Satisfied by the audit service.
type Auditor interface {
	Record(ctx context.Context, l *audit.AuditLog) error
}

// Policy is the account/session policy derived from configuration.
type Policy struct {
	CookieName           string
	MaxHours             int
	IdleMinutes          int
	MaxFailures          int
	LockoutMinutes       int
	MinPasswordLen       int
	PasswordExpiryDays   int // 0 = no expiry
	PasswordHistoryCount int // 0 = no history check
	MaxConcurrentSessions int // 0 = unlimited
}

type Service interface {
	// Login authenticates and issues a new session (a new cookie value).
	Login(ctx context.Context, username, password, ip, userAgent string) (*session.Session, error)
	// Validate resolves a session token to its user, enforcing expiry/idle
	// timeout and suspension, and touching LastSeen.
	Validate(ctx context.Context, token string) (*user.User, error)
	// Logout destroys a session.
	Logout(ctx context.Context, token string) error
	// ChangePassword verifies the current password and replaces it, revoking
	// all of the user's other sessions.
	ChangePassword(ctx context.Context, actorID, current, newPassword string) error
	// LogoutAll destroys all sessions for a user (force logout on all devices).
	LogoutAll(ctx context.Context, userID string) error
	// ListSessions returns all active sessions for a user.
	ListSessions(ctx context.Context, userID string) ([]*session.Session, error)
}

type service struct {
	users    user.Repository
	sessions session.Repository
	audit    Auditor
	policy   Policy
	now      func() time.Time
}

func NewService(users user.Repository, sessions session.Repository, audit Auditor, p Policy) Service {
	return &service{
		users:    users,
		sessions: sessions,
		audit:    audit,
		policy:   p,
		now:      time.Now,
	}
}

func (s *service) Login(ctx context.Context, username, password, ip, userAgent string) (*session.Session, error) {
	u, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		// Generic error — do not reveal whether the username exists.
		_ = s.audit.Record(ctx, &audit.AuditLog{
			UserCode: username, Module: "auth", Action: "login.failed", Timestamp: s.ts(),
		})
		return nil, user.ErrInvalidCredentials
	}

	if u.IsLocked() {
		_ = s.audit.Record(ctx, &audit.AuditLog{
			UserCode: u.Username, Module: "auth", Action: "login.locked", TargetID: u.ID, Timestamp: s.ts(),
		})
		return nil, user.ErrLockedOut
	}
	if u.IsSuspended() {
		_ = s.audit.Record(ctx, &audit.AuditLog{
			UserCode: u.Username, Module: "auth", Action: "login.suspended", TargetID: u.ID, Timestamp: s.ts(),
		})
		return nil, user.ErrSuspended
	}

	ok, err := user.VerifyPassword(u.PasswordHash, password)
	if err != nil {
		return nil, err
	}
	if !ok {
		u.FailedAttempts++
		if u.FailedAttempts >= s.policy.MaxFailures {
			u.LockedUntil = s.now().Add(time.Duration(s.policy.LockoutMinutes) * time.Minute)
		}
		u.UpdatedAt = s.now().UTC()
		_ = s.users.Update(ctx, u)
		_ = s.audit.Record(ctx, &audit.AuditLog{
			UserCode: u.Username, Module: "auth", Action: "login.failed", TargetID: u.ID, Timestamp: s.ts(),
		})
		return nil, user.ErrInvalidCredentials
	}

	u.FailedAttempts = 0
	u.LockedUntil = time.Time{}
	u.UpdatedAt = s.now().UTC()

	// Check password expiry (Circular 99/2025 compliance).
	if s.policy.PasswordExpiryDays > 0 && !u.PasswordChangedAt.IsZero() {
		expiresAt := u.PasswordChangedAt.Add(time.Duration(s.policy.PasswordExpiryDays) * 24 * time.Hour)
		if s.now().After(expiresAt) {
			u.MustChangePassword = true
		}
	}

	if err := s.users.Update(ctx, u); err != nil {
		return nil, err
	}

	ses := &session.Session{
		ID:        newToken(),
		UserID:    u.ID,
		Username:  u.Username,
		CreatedAt: s.ts(),
		LastSeen:  s.ts(),
		ExpiresAt: s.tsAddHours(s.policy.MaxHours),
		IP:        ip,
		UserAgent: userAgent,
	}

	// Enforce concurrent session limit.
	if s.policy.MaxConcurrentSessions > 0 {
		count, err := s.sessions.CountByUser(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		if count >= s.policy.MaxConcurrentSessions {
			// Delete oldest session
			existing, err := s.sessions.ListByUser(ctx, u.ID)
			if err != nil {
				return nil, err
			}
			if len(existing) > 0 {
				_ = s.sessions.Delete(ctx, existing[0].ID)
			}
		}
	}

	if err := s.sessions.Create(ctx, ses); err != nil {
		return nil, err
	}
	_ = s.audit.Record(ctx, &audit.AuditLog{
		UserCode: u.Username, Module: "auth", Action: "login.success", TargetID: u.ID, Timestamp: s.ts(),
	})
	return ses, nil
}

func (s *service) Validate(ctx context.Context, token string) (*user.User, error) {
	ses, err := s.sessions.FindByID(ctx, token)
	if err != nil {
		return nil, user.ErrInvalidSession
	}

	now := s.now()
	expires, err := time.Parse(time.RFC3339, ses.ExpiresAt)
	if err == nil && now.After(expires) {
		_ = s.sessions.Delete(ctx, token)
		return nil, user.ErrInvalidSession
	}
	lastSeen, err := time.Parse(time.RFC3339, ses.LastSeen)
	if err == nil && now.Sub(lastSeen) > time.Duration(s.policy.IdleMinutes)*time.Minute {
		_ = s.sessions.Delete(ctx, token)
		return nil, user.ErrInvalidSession
	}

	u, err := s.users.FindByID(ctx, ses.UserID)
	if err != nil {
		return nil, user.ErrInvalidSession
	}
	if u.IsSuspended() {
		_ = s.sessions.Delete(ctx, token)
		return nil, user.ErrSuspended
	}

	ses.LastSeen = s.ts()
	_ = s.sessions.Update(ctx, ses)
	return u, nil
}

func (s *service) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, token)
}

func (s *service) LogoutAll(ctx context.Context, userID string) error {
	return s.sessions.DeleteByUser(ctx, userID)
}

func (s *service) ListSessions(ctx context.Context, userID string) ([]*session.Session, error) {
	return s.sessions.ListByUser(ctx, userID)
}

func (s *service) ChangePassword(ctx context.Context, actorID, current, newPassword string) error {
	u, err := s.users.FindByID(ctx, actorID)
	if err != nil {
		return user.ErrInvalidSession
	}
	ok, err := user.VerifyPassword(u.PasswordHash, current)
	if err != nil {
		return err
	}
	if !ok {
		return user.ErrInvalidCredentials
	}
	if len(newPassword) < s.policy.MinPasswordLen {
		return user.ErrWeakPassword
	}
	if newPassword == current {
		return user.ErrWeakPassword
	}

	// Check password history (prevent reuse).
	if s.policy.PasswordHistoryCount > 0 {
		history, err := s.users.GetPasswordHistory(ctx, actorID, s.policy.PasswordHistoryCount)
		if err != nil {
			return err
		}
		for _, oldHash := range history {
			if match, _ := user.VerifyPassword(oldHash, newPassword); match {
				return user.ErrPasswordReused
			}
		}
	}

	hash, err := user.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Store old hash in history before updating.
	if s.policy.PasswordHistoryCount > 0 {
		_ = s.users.AddPasswordHistory(ctx, actorID, u.PasswordHash)
	}

	u.PasswordHash = hash
	u.MustChangePassword = false
	u.FailedAttempts = 0
	u.LockedUntil = time.Time{}
	u.PasswordChangedAt = s.now().UTC()
	u.UpdatedAt = s.now().UTC()
	if err := s.users.Update(ctx, u); err != nil {
		return err
	}
	if err := s.sessions.DeleteByUser(ctx, u.ID); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, &audit.AuditLog{
		UserCode: u.Username, Module: "auth", Action: "password.changed", TargetID: u.ID, Timestamp: s.ts(),
	})
	return nil
}

func (s *service) ts() string  { return s.now().UTC().Format(time.RFC3339) }
func (s *service) tsAddHours(h int) string {
	return s.now().Add(time.Duration(h) * time.Hour).UTC().Format(time.RFC3339)
}

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(errors.New("auth: failed to read random token"))
	}
	return hex.EncodeToString(b)
}
