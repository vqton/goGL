package user

import "errors"

var (
	ErrNotFound           = errors.New("user not found")
	ErrRoleNotFound       = errors.New("role not found")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrRoleExists         = errors.New("role already exists")
	ErrRoleInUse          = errors.New("role is assigned to users")
	ErrRoleSystemProtected = errors.New("system role cannot be modified")
	ErrLastAdmin          = errors.New("cannot remove the last active admin")
	ErrWeakPassword       = errors.New("password does not meet the minimum requirements")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrLockedOut          = errors.New("account is locked due to failed login attempts")
	ErrSuspended          = errors.New("account is suspended")
	ErrInvalidSession     = errors.New("invalid or expired session")
	ErrPasswordReused     = errors.New("new password cannot be the same as a recent password")
)
