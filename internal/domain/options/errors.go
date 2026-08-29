package options

import "errors"

var (
	ErrNotFound     = errors.New("option not found")
	ErrInvalidValue = errors.New("invalid option value for the declared type")
)
