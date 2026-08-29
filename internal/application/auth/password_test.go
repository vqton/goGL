package auth

import (
	"context"
	"testing"
	"time"

	"goGL/internal/domain/audit"
	"goGL/internal/domain/session"
	"goGL/internal/domain/user"
)

// mockUserRepo implements user.Repository for testing.
type mockUserRepo struct {
	users    map[string]*user.User
	byUser   map[string]*user.User
	roles    map[string]*user.Role
	history  []string // password hashes in order
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:  make(map[string]*user.User),
		byUser: make(map[string]*user.User),
		roles:  make(map[string]*user.Role),
	}
}

func (m *mockUserRepo) Create(_ context.Context, u *user.User) error {
	m.users[u.ID] = u
	m.byUser[u.Username] = u
	return nil
}
func (m *mockUserRepo) FindByID(_ context.Context, id string) (*user.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}
func (m *mockUserRepo) FindByUsername(_ context.Context, username string) (*user.User, error) {
	u, ok := m.byUser[username]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}
func (m *mockUserRepo) Update(_ context.Context, u *user.User) error {
	m.users[u.ID] = u
	m.byUser[u.Username] = u
	return nil
}
func (m *mockUserRepo) Delete(_ context.Context, id string) error {
	delete(m.users, id)
	return nil
}
func (m *mockUserRepo) List(_ context.Context) ([]*user.User, error) {
	var out []*user.User
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, nil
}
func (m *mockUserRepo) SaveRole(_ context.Context, r *user.Role) error {
	m.roles[r.Code] = r
	return nil
}
func (m *mockUserRepo) FindRoleByCode(_ context.Context, code string) (*user.Role, error) {
	r, ok := m.roles[code]
	if !ok {
		return nil, user.ErrRoleNotFound
	}
	return r, nil
}
func (m *mockUserRepo) ListRoles(_ context.Context) ([]*user.Role, error) {
	var out []*user.Role
	for _, r := range m.roles {
		out = append(out, r)
	}
	return out, nil
}
func (m *mockUserRepo) DeleteRole(_ context.Context, code string) error {
	delete(m.roles, code)
	return nil
}

// Password history methods
func (m *mockUserRepo) AddPasswordHistory(_ context.Context, userID, hash string) error {
	m.history = append(m.history, hash)
	return nil
}
func (m *mockUserRepo) GetPasswordHistory(_ context.Context, userID string, limit int) ([]string, error) {
	if limit > len(m.history) {
		limit = len(m.history)
	}
	start := len(m.history) - limit
	if start < 0 {
		start = 0
	}
	return m.history[start:], nil
}
func (m *mockUserRepo) UpdatePasswordChangedAt(_ context.Context, userID string, t time.Time) error {
	u, ok := m.users[userID]
	if ok {
		u.PasswordChangedAt = t
	}
	return nil
}

// mockSessionRepo implements session.Repository for testing.
type mockSessionRepo struct {
	sessions map[string]*session.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{sessions: make(map[string]*session.Session)}
}

func (m *mockSessionRepo) Create(_ context.Context, s *session.Session) error {
	m.sessions[s.ID] = s
	return nil
}
func (m *mockSessionRepo) FindByID(_ context.Context, id string) (*session.Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	return s, nil
}
func (m *mockSessionRepo) Update(_ context.Context, s *session.Session) error {
	m.sessions[s.ID] = s
	return nil
}
func (m *mockSessionRepo) Delete(_ context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}
func (m *mockSessionRepo) DeleteByUser(_ context.Context, userID string) error {
	for id, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, id)
		}
	}
	return nil
}
func (m *mockSessionRepo) CountActive(_ context.Context) (int, error) {
	return len(m.sessions), nil
}
func (m *mockSessionRepo) CountByUser(_ context.Context, userID string) (int, error) {
	count := 0
	for _, s := range m.sessions {
		if s.UserID == userID {
			count++
		}
	}
	return count, nil
}
func (m *mockSessionRepo) ListByUser(_ context.Context, userID string) ([]*session.Session, error) {
	var out []*session.Session
	for _, s := range m.sessions {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

// mockAuditor implements Auditor for testing.
type mockAuditor struct {
	logs []*audit.AuditLog
}

func (m *mockAuditor) Record(_ context.Context, l *audit.AuditLog) error {
	m.logs = append(m.logs, l)
	return nil
}

func TestLogin_PasswordExpired_ForceChange(t *testing.T) {
	// Arrange
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	auditor := &mockAuditor{}

	hash, _ := user.HashPassword("currentPass123")
	u := &user.User{
		ID:                 "u_testuser",
		Username:           "testuser",
		FullName:           "Test User",
		PasswordHash:       hash,
		RoleCodes:          []string{"accountant"},
		Status:             user.StatusActive,
		MustChangePassword: false,
		PasswordChangedAt:  time.Now().Add(-100 * 24 * time.Hour), // 100 days ago
	}
	_ = userRepo.Create(context.Background(), u)

	svc := NewService(userRepo, sessionRepo, auditor, Policy{
		CookieName:        "session",
		MaxHours:          24,
		IdleMinutes:       30,
		MaxFailures:       5,
		LockoutMinutes:    15,
		MinPasswordLen:    8,
		PasswordExpiryDays: 90, // expires after 90 days
	})

	// Act
	ses, err := svc.Login(context.Background(), "testuser", "currentPass123", "127.0.0.1", "test-agent")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ses == nil {
		t.Fatal("expected session, got nil")
	}
	// After login with expired password, MustChangePassword should be true
	updated, _ := userRepo.FindByID(context.Background(), "u_testuser")
	if !updated.MustChangePassword {
		t.Error("expected MustChangePassword to be true after login with expired password")
	}
}

func TestLogin_PasswordNotExpired_NormalLogin(t *testing.T) {
	// Arrange
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	auditor := &mockAuditor{}

	hash, _ := user.HashPassword("currentPass123")
	u := &user.User{
		ID:                 "u_testuser",
		Username:           "testuser",
		FullName:           "Test User",
		PasswordHash:       hash,
		RoleCodes:          []string{"accountant"},
		Status:             user.StatusActive,
		MustChangePassword: false,
		PasswordChangedAt:  time.Now(), // just changed
	}
	_ = userRepo.Create(context.Background(), u)

	svc := NewService(userRepo, sessionRepo, auditor, Policy{
		CookieName:        "session",
		MaxHours:          24,
		IdleMinutes:       30,
		MaxFailures:       5,
		LockoutMinutes:    15,
		MinPasswordLen:    8,
		PasswordExpiryDays: 90,
	})

	// Act
	ses, err := svc.Login(context.Background(), "testuser", "currentPass123", "127.0.0.1", "test-agent")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ses == nil {
		t.Fatal("expected session, got nil")
	}
	// MustChangePassword should remain false
	updated, _ := userRepo.FindByID(context.Background(), "u_testuser")
	if updated.MustChangePassword {
		t.Error("expected MustChangePassword to remain false when password is not expired")
	}
}

func TestChangePassword_StoresHistory(t *testing.T) {
	// Arrange
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	auditor := &mockAuditor{}

	hash, _ := user.HashPassword("oldPass123")
	u := &user.User{
		ID:                 "u_testuser",
		Username:           "testuser",
		FullName:           "Test User",
		PasswordHash:       hash,
		RoleCodes:          []string{"accountant"},
		Status:             user.StatusActive,
	}
	_ = userRepo.Create(context.Background(), u)

	svc := NewService(userRepo, sessionRepo, auditor, Policy{
		CookieName:        "session",
		MaxHours:          24,
		IdleMinutes:       30,
		MaxFailures:       5,
		LockoutMinutes:    15,
		MinPasswordLen:    8,
		PasswordExpiryDays: 90,
		PasswordHistoryCount: 5,
	})

	// Act - change password
	err := svc.ChangePassword(context.Background(), "u_testuser", "oldPass123", "newPass123")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Check that old hash was stored in history
	history, err := userRepo.GetPasswordHistory(context.Background(), "u_testuser", 5)
	if err != nil {
		t.Fatalf("expected no error getting history, got: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
	if history[0] != hash {
		t.Error("expected old hash to be stored in history")
	}
}

func TestChangePassword_ReusesOldPassword_Rejected(t *testing.T) {
	// Arrange
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	auditor := &mockAuditor{}

	hash, _ := user.HashPassword("oldPass123")
	u := &user.User{
		ID:                 "u_testuser",
		Username:           "testuser",
		FullName:           "Test User",
		PasswordHash:       hash,
		RoleCodes:          []string{"accountant"},
		Status:             user.StatusActive,
	}
	_ = userRepo.Create(context.Background(), u)

	svc := NewService(userRepo, sessionRepo, auditor, Policy{
		CookieName:        "session",
		MaxHours:          24,
		IdleMinutes:       30,
		MaxFailures:       5,
		LockoutMinutes:    15,
		MinPasswordLen:    8,
		PasswordExpiryDays: 90,
		PasswordHistoryCount: 5,
	})

	// First change - should succeed
	err := svc.ChangePassword(context.Background(), "u_testuser", "oldPass123", "newPass123")
	if err != nil {
		t.Fatalf("first change should succeed, got: %v", err)
	}

	// Second change - try to reuse old password
	err = svc.ChangePassword(context.Background(), "u_testuser", "newPass123", "oldPass123")
	if err == nil {
		t.Fatal("expected error when reusing old password, got nil")
	}
	if err != user.ErrPasswordReused {
		t.Fatalf("expected ErrPasswordReused, got: %v", err)
	}
}

func TestChangePassword_EnforceExpiry(t *testing.T) {
	// Arrange
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	auditor := &mockAuditor{}

	hash, _ := user.HashPassword("currentPass123")
	u := &user.User{
		ID:                 "u_testuser",
		Username:           "testuser",
		FullName:           "Test User",
		PasswordHash:       hash,
		RoleCodes:          []string{"accountant"},
		Status:             user.StatusActive,
		PasswordChangedAt:  time.Now().Add(-100 * 24 * time.Hour), // 100 days ago
	}
	_ = userRepo.Create(context.Background(), u)

	svc := NewService(userRepo, sessionRepo, auditor, Policy{
		CookieName:        "session",
		MaxHours:          24,
		IdleMinutes:       30,
		MaxFailures:       5,
		LockoutMinutes:    15,
		MinPasswordLen:    8,
		PasswordExpiryDays: 90,
	})

	// Act
	err := svc.ChangePassword(context.Background(), "u_testuser", "currentPass123", "newPass123")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// PasswordChangedAt should be updated
	updated, _ := userRepo.FindByID(context.Background(), "u_testuser")
	if time.Since(updated.PasswordChangedAt) > time.Minute {
		t.Error("expected PasswordChangedAt to be updated to now")
	}
}

func TestLogin_ZeroExpiryDate_NoExpiryEnforcement(t *testing.T) {
	// Arrange
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	auditor := &mockAuditor{}

	hash, _ := user.HashPassword("currentPass123")
	u := &user.User{
		ID:                 "u_testuser",
		Username:           "testuser",
		FullName:           "Test User",
		PasswordHash:       hash,
		RoleCodes:          []string{"accountant"},
		Status:             user.StatusActive,
		PasswordChangedAt:  time.Time{}, // zero value - never changed
	}
	_ = userRepo.Create(context.Background(), u)

	svc := NewService(userRepo, sessionRepo, auditor, Policy{
		CookieName:        "session",
		MaxHours:          24,
		IdleMinutes:       30,
		MaxFailures:       5,
		LockoutMinutes:    15,
		MinPasswordLen:    8,
		PasswordExpiryDays: 90,
	})

	// Act
	ses, err := svc.Login(context.Background(), "testuser", "currentPass123", "127.0.0.1", "test-agent")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ses == nil {
		t.Fatal("expected session, got nil")
	}
}
