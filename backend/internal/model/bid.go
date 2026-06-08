package model

import "time"

type Bid struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AuctionID  uint64    `gorm:"column:auction_id;not null" json:"auction_id"`
	UserID     uint64    `gorm:"column:user_id;not null" json:"user_id"`
	Amount     string    `gorm:"column:amount;type:decimal(15,2);not null" json:"amount"`
	BidTime    time.Time `gorm:"column:bid_time;not null" json:"bid_time"`
	ClientIP   string    `gorm:"column:client_ip;type:varchar(64);not null;default:''" json:"client_ip"`
	IsValid    uint8     `gorm:"column:is_valid;not null;default:1" json:"is_valid"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Transient fields (not persisted, used for Redis ZSET member & broadcasts)
	Nickname       string `gorm:"-" json:"nickname,omitempty"`
	Avatar         string `gorm:"-" json:"avatar,omitempty"`
	IdempotencyKey string `gorm:"-" json:"idempotency_key,omitempty"`
}

func (Bid) TableName() string { return "bids" }
