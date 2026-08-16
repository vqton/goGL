package masterdata

import (
	"errors"
	"fmt"
)

// Sentinel errors. Callers compare with errors.Is; the HTTP layer maps them to
// status codes in one place.
var (
	ErrNotFound       = errors.New("masterdata: record not found")
	ErrDuplicate      = errors.New("masterdata: duplicate code")
	ErrUnknownKind    = errors.New("masterdata: unknown kind")
	ErrBlockedRefs    = errors.New("masterdata: record is referenced")
	ErrCycle          = errors.New("masterdata: group cycle")
	ErrInactive       = errors.New("masterdata: record is inactive")
	ErrCodeImmutable  = errors.New("masterdata: code is immutable")
	ErrRegimeMismatch = errors.New("masterdata: regime mismatch")
)

// ValidationError carries a user-facing message (VN + EN) for invalid master
// data. It is what handlers render as a 422.
type ValidationError struct {
	Kind      Kind
	Code      string
	MessageVn string
	MessageEn string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("masterdata: invalid %s %q: %s", e.Kind, e.Code, e.MessageEn)
}
