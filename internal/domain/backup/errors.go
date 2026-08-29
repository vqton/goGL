package backup

import "errors"

var (
	ErrNotFound       = errors.New("backup record not found")
	ErrIntegrity      = errors.New("backup failed integrity verification")
	ErrNoActivePlan   = errors.New("no staged restore plan awaiting approval")
	ErrEmptyArtifact  = errors.New("artifact has no file")
)
