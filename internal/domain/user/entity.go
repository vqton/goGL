package user

import (
	"context"

	"goGL/internal/domain/core"
)

type Role struct {
	Code        string   `json:"code" bson:"code"`
	Name        string   `json:"name" bson:"name"`
	Permissions []string `json:"permissions" bson:"permissions"`
}

type User struct {
	ID        string      `json:"id" bson:"_id"`
	Username  string      `json:"username" bson:"username"`
	FullName  string      `json:"full_name" bson:"full_name"`
	RoleCodes []string    `json:"role_codes" bson:"role_codes"`
	Status    core.Status `json:"status" bson:"status"`
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, u *User) error
	SaveRole(ctx context.Context, r *Role) error
}
