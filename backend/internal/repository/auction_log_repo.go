package repository

import (
	"context"

	"auction/internal/model"

	"gorm.io/gorm"
)

type AuctionLogRepo struct {
	*BaseRepo[model.AuctionLog]
}

func NewAuctionLogRepo(db *gorm.DB) *AuctionLogRepo {
	return &AuctionLogRepo{BaseRepo: NewBaseRepo[model.AuctionLog](db)}
}

func (r *AuctionLogRepo) ListByAuction(ctx context.Context, auctionID uint64, page model.PageRequest) ([]model.AuctionLog, int64, error) {
	var logs []model.AuctionLog
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.AuctionLog{}).Where("auction_id = ?", auctionID)
	db.Count(&total)

	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("created_at DESC").Find(&logs).Error
	return logs, total, err
}
