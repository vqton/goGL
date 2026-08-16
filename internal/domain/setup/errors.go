package setup

import "errors"

var (
	ErrAlreadyInitialized = errors.New("setup: system already initialized")
	ErrNotInitialized     = errors.New("setup: system not initialized")
	ErrWrongState         = errors.New("setup: status does not permit the action")
	ErrInvalidProfile     = errors.New("setup: invalid company profile")
	ErrInvalidTaxCode     = errors.New("setup: invalid MST (tax code)")
	ErrInvalidFiscalYear  = errors.New("setup: fiscal year must start on the 1st of a month and span 12 months")
	ErrInvalidCurrency    = errors.New("setup: accounting currency must be VND in v1")
	ErrInvalidRegime      = errors.New("setup: unsupported accounting regime")
	ErrAccountNotFound    = errors.New("setup: account does not exist or is not postable")
	ErrInvalidBalance     = errors.New("setup: invalid opening balance")
	ErrUnbalanced         = errors.New("setup: opening balances do not balance (sum of debits != sum of credits)")
	ErrObjectRequired     = errors.New("setup: object detail is required for this account")
	ErrObjectNotFound     = errors.New("setup: object does not exist or is inactive")
	ErrBalanceLocked      = errors.New("setup: opening balances are locked")
	ErrReopenBlocked      = errors.New("setup: reopen blocked — posted vouchers reference this account")
	ErrBalanceNotFound    = errors.New("setup: opening balance not found")
)

// ValidationError is a client-facing 422 carrying VN + EN messages, mirroring
// masterdata's convention.
type ValidationError struct {
	Field     string
	MessageVn string
	MessageEn string
}

func (e *ValidationError) Error() string { return e.MessageEn }
