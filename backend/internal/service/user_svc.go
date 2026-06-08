package service

import (
	"context"

	"auction/internal/model"
	"auction/internal/repository"
)

type UserSvc struct {
	repo *repository.UserRepo
}

func NewUserSvc(repo *repository.UserRepo) *UserSvc {
	return &UserSvc{repo: repo}
}

func (s *UserSvc) GetByID(ctx context.Context, id uint64) (*model.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserSvc) Update(ctx context.Context, id uint64, updates map[string]any) error {
	return s.repo.Update(ctx, id, updates)
}

func (s *UserSvc) List(ctx context.Context, page model.PageRequest) ([]model.User, int64, error) {
	return s.repo.List(ctx, page)
}

func (s *UserSvc) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return s.repo.GetByUsername(ctx, username)
}
