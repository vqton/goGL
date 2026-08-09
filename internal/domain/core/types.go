package core

import "errors"

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
