package service

import (
	"context"

	"auction/internal/model"
	"auction/internal/repository"
)

type PaymentRecordSvc struct {
	repo *repository.PaymentRecordRepo
}

func NewPaymentRecordSvc(repo *repository.PaymentRecordRepo) *PaymentRecordSvc {
	return &PaymentRecordSvc{repo: repo}
}

func (s *PaymentRecordSvc) GetByID(ctx context.Context, id uint64) (*model.PaymentRecord, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PaymentRecordSvc) List(ctx context.Context, page model.PageRequest) ([]model.PaymentRecord, int64, error) {
	return s.repo.List(ctx, page)
}

func (s *PaymentRecordSvc) GetByTransaction(ctx context.Context, txnNo string) (*model.PaymentRecord, error) {
	return s.repo.GetByTransaction(ctx, txnNo)
}

func (s *PaymentRecordSvc) ListByOrder(ctx context.Context, orderID uint64) ([]model.PaymentRecord, error) {
	return s.repo.ListByOrder(ctx, orderID)
}
