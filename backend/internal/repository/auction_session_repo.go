package repository

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"auction/internal/model"
	redisPkg "auction/pkg/redis"

	"gorm.io/gorm"
)

type AuctionSessionRepo struct {
	*BaseRepo[model.AuctionSession]
	rdb     *redisPkg.Client
	rdbRead *redisPkg.Client
}

func NewAuctionSessionRepo(db *gorm.DB, rdb, rdbRead *redisPkg.Client) *AuctionSessionRepo {
	return &AuctionSessionRepo{BaseRepo: NewBaseRepo[model.AuctionSession](db), rdb: rdb, rdbRead: rdbRead}
}

func (r *AuctionSessionRepo) ListByRoom(ctx context.Context, roomID uint64, page model.PageRequest) ([]model.AuctionSession, int64, error) {
	var sessions []model.AuctionSession
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.AuctionSession{}).Where("room_id = ?", roomID)
	db.Count(&total)
	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("sort_order ASC").Find(&sessions).Error
	return sessions, total, err
}

// ListByRoomWithProducts returns all auction sessions for a room with preloaded products.
func (r *AuctionSessionRepo) ListByRoomWithProducts(ctx context.Context, roomID uint64) ([]model.AuctionSession, error) {
	var sessions []model.AuctionSession
	err := r.DB.WithContext(ctx).
		Preload("Product").
		Where("room_id = ?", roomID).
		Order("sort_order ASC").
		Find(&sessions).Error
	return sessions, err
}

// ListByRoomSince returns auction sessions for a room created after the given time.
// Used to scope results to the current live session only.
func (r *AuctionSessionRepo) ListByRoomSince(ctx context.Context, roomID uint64, since time.Time) ([]model.AuctionSession, error) {
	var sessions []model.AuctionSession
	err := r.DB.WithContext(ctx).
		Preload("Product").
		Where("room_id = ? AND created_at >= ?", roomID, since).
		Order("sort_order ASC").
		Find(&sessions).Error
	return sessions, err
}

func (r *AuctionSessionRepo) GetActiveByProduct(ctx context.Context, productID uint64) (*model.AuctionSession, error) {
	var session model.AuctionSession
	err := r.DB.WithContext(ctx).
		Where("product_id = ? AND status = ?", productID, model.AuctionStatusActive).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// FinalizeAuction updates the auction session result and product status in MySQL.
// Called by AuctionWatcher when deadline expires.
func (r *AuctionSessionRepo) FinalizeAuction(ctx context.Context, auctionID uint64, status model.AuctionStatus, winnerID *uint64, finalPrice string, bidCount uint) error {
	updates := map[string]any{
		"status":          status,
		"actual_end_time": gorm.Expr("NOW()"),
	}
	if winnerID != nil {
		updates["winner_id"] = *winnerID
		updates["final_price"] = finalPrice
	}
	if bidCount > 0 {
		updates["bid_count"] = bidCount
	}

	tx := r.DB.WithContext(ctx).Begin()

	// Update auction session
	if err := tx.Model(&model.AuctionSession{}).Where("id = ?", auctionID).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Get the session to find product_id
	var session model.AuctionSession
	if err := tx.Where("id = ?", auctionID).First(&session).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update product status
	if status == model.AuctionStatusSold {
		if err := tx.Model(&model.Product{}).Where("id = ?", session.ProductID).
			Update("status", model.ProductStatusSold).Error; err != nil {
			tx.Rollback()
			return err
		}
	} else if status == model.AuctionStatusUnsold {
		if err := tx.Model(&model.Product{}).Where("id = ?", session.ProductID).
			Update("status", model.ProductStatusListed).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// FinalizeWithOrder finalizes the auction and creates the order in a single transaction.
// If order creation fails, the auction status is NOT updated, allowing safe retry.
func (r *AuctionSessionRepo) FinalizeWithOrder(ctx context.Context, auctionID uint64, status model.AuctionStatus, winnerID *uint64, finalPrice string, order *model.Order, bidCount uint) error {
	updates := map[string]any{
		"status":          status,
		"actual_end_time": gorm.Expr("NOW()"),
	}
	if winnerID != nil {
		updates["winner_id"] = *winnerID
		updates["final_price"] = finalPrice
	}
	if bidCount > 0 {
		updates["bid_count"] = bidCount
	}

	tx := r.DB.WithContext(ctx).Begin()

	// Update auction session
	if err := tx.Model(&model.AuctionSession{}).Where("id = ?", auctionID).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Get the session to find product_id
	var session model.AuctionSession
	if err := tx.Where("id = ?", auctionID).First(&session).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update product status
	if status == model.AuctionStatusSold {
		if err := tx.Model(&model.Product{}).Where("id = ?", session.ProductID).
			Update("status", model.ProductStatusSold).Error; err != nil {
			tx.Rollback()
			return err
		}
	} else if status == model.AuctionStatusUnsold {
		if err := tx.Model(&model.Product{}).Where("id = ?", session.ProductID).
			Update("status", model.ProductStatusListed).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Create order (inside same transaction)
	if order.OrderNo == "" {
		order.OrderNo = newOrderNo()
	}
	if err := tx.Create(order).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ListActiveAuctionIDs returns IDs of all active auctions from MySQL (for recovery/startup).
func (r *AuctionSessionRepo) ListActiveAuctionIDs(ctx context.Context) ([]uint64, error) {
	var ids []uint64
	err := r.DB.WithContext(ctx).Model(&model.AuctionSession{}).
		Where("status = ?", model.AuctionStatusActive).
		Pluck("id", &ids).Error
	return ids, err
}

// GetByIDWithProduct loads an auction session with its product association.
func (r *AuctionSessionRepo) GetByIDWithProduct(ctx context.Context, id uint64) (*model.AuctionSession, error) {
	var session model.AuctionSession
	err := r.DB.WithContext(ctx).Preload("Product").Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// CountBySeller returns auction session counts grouped by status for a seller's products.
func (r *AuctionSessionRepo) CountBySeller(ctx context.Context, sellerID uint64) (map[int]int64, error) {
	type row struct {
		Status int   `gorm:"column:status"`
		Cnt    int64 `gorm:"column:cnt"`
	}
	var rows []row
	err := r.DB.WithContext(ctx).Model(&model.AuctionSession{}).
		Select("status, COUNT(*) as cnt").
		Where("product_id IN (SELECT id FROM products WHERE seller_id = ?)", sellerID).
		Group("status").Find(&rows).Error

	m := map[int]int64{}
	for _, r := range rows {
		m[r.Status] = r.Cnt
	}
	return m, err
}

// CancelByRoom cancels all active/pending auction sessions for a room (soft cancel).
func (r *AuctionSessionRepo) CancelByRoom(ctx context.Context, roomID uint64) error {
	return r.DB.WithContext(ctx).Model(&model.AuctionSession{}).
		Where("room_id = ? AND status IN (?, ?)", roomID, model.AuctionStatusActive, model.AuctionStatusPending).
		Updates(map[string]any{"status": model.AuctionStatusCancelled, "cancel_reason": "直播结束", "actual_end_time": gorm.Expr("NOW()")}).Error
}

// ListActiveBySeller returns the seller's currently active auction sessions with product info.
func (r *AuctionSessionRepo) ListActiveBySeller(ctx context.Context, sellerID uint64) ([]model.AuctionSession, error) {
	var sessions []model.AuctionSession
	err := r.DB.WithContext(ctx).Preload("Product").
		Where("status = ? AND product_id IN (SELECT id FROM products WHERE seller_id = ?)",
			model.AuctionStatusActive, sellerID).
		Order("sort_order ASC").Find(&sessions).Error
	return sessions, err
}

// ListExpiredActiveAuctions returns active auctions whose planned_end_time has passed.
// Used as MySQL fallback when Redis deadline ZSET is unavailable.
func (r *AuctionSessionRepo) ListExpiredActiveAuctions(ctx context.Context) ([]uint64, error) {
	var ids []uint64
	err := r.DB.WithContext(ctx).Model(&model.AuctionSession{}).
		Where("status = ? AND planned_end_time IS NOT NULL AND planned_end_time <= NOW()", model.AuctionStatusActive).
		Pluck("id", &ids).Error
	return ids, err
}

func newOrderNo() string {
	now := time.Now()
	prefix := now.Format("20060102150405")
	r := rand.New(rand.NewSource(now.UnixNano()))
	return fmt.Sprintf("%s%d", prefix, r.Intn(900000)+100000)
}
