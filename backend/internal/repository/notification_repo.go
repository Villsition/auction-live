package repository

import (
	"context"

	"auction/internal/model"

	"gorm.io/gorm"
)

type NotificationRepo struct {
	*BaseRepo[model.Notification]
}

func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{BaseRepo: NewBaseRepo[model.Notification](db)}
}

func (r *NotificationRepo) ListByUser(ctx context.Context, userID uint64, page model.PageRequest) ([]model.Notification, int64, error) {
	var list []model.Notification
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.Notification{}).Where("user_id = ?", userID)
	db.Count(&total)

	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("created_at DESC").Find(&list).Error
	return list, total, err
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID uint64) error {
	return r.DB.WithContext(ctx).Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", 1).Error
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, userID uint64) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(&model.Notification{}).
		Where("user_id = ? AND is_read = 0", userID).
		Count(&count).Error
	return count, err
}
