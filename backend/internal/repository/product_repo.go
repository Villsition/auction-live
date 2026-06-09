package repository

import (
	"context"
	"strconv"
	"time"

	"auction/internal/model"

	"gorm.io/gorm"
)

type ProductRepo struct {
	*BaseRepo[model.Product]
}

func NewProductRepo(db *gorm.DB) *ProductRepo {
	return &ProductRepo{BaseRepo: NewBaseRepo[model.Product](db)}
}

func (r *ProductRepo) ListBySeller(ctx context.Context, sellerID uint64, page model.PageRequest) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.Product{}).Where("seller_id = ?", sellerID)
	db.Count(&total)

	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("created_at DESC").Find(&products).Error
	return products, total, err
}

func (r *ProductRepo) ListByStatus(ctx context.Context, status model.ProductStatus, page model.PageRequest) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.Product{}).Where("status = ?", status)
	db.Count(&total)

	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("created_at DESC").Find(&products).Error
	return products, total, err
}

// ProductWithAuction holds a product row joined with its latest auction session.
type ProductWithAuction struct {
	model.Product
	AuctionID     *uint64 `gorm:"column:auction_id"     json:"auction_id"`
	AuctionStatus *uint8  `gorm:"column:auction_status" json:"auction_status"`
	CurrentPrice  *string `gorm:"column:auc_current_price" json:"auc_current_price"`
	FinalPrice    *string    `gorm:"column:auc_final_price"   json:"auc_final_price"`
	BidCount      *uint      `gorm:"column:auc_bid_count"     json:"auc_bid_count"`
	AuctionStart  *time.Time `gorm:"column:auc_start_time"    json:"auction_start"`
}

// ListWithAuction returns seller's products with their latest auction session info.
// Supports keyword search (title or ID) and status filter.
func (r *ProductRepo) ListWithAuction(ctx context.Context, sellerID uint64, keyword string, status int, page model.PageRequest) ([]ProductWithAuction, int64, error) {
	var rows []ProductWithAuction
	var total int64

	base := r.DB.WithContext(ctx).Table("products p").
		Joins(`LEFT JOIN auction_sessions auc ON auc.product_id = p.id
			AND auc.id = (SELECT MAX(id) FROM auction_sessions WHERE product_id = p.id)`).
		Where("p.seller_id = ?", sellerID)

	if keyword != "" {
		if id, err := strconv.ParseUint(keyword, 10, 64); err == nil {
			base = base.Where("(p.title LIKE ? OR p.id = ?)", "%"+keyword+"%", id)
		} else {
			base = base.Where("p.title LIKE ?", "%"+keyword+"%")
		}
	}
	if status >= 0 {
		base = base.Where("p.status = ?", status)
	}

	base.Count(&total)

	err := base.Select(`p.*, auc.id as auction_id, auc.status as auction_status,
		auc.current_price as auc_current_price, auc.final_price as auc_final_price,
		auc.bid_count as auc_bid_count, auc.start_time as auc_start_time`).
		Offset(page.Offset()).Limit(page.PageSize).
		Order("p.created_at DESC").Find(&rows).Error
	return rows, total, err
}

// CountByStatus returns product counts grouped by status for a seller.
func (r *ProductRepo) CountByStatus(ctx context.Context, sellerID uint64) (map[int]int64, error) {
	type row struct {
		Status int   `gorm:"column:status"`
		Cnt    int64 `gorm:"column:cnt"`
	}
	var rows []row
	err := r.DB.WithContext(ctx).Model(&model.Product{}).
		Select("status, COUNT(*) as cnt").
		Where("seller_id = ?", sellerID).
		Group("status").Find(&rows).Error

	m := map[int]int64{}
	for _, r := range rows {
		m[r.Status] = r.Cnt
	}
	return m, err
}
