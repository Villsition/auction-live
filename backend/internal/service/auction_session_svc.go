package service

import (
	"context"
	"time"

	"auction/internal/model"
	"auction/internal/repository"
)

type AuctionSessionSvc struct {
	repo *repository.AuctionSessionRepo
}

func NewAuctionSessionSvc(repo *repository.AuctionSessionRepo) *AuctionSessionSvc {
	return &AuctionSessionSvc{repo: repo}
}

func (s *AuctionSessionSvc) Create(ctx context.Context, session *model.AuctionSession) error {
	return s.repo.Create(ctx, session)
}

func (s *AuctionSessionSvc) GetByID(ctx context.Context, id uint64) (*model.AuctionSession, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *AuctionSessionSvc) Update(ctx context.Context, id uint64, updates map[string]any) error {
	return s.repo.Update(ctx, id, updates)
}

func (s *AuctionSessionSvc) List(ctx context.Context, page model.PageRequest) ([]model.AuctionSession, int64, error) {
	return s.repo.List(ctx, page)
}

func (s *AuctionSessionSvc) ListByRoom(ctx context.Context, roomID uint64, page model.PageRequest) ([]model.AuctionSession, int64, error) {
	return s.repo.ListByRoom(ctx, roomID, page)
}

func (s *AuctionSessionSvc) ListByRoomWithProducts(ctx context.Context, roomID uint64) ([]model.AuctionSession, error) {
	return s.repo.ListByRoomWithProducts(ctx, roomID)
}

func (s *AuctionSessionSvc) ListByRoomSince(ctx context.Context, roomID uint64, since time.Time) ([]model.AuctionSession, error) {
	return s.repo.ListByRoomSince(ctx, roomID, since)
}

func (s *AuctionSessionSvc) CountBySeller(ctx context.Context, sellerID uint64) (map[int]int64, error) {
	return s.repo.CountBySeller(ctx, sellerID)
}

func (s *AuctionSessionSvc) ListActiveBySeller(ctx context.Context, sellerID uint64) ([]model.AuctionSession, error) {
	return s.repo.ListActiveBySeller(ctx, sellerID)
}

// CancelByRoom cancels all non-cancelled auction sessions in a room.
func (s *AuctionSessionSvc) CancelByRoom(ctx context.Context, roomID uint64) error {
	return s.repo.CancelByRoom(ctx, roomID)
}
