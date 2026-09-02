package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"goGL/internal/domain/audit"
	"goGL/internal/domain/user"
)

// PolicyStore syncs casbin rules for role management. Satisfied structurally
// by *casbin.Enforcer (its variadic domain parameter matches).
type PolicyStore interface {
	AddPolicies(rules [][]string) (bool, error)
	RemovePolicies(rules [][]string) (bool, error)
	AddGroupingPolicies(rules [][]string) (bool, error)
	RemoveGroupingPolicies(rules [][]string) (bool, error)
	GetRolesForUser(user string, domain ...string) ([]string, error)
	GetFilteredPolicy(fieldIndex int, fieldValues ...string) ([][]string, error)
}

// Auditor records management actions. Satisfied by the audit service.
type Auditor interface {
	Record(ctx context.Context, l *audit.AuditLog) error
}

type Service interface {
	// CreateUser creates a user with a hashed password and links its roles
	// in casbin. The actor must not already exist.
	CreateUser(ctx context.Context, actor string, u *user.User, password string) error
	GetUser(ctx context.Context, id string) (*user.User, error)
	ListUsers(ctx context.Context) ([]*user.User, error)
	UpdateUser(ctx context.Context, actor string, u *user.User) error
	SuspendUser(ctx context.Context, actor, id string) error
	ActivateUser(ctx context.Context, actor, id string) error
	DeleteUser(ctx context.Context, actor, id string) error
	ResetPassword(ctx context.Context, actor, id, newPassword string) error
	// CreateRole stores a custom role and writes its casbin policies.
	CreateRole(ctx context.Context, actor string, r *user.Role) error
	UpdateRole(ctx context.Context, actor string, r *user.Role) error
	DeleteRole(ctx context.Context, actor, code string) error
	ListRoles(ctx context.Context) ([]*user.Role, error)
	// SeedRoles inserts the built-in roles (from casbin defaults) as system
	// roles. Idempotent.
	SeedRoles(ctx context.Context, codes []string) error
}

type service struct {
	repo     user.Repository
	policies PolicyStore
	auditor  Auditor
	builtin  map[string]bool
	now      func() time.Time
	minPwLen int
}

func NewService(repo user.Repository, policies PolicyStore, auditor Auditor, builtinRoles []string, minPasswordLen int) Service {
	builtin := make(map[string]bool, len(builtinRoles))
	for _, c := range builtinRoles {
		builtin[strings.TrimPrefix(c, "role:")] = true
	}
	return &service{
		repo:     repo,
		policies: policies,
		auditor:  auditor,
		builtin:  builtin,
		now:      time.Now,
		minPwLen: minPasswordLen,
	}
}

func (s *service) CreateUser(ctx context.Context, actor string, u *user.User, password string) error {
	if strings.TrimSpace(u.Username) == "" {
		return errors.New("username is required")
	}
	if len(password) < s.minPwLen {
		return user.ErrWeakPassword
	}
	if existing, err := s.repo.FindByUsername(ctx, u.Username); err == nil {
		_ = existing
		return user.ErrUsernameTaken
	} else if !errors.Is(err, user.ErrNotFound) {
		return err
	}

	hash, err := user.HashPassword(password)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	u.ID = newID(u.Username)
	u.PasswordHash = hash
	u.Status = user.StatusActive
	u.MustChangePassword = true
	u.CreatedAt = now
	u.UpdatedAt = now
	u.CreatedBy = actor
	if err := s.repo.Create(ctx, u); err != nil {
		return err
	}
	if err := s.syncRoles(ctx, u.Username, u.RoleCodes); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: actor, Module: "user", Action: "user.create", TargetID: u.ID, Timestamp: now.Format(time.RFC3339),
	})
}

func (s *service) GetUser(ctx context.Context, id string) (*user.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) ListUsers(ctx context.Context) ([]*user.User, error) {
	return s.repo.List(ctx)
}

func (s *service) UpdateUser(ctx context.Context, actor string, u *user.User) error {
	existing, err := s.repo.FindByID(ctx, u.ID)
	if err != nil {
		return err
	}
	if existing.IsSuspended() != (u.Status == user.StatusSuspended) && u.Status == user.StatusSuspended {
		return s.SuspendUser(ctx, actor, u.ID)
	}
	if existing.IsSuspended() && u.Status == user.StatusActive {
		return s.ActivateUser(ctx, actor, u.ID)
	}

	existing.FullName = u.FullName
	existing.RoleCodes = u.RoleCodes
	existing.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, existing); err != nil {
		return err
	}
	if err := s.syncRoles(ctx, existing.Username, u.RoleCodes); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: actor, Module: "user", Action: "user.update", TargetID: u.ID, Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) SuspendUser(ctx context.Context, actor, id string) error {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if u.IsSuspended() {
		return nil
	}
	if hasRole(u, "admin") {
		admins, err := s.activeAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return user.ErrLastAdmin
		}
	}
	u.Status = user.StatusSuspended
	u.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, u); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: actor, Module: "user", Action: "user.suspend", TargetID: id, Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) ActivateUser(ctx context.Context, actor, id string) error {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	u.Status = user.StatusActive
	u.FailedAttempts = 0
	u.LockedUntil = time.Time{}
	u.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, u); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: actor, Module: "user", Action: "user.activate", TargetID: id, Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) DeleteUser(ctx context.Context, actor, id string) error {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if hasRole(u, "admin") {
		admins, err := s.activeAdmins(ctx)
		if err != nil {
			return err
		}
		if admins <= 1 {
			return user.ErrLastAdmin
		}
	}
	if err := s.syncRoles(ctx, u.Username, nil); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: actor, Module: "user", Action: "user.delete", TargetID: id, Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) ResetPassword(ctx context.Context, actor, id, newPassword string) error {
	if len(newPassword) < s.minPwLen {
		return user.ErrWeakPassword
	}
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	hash, err := user.HashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	u.MustChangePassword = true
	u.FailedAttempts = 0
	u.LockedUntil = time.Time{}
	u.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(ctx, u); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: actor, Module: "user", Action: "user.reset-password", TargetID: id, Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) CreateRole(ctx context.Context, actor string, r *user.Role) error {
	code := strings.TrimSpace(r.Code)
	if code == "" {
		return errors.New("role code is required")
	}
	if s.builtin[code] {
		return user.ErrRoleSystemProtected
	}
	if existing, err := s.repo.FindRoleByCode(ctx, code); err == nil && existing != nil {
		return user.ErrRoleExists
	}
	r.Code = code
	r.IsSystem = false
	if err := s.repo.SaveRole(ctx, r); err != nil {
		return err
	}
	if err := s.syncRolePolicies(ctx, code, r.Policies); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: actor, Module: "user", Action: "role.create", TargetID: code, Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) UpdateRole(ctx context.Context, actor string, r *user.Role) error {
	if s.builtin[r.Code] {
		return user.ErrRoleSystemProtected
	}
	existing, err := s.repo.FindRoleByCode(ctx, r.Code)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return user.ErrRoleSystemProtected
	}
	existing.Name = r.Name
	existing.Description = r.Description
	existing.Policies = r.Policies
	if err := s.repo.SaveRole(ctx, existing); err != nil {
		return err
	}
	if err := s.syncRolePolicies(ctx, r.Code, r.Policies); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: actor, Module: "user", Action: "role.update", TargetID: r.Code, Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) DeleteRole(ctx context.Context, actor, code string) error {
	if s.builtin[code] {
		return user.ErrRoleSystemProtected
	}
	existing, err := s.repo.FindRoleByCode(ctx, code)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return user.ErrRoleSystemProtected
	}
	// Guard: a role assigned to any user cannot be deleted (referential
	// integrity through casbin grouping).
	users, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, u := range users {
		if hasRole(u, code) {
			return user.ErrRoleInUse
		}
	}
	if err := s.repo.DeleteRole(ctx, code); err != nil {
		return err
	}
	if _, err := s.policies.RemovePolicies(roleRules("role:"+code, existing.Policies)); err != nil {
		return err
	}
	return s.auditor.Record(ctx, &audit.AuditLog{
		UserCode: actor, Module: "user", Action: "role.delete", TargetID: code, Timestamp: s.now().UTC().Format(time.RFC3339),
	})
}

func (s *service) ListRoles(ctx context.Context) ([]*user.Role, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	for code := range s.builtin {
		found := false
		for _, r := range roles {
			if r.Code == code {
				found = true
				break
			}
		}
		if !found {
			roles = append(roles, &user.Role{Code: code, Name: code, IsSystem: true})
		}
	}
	return roles, nil
}

func (s *service) SeedRoles(ctx context.Context, codes []string) error {
	for _, c := range codes {
		code := strings.TrimPrefix(c, "role:")
		if _, err := s.repo.FindRoleByCode(ctx, code); err == nil {
			continue
		}
		if err := s.repo.SaveRole(ctx, &user.Role{Code: code, Name: code, IsSystem: true}); err != nil {
			return err
		}
	}
	return nil
}

// --- helpers ---------------------------------------------------------------

// syncRoles reconciles the user's casbin grouping rules to the given codes.
// The casbin subject for a user is the username (matches the header/session
// principal resolver, which returns the username).
func (s *service) syncRoles(ctx context.Context, username string, codes []string) error {
	current, err := s.policies.GetRolesForUser(username)
	if err != nil {
		return err
	}

	want := map[string]bool{}
	for _, c := range codes {
		want["role:"+strings.TrimPrefix(c, "role:")] = true
	}
	have := map[string]bool{}
	for _, r := range current {
		have[r] = true
	}

	var toAdd, toRemove [][]string
	for r := range want {
		if !have[r] {
			toAdd = append(toAdd, []string{username, r})
		}
	}
	for r := range have {
		if !want[r] {
			toRemove = append(toRemove, []string{username, r})
		}
	}
	if len(toAdd) > 0 {
		if _, err := s.policies.AddGroupingPolicies(toAdd); err != nil {
			return err
		}
	}
	if len(toRemove) > 0 {
		if _, err := s.policies.RemoveGroupingPolicies(toRemove); err != nil {
			return err
		}
	}
	return nil
}

// syncRolePolicies replaces a custom role's casbin permission rules: remove
// every p rule whose subject is role:<code>, then add the new ones.
func (s *service) syncRolePolicies(ctx context.Context, code string, policies []user.PolicyRule) error {
	subject := "role:" + code
	existing, err := s.policies.GetFilteredPolicy(0, subject)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		if _, err := s.policies.RemovePolicies(existing); err != nil {
			return err
		}
	}
	if rules := roleRules(subject, policies); len(rules) > 0 {
		if _, err := s.policies.AddPolicies(rules); err != nil {
			return err
		}
	}
	return nil
}

func roleRules(subject string, policies []user.PolicyRule) [][]string {
	out := make([][]string, 0, len(policies))
	for _, p := range policies {
		if p.Object == "" || p.Action == "" {
			continue
		}
		out = append(out, []string{subject, p.Object, p.Action})
	}
	return out
}

func hasRole(u *user.User, code string) bool {
	for _, c := range u.RoleCodes {
		if c == code || c == "role:"+code {
			return true
		}
	}
	return false
}

func (s *service) activeAdmins(ctx context.Context) (int, error) {
	users, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, u := range users {
		if u.Status == user.StatusActive && hasRole(u, "admin") {
			n++
		}
	}
	return n, nil
}

func newID(username string) string {
	return "u_" + username
}
