package service

import (
	"context"

	"auction/internal/model"
	"auction/internal/repository"
)

type AuctionLogSvc struct {
	repo *repository.AuctionLogRepo
}

func NewAuctionLogSvc(repo *repository.AuctionLogRepo) *AuctionLogSvc {
	return &AuctionLogSvc{repo: repo}
}

func (s *AuctionLogSvc) Create(ctx context.Context, log *model.AuctionLog) error {
	return s.repo.Create(ctx, log)
}

func (s *AuctionLogSvc) ListByAuction(ctx context.Context, auctionID uint64, page model.PageRequest) ([]model.AuctionLog, int64, error) {
	return s.repo.ListByAuction(ctx, auctionID, page)
}
