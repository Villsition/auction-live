package repository

import (
	"context"

	"auction/internal/model"

	"gorm.io/gorm"
)

type BaseRepo[T any] struct {
	DB *gorm.DB
}

func NewBaseRepo[T any](db *gorm.DB) *BaseRepo[T] {
	return &BaseRepo[T]{DB: db}
}

func (r *BaseRepo[T]) WithContext(ctx context.Context) *gorm.DB {
	return r.DB.WithContext(ctx)
}

func (r *BaseRepo[T]) Create(ctx context.Context, entity *T) error {
	return r.DB.WithContext(ctx).Create(entity).Error
}

func (r *BaseRepo[T]) GetByID(ctx context.Context, id uint64) (*T, error) {
	var entity T
	err := r.DB.WithContext(ctx).Where("id = ?", id).First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *BaseRepo[T]) Update(ctx context.Context, id uint64, updates map[string]any) error {
	return r.DB.WithContext(ctx).Model(new(T)).Where("id = ?", id).Updates(updates).Error
}

func (r *BaseRepo[T]) Delete(ctx context.Context, id uint64) error {
	return r.DB.WithContext(ctx).Where("id = ?", id).Delete(new(T)).Error
}

func (r *BaseRepo[T]) List(ctx context.Context, page model.PageRequest) ([]T, int64, error) {
	var entities []T
	var total int64

	db := r.DB.WithContext(ctx).Model(new(T))
	db.Count(&total)

	err := db.Offset(page.Offset()).Limit(page.PageSize).Find(&entities).Error
	return entities, total, err
}

func (r *BaseRepo[T]) Exists(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(new(T)).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}
