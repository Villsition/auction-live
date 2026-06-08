package repository

import (
	"context"

	"auction/internal/model"

	"gorm.io/gorm"
)

type UserRepo struct {
	*BaseRepo[model.User]
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{BaseRepo: NewBaseRepo[model.User](db)}
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.DB.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) ListByRole(ctx context.Context, role model.UserRole, page model.PageRequest) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.User{}).Where("role = ?", role)
	db.Count(&total)

	err := db.Offset(page.Offset()).Limit(page.PageSize).Find(&users).Error
	return users, total, err
}
