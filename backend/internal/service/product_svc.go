package service

import (
	"context"

	"auction/internal/model"
	"auction/internal/repository"
)

type ProductSvc struct {
	repo *repository.ProductRepo
}

func NewProductSvc(repo *repository.ProductRepo) *ProductSvc {
	return &ProductSvc{repo: repo}
}

func (s *ProductSvc) Create(ctx context.Context, p *model.Product) error {
	return s.repo.Create(ctx, p)
}

func (s *ProductSvc) GetByID(ctx context.Context, id uint64) (*model.Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ProductSvc) Update(ctx context.Context, id uint64, updates map[string]any) error {
	return s.repo.Update(ctx, id, updates)
}

func (s *ProductSvc) List(ctx context.Context, page model.PageRequest) ([]model.Product, int64, error) {
	return s.repo.List(ctx, page)
}

func (s *ProductSvc) ListBySeller(ctx context.Context, sellerID uint64, page model.PageRequest) ([]model.Product, int64, error) {
	return s.repo.ListBySeller(ctx, sellerID, page)
}

func (s *ProductSvc) ListByStatus(ctx context.Context, status model.ProductStatus, page model.PageRequest) ([]model.Product, int64, error) {
	return s.repo.ListByStatus(ctx, status, page)
}

func (s *ProductSvc) ListWithAuction(ctx context.Context, sellerID uint64, keyword string, status int, page model.PageRequest) ([]repository.ProductWithAuction, int64, error) {
	return s.repo.ListWithAuction(ctx, sellerID, keyword, status, page)
}

func (s *ProductSvc) CountByStatus(ctx context.Context, sellerID uint64) (map[int]int64, error) {
	return s.repo.CountByStatus(ctx, sellerID)
}
