package core

import "errors"

var ErrNotImplemented = errors.New("not implemented")

type Money struct {
	Amount   float64 `json:"amount" bson:"amount"`
	Currency string  `json:"currency" bson:"currency"`
}

type Period struct {
	From string `json:"from" bson:"from"`
	To   string `json:"to" bson:"to"`
}

type Status string

const (
	StatusDraft  Status = "draft"
	StatusPosted Status = "posted"
	StatusVoided Status = "voided"
)
