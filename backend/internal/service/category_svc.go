package service

import (
	"context"

	"auction/internal/model"
	"auction/internal/repository"
)

type CategorySvc struct {
	repo *repository.CategoryRepo
}

func NewCategorySvc(repo *repository.CategoryRepo) *CategorySvc {
	return &CategorySvc{repo: repo}
}

func (s *CategorySvc) GetByID(ctx context.Context, id uint64) (*model.Category, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CategorySvc) List(ctx context.Context, page model.PageRequest) ([]model.Category, int64, error) {
	return s.repo.List(ctx, page)
}

func (s *CategorySvc) ListByParent(ctx context.Context, parentID uint64) ([]model.Category, error) {
	return s.repo.ListByParent(ctx, parentID)
}
