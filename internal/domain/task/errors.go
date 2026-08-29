package task

import "errors"

var (
	ErrNotFound   = errors.New("task run not found")
	ErrUnknown    = errors.New("unknown task")
	ErrSuspended  = errors.New("task is disabled")
	ErrInProgress = errors.New("task is already running")
)
