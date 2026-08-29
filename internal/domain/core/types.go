package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

var ErrNotImplemented = errors.New("not implemented")

type Money struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type Period struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Status string

const (
	StatusDraft  Status = "draft"
	StatusPosted Status = "posted"
	StatusVoided Status = "voided"
)

func RowID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
