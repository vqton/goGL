package ledger

import "errors"

var (
	ErrNotFound         = errors.New("ledger: entry not found")
	ErrInvalidDate      = errors.New("ledger: voucher date is required")
	ErrUnbalanced       = errors.New("ledger: entry must balance (sum of debits = sum of credits)")
	ErrInvalidLine      = errors.New("ledger: each line must have exactly one side (debit or credit) above zero")
	ErrAccountNotFound  = errors.New("ledger: account not found")
	ErrAccountInactive  = errors.New("ledger: account is not postable")
	ErrPeriodClosed     = errors.New("ledger: accounting period is closed")
	ErrDuplicateSource  = errors.New("ledger: an entry for this source already exists")
	ErrWrongState       = errors.New("ledger: entry state does not permit the action")
	ErrReversalMismatch = errors.New("ledger: reversal must negate the original entry exactly")

	ErrInvalidAccount      = errors.New("ledger: account code and name are required")
	ErrInvalidType         = errors.New("ledger: unknown account type")
	ErrInvalidLevel        = errors.New("ledger: account level must be between 1 and 6")
	ErrParentNotFound      = errors.New("ledger: parent account not found")
	ErrInvalidHierarchy    = errors.New("ledger: account level must be one more than its parent")
	ErrTypeMismatch        = errors.New("ledger: account type must match its parent")
	ErrInvalidPeriod       = errors.New("ledger: period id must be YYYY-MM")
	ErrInvalidRange        = errors.New("ledger: from period must be before or equal to to period")
	ErrCloseReasonRequired = errors.New("ledger: closing or reopening a period requires a reason")
)
