package service

import (
	"context"
	"fmt"

	"auction/internal/model"
	"auction/internal/repository"
	"auction/internal/ws"
)

type NotificationSvc struct {
	repo *repository.NotificationRepo
	hub  *ws.Hub
}

func NewNotificationSvc(repo *repository.NotificationRepo, hub *ws.Hub) *NotificationSvc {
	return &NotificationSvc{repo: repo, hub: hub}
}

// Create persists a notification and pushes it via WebSocket.
func (s *NotificationSvc) Create(ctx context.Context, notif *model.Notification) error {
	if err := s.repo.Create(ctx, notif); err != nil {
		return err
	}
	// Push to user via WS with unread count
	count, _ := s.repo.UnreadCount(ctx, notif.UserID)
	s.hub.SendToUser(notif.RelatedID, notif.UserID, map[string]any{
		"type":         "notification",
		"id":           notif.ID,
		"title":        notif.Title,
		"content":      notif.Content,
		"notif_type":   notif.Type,
		"unread_count": count,
	})
	return nil
}

// NotifyOutbid creates a "被超越" notification when a user loses the top spot.
func (s *NotificationSvc) NotifyOutbid(ctx context.Context, auctionID, userID uint64, newAmount string, rank int64) {
	s.Create(ctx, &model.Notification{
		UserID:    userID,
		Title:     "出价被超越",
		Content:   fmt.Sprintf("有人出价 %s 元超过了您，当前排名第 %d", newAmount, rank),
		Type:      model.NotifTypeOutbid,
		RelatedID: auctionID,
	})
	// Personal WS event
	s.hub.SendToUser(auctionID, userID, &ws.OutbidEvent{
		Type:      ws.EventOutbid,
		AuctionID: auctionID,
		UserID:    userID,
		NewAmount: newAmount,
		MyRank:    rank,
	})
}

// NotifyDeal creates a "竞拍成功" notification for the winner.
func (s *NotificationSvc) NotifyDeal(ctx context.Context, auctionID, userID uint64, productTitle, finalPrice string) {
	s.Create(ctx, &model.Notification{
		UserID:    userID,
		Title:     "竞拍成功",
		Content:   fmt.Sprintf("恭喜！您以 %s 元拍得「%s」，请尽快支付", finalPrice, productTitle),
		Type:      model.NotifTypeDeal,
		RelatedID: auctionID,
	})
}

// NotifyAuctionStart notifies all viewers that the auction is starting.
// Sends a system notification to all connected users in the room.
func (s *NotificationSvc) NotifyAuctionStart(ctx context.Context, auctionID uint64, productTitle string) {
	s.hub.BroadcastToAuction(auctionID, &ws.AuctionEvent{
		Type:      ws.EventAuctionStart,
		AuctionID: auctionID,
		Status:    "started",
		Message:   "竞拍开始：" + productTitle,
	})
}

func (s *NotificationSvc) GetByID(ctx context.Context, id uint64) (*model.Notification, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *NotificationSvc) ListByUser(ctx context.Context, userID uint64, page model.PageRequest) ([]model.Notification, int64, error) {
	return s.repo.ListByUser(ctx, userID, page)
}

func (s *NotificationSvc) MarkRead(ctx context.Context, id, userID uint64) error {
	return s.repo.MarkRead(ctx, id, userID)
}

func (s *NotificationSvc) UnreadCount(ctx context.Context, userID uint64) (int64, error) {
	return s.repo.UnreadCount(ctx, userID)
}
