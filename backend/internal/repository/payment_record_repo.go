package repository

import (
	"context"

	"auction/internal/model"

	"gorm.io/gorm"
)

type PaymentRecordRepo struct {
	*BaseRepo[model.PaymentRecord]
}

func NewPaymentRecordRepo(db *gorm.DB) *PaymentRecordRepo {
	return &PaymentRecordRepo{BaseRepo: NewBaseRepo[model.PaymentRecord](db)}
}

func (r *PaymentRecordRepo) GetByTransaction(ctx context.Context, txnNo string) (*model.PaymentRecord, error) {
	var record model.PaymentRecord
	err := r.DB.WithContext(ctx).Where("transaction_no = ?", txnNo).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *PaymentRecordRepo) ListByOrder(ctx context.Context, orderID uint64) ([]model.PaymentRecord, error) {
	var records []model.PaymentRecord
	err := r.DB.WithContext(ctx).Where("order_id = ?", orderID).Order("created_at DESC").Find(&records).Error
	return records, err
}
