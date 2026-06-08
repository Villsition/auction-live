package scheduler

import (
	"context"
	"fmt"
	"time"

	"auction/internal/model"

	redisPkg "auction/pkg/redis"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LikeFlusher periodically syncs like counts from Redis to MySQL.
// Worst-case loss if Redis crashes: last 10s of likes.
type LikeFlusher struct {
	rdb    *redisPkg.Client
	db     *gorm.DB
	logger *zap.Logger
	ticker *time.Ticker
	stopCh chan struct{}
}

func NewLikeFlusher(rdb *redisPkg.Client, db *gorm.DB, logger *zap.Logger) *LikeFlusher {
	return &LikeFlusher{
		rdb:    rdb,
		db:     db,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

func (f *LikeFlusher) Start() {
	f.ticker = time.NewTicker(10 * time.Second)
	f.logger.Info("like flusher started (10s interval)")

	go func() {
		for {
			select {
			case <-f.ticker.C:
				f.flush()
			case <-f.stopCh:
				f.ticker.Stop()
				f.logger.Info("like flusher stopped")
				return
			}
		}
	}()
}

func (f *LikeFlusher) Stop() {
	close(f.stopCh)
}

func (f *LikeFlusher) flush() {
	ctx := context.Background()

	// Find all live rooms
	var rooms []model.LiveRoom
	if err := f.db.WithContext(ctx).
		Where("status = ?", model.LiveRoomStatusLive).
		Find(&rooms).Error; err != nil {
		return
	}

	for _, room := range rooms {
		likesKey := fmt.Sprintf("room:%d:likes", room.ID)
		flushedKey := fmt.Sprintf("room:%d:likes:flushed", room.ID)

		total, err := f.rdb.Get(ctx, likesKey).Int64()
		if err != nil || total == 0 {
			continue
		}

		flushed, err := f.rdb.Get(ctx, flushedKey).Int64()
		if err != nil {
			flushed = 0
		}

		delta := total - flushed
		if delta <= 0 {
			continue
		}

		// Increment MySQL by delta
		f.db.WithContext(ctx).Model(&model.LiveRoom{}).
			Where("id = ?", room.ID).
			Update("total_likes", gorm.Expr("total_likes + ?", delta))

		// Mark flushed
		f.rdb.Set(ctx, flushedKey, total, 0)

		f.logger.Debug("flushed likes",
			zap.Uint64("room", room.ID),
			zap.Int64("delta", delta),
			zap.Int64("total", total),
		)
	}
}
