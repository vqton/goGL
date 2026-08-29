package options

import (
	"context"
	"time"
)

// ValueType describes how an option value is stored and validated.
type ValueType string

const (
	TypeString ValueType = "string"
	TypeInt    ValueType = "int"
	TypeBool   ValueType = "bool"
)

// Option is a typed, validated system setting. Values are stored as their
// canonical string form ("true", "42", …); the service validates against the
// declared type before persisting. DefaultValue is returned when no value has
// been set, and Reset restores it.
type Option struct {
	ID           string    `json:"id"`
	Key          string    `json:"key"`
	Category     string    `json:"category"`
	Type         ValueType `json:"type"`
	Value        string    `json:"value"`
	DefaultValue string    `json:"default_value"`
	Description  string    `json:"description"`
	UpdatedAt    time.Time `json:"updated_at"`
	UpdatedBy    string    `json:"updated_by"`
}

type Repository interface {
	Create(ctx context.Context, o *Option) error
	FindByKey(ctx context.Context, key string) (*Option, error)
	Update(ctx context.Context, o *Option) error
	List(ctx context.Context) ([]*Option, error)
}
