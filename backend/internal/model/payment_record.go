package model

import "time"

type PaymentRecord struct {
	BaseModel
	OrderID       uint64        `gorm:"column:order_id;not null" json:"order_id"`
	UserID        uint64        `gorm:"column:user_id;not null" json:"user_id"`
	TransactionNo string        `gorm:"column:transaction_no;uniqueIndex:uk_transaction;type:varchar(64);not null;default:''" json:"transaction_no"`
	Amount        string        `gorm:"column:amount;type:decimal(15,2);not null" json:"amount"`
	Status        PaymentStatus `gorm:"column:status;not null;default:0" json:"status"`
	PaidAt        *time.Time    `gorm:"column:paid_at" json:"paid_at"`
}

func (PaymentRecord) TableName() string { return "payment_records" }
