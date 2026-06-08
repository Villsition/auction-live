package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"auction/internal/model"
	"auction/internal/repository"
	"auction/internal/service"
	"auction/internal/ws"
	redisPkg "auction/pkg/redis"

	"go.uber.org/zap"
)

type AuctionWatcher struct {
	rdb         *redisPkg.Client
	auctionRepo *repository.AuctionSessionRepo
	bidRepo     *repository.BidRepo
	orderRepo   *repository.OrderRepo
	notifSvc    *service.NotificationSvc
	hub         *ws.Hub
	logger      *zap.Logger
	ticker      *time.Ticker
	stopCh      chan struct{}
}

type bidSnapshot struct {
	UserID    uint64 `json:"user_id"`
	Amount    string `json:"amount"`
	AuctionID uint64 `json:"auction_id"`
}

func NewAuctionWatcher(
	rdb *redisPkg.Client,
	auctionRepo *repository.AuctionSessionRepo,
	bidRepo *repository.BidRepo,
	orderRepo *repository.OrderRepo,
	notifSvc *service.NotificationSvc,
	hub *ws.Hub,
	logger *zap.Logger,
) *AuctionWatcher {
	return &AuctionWatcher{
		rdb:         rdb,
		auctionRepo: auctionRepo,
		bidRepo:     bidRepo,
		orderRepo:   orderRepo,
		notifSvc:    notifSvc,
		hub:         hub,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}
}

func (w *AuctionWatcher) Start() {
	w.ticker = time.NewTicker(1 * time.Second)
	w.logger.Info("auction watcher started")

	go func() {
		tickN := 0
		for {
			select {
			case <-w.ticker.C:
				tickN++
				w.tick(tickN%10 == 0)
			case <-w.stopCh:
				w.ticker.Stop()
				w.logger.Info("auction watcher stopped")
				return
			}
		}
	}()
}

func (w *AuctionWatcher) Stop() {
	close(w.stopCh)
}

func (w *AuctionWatcher) tick(runMySQLFallback bool) {
	ctx := context.Background()

	// Primary: Redis deadline ZSET (runs every tick, sub-ms precision)
	seen := make(map[uint64]bool)
	expiredIDs, err := w.bidRepo.GetExpiredAuctions(ctx)
	if err != nil {
		w.logger.Error("failed to get expired auctions from Redis", zap.Error(err))
	} else {
		for _, id := range expiredIDs {
			seen[id] = true
			w.finalize(ctx, id)
		}
	}

	// MySQL fallback: runs every 10s to catch auctions Redis might have missed
	if !runMySQLFallback {
		return
	}
	mysqlExpired, err := w.auctionRepo.ListExpiredActiveAuctions(ctx)
	if err != nil {
		w.logger.Error("failed to query expired auctions from MySQL", zap.Error(err))
		return
	}
	for _, id := range mysqlExpired {
		if seen[id] {
			continue
		}
		// Guard: if Redis still has rules for this auction, trust Redis
		// (deadline was likely extended via Lua, MySQL planned_end_time is stale)
		if rules, err := w.bidRepo.GetAuctionRules(ctx, id); err == nil && len(rules) > 0 {
			w.logger.Debug("MySQL fallback skipped: auction still active in Redis",
				zap.Uint64("auction", id))
			continue
		}
		w.logger.Warn("MySQL fallback: picking up missed auction", zap.Uint64("auction", id))
		if _, err := w.auctionRepo.GetByIDWithProduct(ctx, id); err == nil {
			w.bidRepo.RecoverRules(ctx, id)
		}
		w.finalize(ctx, id)
	}
}

func (w *AuctionWatcher) finalize(ctx context.Context, auctionID uint64) {
	status, err := w.bidRepo.GetAuctionStatus(ctx, auctionID)
	if err != nil {
		// Redis status key missing — fall back to MySQL
		session, dbErr := w.auctionRepo.GetByID(ctx, auctionID)
		if dbErr != nil {
			w.logger.Error("failed to get auction status from both Redis and MySQL",
				zap.Uint64("auction", auctionID), zap.Error(err), zap.Error(dbErr))
			return
		}
		status = int(session.Status)
		w.logger.Warn("Redis status missing, falling back to MySQL",
			zap.Uint64("auction", auctionID), zap.Int("status", status))
	}
	if status != int(model.AuctionStatusActive) && status != int(model.AuctionStatusSold) {
		w.bidRepo.RemoveDeadline(ctx, auctionID)
		return
	}

	bidCount, _ := w.bidRepo.GetBidCount(ctx, auctionID)

	// MySQL fallback: if Redis bidCount is 0, check if there were actual bids in MySQL
	if bidCount == 0 {
		latestBid, err := w.bidRepo.GetLatestBid(ctx, auctionID)
		if err == nil && latestBid != nil {
			bidCount = 1 // found a bid, proceed to sold flow
			w.logger.Warn("MySQL fallback: recovered bid from DB", zap.Uint64("auction", auctionID))
		}
	}

	if bidCount == 0 {
		w.logger.Info("auction ended with no bids", zap.Uint64("auction", auctionID))
		if err := w.auctionRepo.FinalizeAuction(ctx, auctionID, model.AuctionStatusUnsold, nil, "", uint(bidCount)); err != nil {
			w.logger.Error("finalize unsold failed", zap.Uint64("auction", auctionID), zap.Error(err))
			return
		}
		w.bidRepo.FlushAuctionCache(ctx, auctionID)

		w.hub.BroadcastToAuction(auctionID, &ws.AuctionEvent{
			Type:      ws.EventAuctionEnd,
			AuctionID: auctionID,
			Status:    "unsold",
			Message:   "竞拍结束，无人出价",
		})
		w.hub.RemoveAuctionRoom(auctionID)
		return
	}

	// Get winner — Redis first, MySQL fallback
	var winner bidSnapshot
	var currentPrice string

	winnerJSON, err := w.bidRepo.GetHighestBidder(ctx, auctionID)
	if err != nil || winnerJSON == "" {
		// MySQL fallback: Redis bid data lost
		latestBid, dbErr := w.bidRepo.GetLatestBid(ctx, auctionID)
		if dbErr != nil || latestBid == nil {
			w.logger.Error("failed to get winner from both Redis and MySQL", zap.Uint64("auction", auctionID))
			return
		}
		winner = bidSnapshot{UserID: latestBid.UserID, Amount: latestBid.Amount, AuctionID: auctionID}
		currentPrice = latestBid.Amount
		w.logger.Warn("MySQL fallback: recovered winner from DB", zap.Uint64("auction", auctionID))
	} else {
		if err := json.Unmarshal([]byte(winnerJSON), &winner); err != nil {
			w.logger.Error("failed to parse winner JSON", zap.Uint64("auction", auctionID), zap.Error(err))
			return
		}
		currentPrice, _ = w.bidRepo.GetCurrentPrice(ctx, auctionID)
		if currentPrice == "" {
			currentPrice = winner.Amount
		}
	}

	// Look up the winner's bid from MySQL (already persisted synchronously on PlaceBid).
	// Fall back to creating one only if MySQL is missing (Redis recovery edge case).
	winningBid, err := w.bidRepo.GetLatestBid(ctx, auctionID)
	if err != nil || winningBid == nil || winningBid.UserID != winner.UserID {
		w.logger.Warn("winning bid not found in MySQL, creating fallback",
			zap.Uint64("auction", auctionID),
			zap.Uint64("winner", winner.UserID),
		)
		winningBid = &model.Bid{
			AuctionID: auctionID,
			UserID:    winner.UserID,
			Amount:    currentPrice,
			BidTime:   time.Now(),
			IsValid:   1,
		}
		if err := w.bidRepo.SyncSave(ctx, winningBid); err != nil {
			w.logger.Error("failed to save winning bid", zap.Uint64("auction", auctionID), zap.Error(err))
			return
		}
	}

	// Get auction session with product to determine seller
	session, err := w.auctionRepo.GetByIDWithProduct(ctx, auctionID)
	if err != nil || session.Product == nil {
		w.logger.Error("failed to get session with product", zap.Uint64("auction", auctionID), zap.Error(err))
		return
	}

	// Create order + finalize in a single DB transaction for atomicity.
	// If either fails, both roll back and the watcher retries cleanly next tick.
	expiresAt := time.Now().Add(30 * time.Minute)
	order := &model.Order{
		AuctionID: auctionID,
		BidID:     winningBid.ID,
		BuyerID:   winner.UserID,
		SellerID:  session.Product.SellerID,
		ProductID: session.ProductID,
		Amount:    currentPrice,
		Status:    model.OrderStatusUnpaid,
		ExpiresAt: &expiresAt,
	}
	if err := w.auctionRepo.FinalizeWithOrder(ctx, auctionID, model.AuctionStatusSold, &winner.UserID, currentPrice, order, uint(bidCount)); err != nil {
		w.logger.Error("finalize with order failed", zap.Uint64("auction", auctionID), zap.Error(err))
		return
	}

	w.logger.Info("order created",
		zap.String("order_no", order.OrderNo),
		zap.Uint64("auction", auctionID),
		zap.Uint64("buyer", winner.UserID),
		zap.String("amount", currentPrice),
	)

	// Notify winner
	productTitle := ""
	if session.Product != nil {
		productTitle = session.Product.Title
	}
	w.notifSvc.NotifyDeal(ctx, auctionID, winner.UserID, productTitle, currentPrice)

	w.bidRepo.FlushAuctionCache(ctx, auctionID)

	w.hub.BroadcastToAuction(auctionID, &ws.AuctionEvent{
		Type:       ws.EventAuctionEnd,
		AuctionID:  auctionID,
		Status:     "sold",
		WinnerID:   winner.UserID,
		FinalPrice: currentPrice,
		Message:    "竞拍成交",
	})
	w.hub.RemoveAuctionRoom(auctionID)
}
