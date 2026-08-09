package cash

import "errors"

var (
	ErrNotFound          = errors.New("cash: not found")
	ErrFundNotFound      = errors.New("cash: fund not found")
	ErrVoucherNotFound   = errors.New("cash: voucher not found")
	ErrFundInactive      = errors.New("cash: fund is inactive")
	ErrInvalidLines      = errors.New("cash: voucher lines must balance")
	ErrPeriodClosed      = errors.New("cash: accounting period is closed")
	ErrNegativeBalance   = errors.New("cash: insufficient fund balance")
	ErrWrongState        = errors.New("cash: voucher state does not permit the action")
	ErrSelfApproval      = errors.New("cash: approver must differ from preparer")
	ErrCashierRequired   = errors.New("cash: action requires the cashier role")
	ErrOpenCountPending  = errors.New("cash: open cash count blocks the operation")
	ErrReversalMissing   = errors.New("cash: posted void requires an offsetting reversal")
	ErrReversalMismatch  = errors.New("cash: reversal amount must equal the original")
	ErrUnauthorizedActor = errors.New("cash: actor is not authorized for this action")
	ErrInvalidSigners    = errors.New("cash: reconciliation requires three distinct signers")
	ErrInvalidCount      = errors.New("cash: counted amount cannot be negative")
	ErrUnpostedVouchers  = errors.New("cash: close day requires every voucher up to the date to be posted")
)
