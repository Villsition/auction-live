package repository

import (
	"context"

	"auction/internal/model"

	"gorm.io/gorm"
)

type CategoryRepo struct {
	*BaseRepo[model.Category]
}

func NewCategoryRepo(db *gorm.DB) *CategoryRepo {
	return &CategoryRepo{BaseRepo: NewBaseRepo[model.Category](db)}
}

func (r *CategoryRepo) ListByParent(ctx context.Context, parentID uint64) ([]model.Category, error) {
	var list []model.Category
	err := r.DB.WithContext(ctx).Where("parent_id = ? AND status = 1", parentID).Order("sort ASC").Find(&list).Error
	return list, err
}
