package repository

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"auction/internal/model"

	"gorm.io/gorm"
)

var ErrOrderExpired = errors.New("order payment deadline has expired")

type OrderRepo struct {
	*BaseRepo[model.Order]
}

func NewOrderRepo(db *gorm.DB) *OrderRepo {
	return &OrderRepo{BaseRepo: NewBaseRepo[model.Order](db)}
}

func (r *OrderRepo) CreateOrder(ctx context.Context, order *model.Order) error {
	if order.OrderNo == "" {
		order.OrderNo = generateOrderNo()
	}
	return r.DB.WithContext(ctx).Create(order).Error
}

func (r *OrderRepo) GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	var order model.Order
	err := r.DB.WithContext(ctx).Where("order_no = ?", orderNo).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// ConfirmAddress updates the shipping address on an unpaid order.
func (r *OrderRepo) ConfirmAddress(ctx context.Context, orderID uint64, address string) error {
	return r.DB.WithContext(ctx).Model(&model.Order{}).
		Where("id = ? AND status = ?", orderID, model.OrderStatusUnpaid).
		Update("address", address).Error
}

// MarkAsPaid marks an order as paid and records the payment time.
func (r *OrderRepo) MarkAsPaid(ctx context.Context, orderID uint64) error {
	now := time.Now()
	result := r.DB.WithContext(ctx).Model(&model.Order{}).
		Where("id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", orderID, model.OrderStatusUnpaid, now).
		Updates(map[string]any{
			"status":  model.OrderStatusPaid,
			"paid_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// Check if it's already paid or expired
		var order model.Order
		if err := r.DB.WithContext(ctx).Where("id = ?", orderID).First(&order).Error; err != nil {
			return err
		}
		if order.Status != model.OrderStatusUnpaid {
			return fmt.Errorf("order already processed")
		}
		return ErrOrderExpired
	}
	return nil
}

func (r *OrderRepo) ListByBuyer(ctx context.Context, buyerID uint64, page model.PageRequest) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.Order{}).Where("buyer_id = ?", buyerID)
	db.Count(&total)
	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("created_at DESC").Find(&orders).Error
	return orders, total, err
}

// ListByBuyerWithDetails joins products to return enriched order data for the buyer UI.
func (r *OrderRepo) ListByBuyerWithDetails(ctx context.Context, buyerID uint64, page model.PageRequest) ([]model.SellerOrderItem, int64, error) {
	var items []model.SellerOrderItem
	var total int64

	db := r.DB.WithContext(ctx).Table("orders").
		Select("orders.*, products.title AS product_title, COALESCE(products.cover_image,'') AS product_image, auction_sessions.start_time AS auction_start, users.nickname AS buyer_nickname, COALESCE(users.avatar,'') AS buyer_avatar").
		Joins("LEFT JOIN products ON products.id = orders.product_id").
		Joins("LEFT JOIN users ON users.id = orders.buyer_id").
		Joins("LEFT JOIN auction_sessions ON auction_sessions.id = orders.auction_id").
		Where("orders.buyer_id = ?", buyerID)

	db.Count(&total)
	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("orders.created_at DESC").Find(&items).Error
	return items, total, err
}

func (r *OrderRepo) ListBySeller(ctx context.Context, sellerID uint64, page model.PageRequest) ([]model.Order, int64, error) {
	var orders []model.Order
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.Order{}).Where("seller_id = ?", sellerID)
	db.Count(&total)
	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("created_at DESC").Find(&orders).Error
	return orders, total, err
}

// ListBySellerWithDetails joins products and users to return enriched order data for the seller UI.
func (r *OrderRepo) ListBySellerWithDetails(ctx context.Context, sellerID uint64, page model.PageRequest, statusFilter ...int) ([]model.SellerOrderItem, int64, error) {
	var items []model.SellerOrderItem
	var total int64

	db := r.DB.WithContext(ctx).Table("orders").
		Select("orders.*, products.title AS product_title, COALESCE(products.cover_image,'') AS product_image, auction_sessions.start_time AS auction_start, users.nickname AS buyer_nickname, COALESCE(users.avatar,'') AS buyer_avatar").
		Joins("LEFT JOIN products ON products.id = orders.product_id").
		Joins("LEFT JOIN users ON users.id = orders.buyer_id").
		Joins("LEFT JOIN auction_sessions ON auction_sessions.id = orders.auction_id").
		Where("orders.seller_id = ?", sellerID)

	if len(statusFilter) > 0 {
		db = db.Where("orders.status IN ?", statusFilter)
	}

	db.Count(&total)
	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("orders.created_at DESC").Find(&items).Error
	return items, total, err
}

// RevenueBySeller returns total revenue (sum of paid/completed orders) for a seller.
func (r *OrderRepo) RevenueBySeller(ctx context.Context, sellerID uint64) (string, error) {
	var total string
	err := r.DB.WithContext(ctx).Model(&model.Order{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("seller_id = ? AND status IN (?,?,?)",
			sellerID, model.OrderStatusPaid, model.OrderStatusShipped, model.OrderStatusCompleted).
		Scan(&total).Error
	return total, err
}

// CountByStatus returns order counts grouped by status for a seller.
func (r *OrderRepo) CountByStatus(ctx context.Context, sellerID uint64) (map[int]int64, error) {
	type row struct {
		Status int   `gorm:"column:status"`
		Cnt    int64 `gorm:"column:cnt"`
	}
	var rows []row
	err := r.DB.WithContext(ctx).Model(&model.Order{}).
		Select("status, COUNT(*) as cnt").
		Where("seller_id = ?", sellerID).
		Group("status").Find(&rows).Error

	m := map[int]int64{}
	for _, r := range rows {
		m[r.Status] = r.Cnt
	}
	return m, err
}

// UpdateStatus transitions an order from one status to another atomically.
func (r *OrderRepo) UpdateStatus(ctx context.Context, orderID uint64, from, to model.OrderStatus) error {
	result := r.DB.WithContext(ctx).Model(&model.Order{}).
		Where("id = ? AND status = ?", orderID, from).
		Update("status", to)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("order status transition not allowed or already processed")
	}
	return nil
}

func generateOrderNo() string {
	now := time.Now()
	prefix := now.Format("20060102150405")
	r := rand.New(rand.NewSource(now.UnixNano()))
	suffix := r.Intn(900000) + 100000
	return fmt.Sprintf("%s%d", prefix, suffix)
}
