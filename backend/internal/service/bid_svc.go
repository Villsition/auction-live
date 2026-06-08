package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"auction/internal/model"
	"auction/internal/repository"
	"auction/internal/ws"

	"github.com/redis/go-redis/v9"
)

type BidSvc struct {
	repo    *repository.BidRepo
	notifSvc *NotificationSvc
	hub     *ws.Hub
}

func NewBidSvc(repo *repository.BidRepo, notifSvc *NotificationSvc, hub *ws.Hub) *BidSvc {
	return &BidSvc{repo: repo, notifSvc: notifSvc, hub: hub}
}

// PlaceBid validates and records a bid via Redis (atomic Lua) + persists to MySQL.
func (s *BidSvc) PlaceBid(ctx context.Context, bid *model.Bid) (*repository.BidResult, error) {
	locked, err := s.repo.AcquireBidLock(ctx, bid.AuctionID, bid.UserID, 2*time.Second)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, ErrBidTooFast
	}
	defer s.repo.ReleaseBidLock(ctx, bid.AuctionID, bid.UserID)

	// Idempotency: if client provides a key, ensure this request runs exactly once
	if bid.IdempotencyKey != "" {
		ok, err := s.repo.ClaimIdempotencyKey(ctx, bid.AuctionID, bid.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrDuplicateBid
		}
		// Release key on failure so client can retry
		defer func() {
			if err != nil {
				s.repo.ReleaseIdempotencyKey(ctx, bid.AuctionID, bid.IdempotencyKey)
			}
		}()
	}

	// Snapshot old leader before placing bid
	oldLeaderID := s.getLeaderID(ctx, bid.AuctionID)

	result, err := s.repo.PlaceBid(ctx, bid)
	if err != nil {
		return nil, err
	}

	// If ceiling deal, clamp bid amount to ceiling price before persisting
	if result.CeilingDeal {
		rules, _ := s.repo.GetAuctionRules(ctx, bid.AuctionID)
		if rules != nil {
			if cp, ok := rules["ceiling_price"]; ok && cp != "" {
				bid.Amount = cp
			}
		}
	}

	// Persist to MySQL synchronously so RecoverBidsFromMySQL can find it if Redis fails.
	if err := s.repo.Create(ctx, bid); err != nil {
		// Redis is the source of truth; MySQL write failure is logged but not fatal.
		log.Printf("bid mysql persist failed: auction=%d user=%d err=%v", bid.AuctionID, bid.UserID, err)
	}

	// Broadcast bid event
	bidCount, _ := s.repo.GetBidCount(ctx, bid.AuctionID)
	eventType := ws.EventBid
	if result.CeilingDeal {
		eventType = ws.EventCeilingDeal
	} else if result.DelayExtend {
		eventType = ws.EventDelayExtend
	}

	s.hub.BroadcastToAuction(bid.AuctionID, &ws.BidEvent{
		Type:        eventType,
		AuctionID:   bid.AuctionID,
		Amount:      bid.Amount,
		UserID:      bid.UserID,
		Nickname:    bid.Nickname,
		Rank:        result.Rank,
		BidCount:    bidCount,
		CeilingDeal: result.CeilingDeal,
		DelayExtend: result.DelayExtend,
		FinalDelay:  result.FinalDelay,
		NewEndTime:  result.NewEndTimestamp,
	})

	// Notification logic: leader change detection
	s.handleLeaderChange(ctx, bid, result, oldLeaderID)

	return result, nil
}

// handleLeaderChange detects leader changes and sends notifications.
func (s *BidSvc) handleLeaderChange(ctx context.Context, bid *model.Bid, result *repository.BidResult, oldLeaderID uint64) {
	if result.Rank != 1 {
		return
	}

	newLeaderID := bid.UserID

	if oldLeaderID == 0 {
		// First bid ever — just announce new leader
		s.hub.BroadcastToAuction(bid.AuctionID, &ws.NewLeaderEvent{
			Type:        ws.EventNewLeader,
			AuctionID:   bid.AuctionID,
			NewLeaderID: newLeaderID,
			Amount:      bid.Amount,
			Message:     "首位出价者！",
		})
		return
	}

	if oldLeaderID == newLeaderID {
		// Same user outbidding themselves — just a price update, no leader change
		return
	}

	// Leader changed: notify old leader, announce new leader
	s.notifSvc.NotifyOutbid(ctx, bid.AuctionID, oldLeaderID, bid.Amount, 2)

	s.hub.BroadcastToAuction(bid.AuctionID, &ws.NewLeaderEvent{
		Type:        ws.EventNewLeader,
		AuctionID:   bid.AuctionID,
		OldLeaderID: oldLeaderID,
		NewLeaderID: newLeaderID,
		Amount:      bid.Amount,
		Message:     "成为新榜首！",
	})
}

// getLeaderID returns the user_id of the current top bidder, or 0 if none.
func (s *BidSvc) getLeaderID(ctx context.Context, auctionID uint64) uint64 {
	winnerJSON, err := s.repo.GetHighestBidder(ctx, auctionID)
	if err != nil || winnerJSON == "" {
		return 0
	}
	var snap struct {
		UserID uint64 `json:"user_id"`
	}
	if json.Unmarshal([]byte(winnerJSON), &snap) != nil {
		return 0
	}
	return snap.UserID
}

// CacheAuctionRules caches auction session rules in Redis before starting.
func (s *BidSvc) CacheAuctionRules(ctx context.Context, session *model.AuctionSession) error {
	return s.repo.CacheAuctionRules(ctx, session)
}

func (s *BidSvc) Create(ctx context.Context, bid *model.Bid) error {
	return s.repo.Create(ctx, bid)
}

func (s *BidSvc) GetByID(ctx context.Context, id uint64) (*model.Bid, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *BidSvc) ListByAuction(ctx context.Context, auctionID uint64, page model.PageRequest) ([]model.Bid, int64, error) {
	return s.repo.ListByAuction(ctx, auctionID, page)
}

func (s *BidSvc) GetLatestBid(ctx context.Context, auctionID uint64) (*model.Bid, error) {
	return s.repo.GetLatestBid(ctx, auctionID)
}

func (s *BidSvc) GetCurrentPrice(ctx context.Context, auctionID uint64) (string, error) {
	return s.repo.GetCurrentPrice(ctx, auctionID)
}

func (s *BidSvc) GetBidRanking(ctx context.Context, auctionID uint64, topN int64) ([]redis.Z, error) {
	return s.repo.GetBidRanking(ctx, auctionID, topN)
}

func (s *BidSvc) GetBidCount(ctx context.Context, auctionID uint64) (int64, error) {
	return s.repo.GetBidCount(ctx, auctionID)
}

func (s *BidSvc) FlushAuctionCache(ctx context.Context, auctionID uint64) error {
	return s.repo.FlushAuctionCache(ctx, auctionID)
}

func (s *BidSvc) SetAuctionStatus(ctx context.Context, auctionID uint64, status model.AuctionStatus) error {
	return s.repo.SetAuctionStatus(ctx, auctionID, status)
}

func (s *BidSvc) GetAuctionStatus(ctx context.Context, auctionID uint64) (int, error) {
	return s.repo.GetAuctionStatus(ctx, auctionID)
}

func (s *BidSvc) GetUserBidInAuction(ctx context.Context, auctionID, userID uint64) (*repository.UserBidRank, error) {
	return s.repo.GetUserBidInAuction(ctx, auctionID, userID)
}

func (s *BidSvc) ListByUser(ctx context.Context, userID uint64, page model.PageRequest) ([]model.Bid, int64, error) {
	return s.repo.ListByUser(ctx, userID, page)
}

var ErrBidTooFast = repository.ErrBidTooFast
var ErrDuplicateBid = errors.New("duplicate bid request")
