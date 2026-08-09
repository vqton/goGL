package user

import (
	"context"

	"goGL/internal/domain/core"
	"goGL/internal/domain/user"
)

type Service interface {
	CreateUser(ctx context.Context, u *user.User) error
	GetUser(ctx context.Context, id string) (*user.User, error)
	AssignRole(ctx context.Context, r *user.Role) error
}

type service struct {
	repo user.Repository
}

func NewService(repo user.Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateUser(ctx context.Context, u *user.User) error {
	// TODO: implement
	return core.ErrNotImplemented
}

func (s *service) GetUser(ctx context.Context, id string) (*user.User, error) {
	// TODO: implement
	return nil, core.ErrNotImplemented
}

func (s *service) AssignRole(ctx context.Context, r *user.Role) error {
	// TODO: implement
	return core.ErrNotImplemented
}
