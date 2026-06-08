package model

import "time"

type Order struct {
	BaseModel
	OrderNo    string      `gorm:"column:order_no;uniqueIndex:uk_order_no;type:varchar(32);not null;default:''" json:"order_no"`
	AuctionID  uint64      `gorm:"column:auction_id;not null" json:"auction_id"`
	BidID      uint64      `gorm:"column:bid_id;uniqueIndex:uk_bid_id;not null" json:"bid_id"`
	BuyerID    uint64      `gorm:"column:buyer_id;not null;index:idx_buyer_id" json:"buyer_id"`
	SellerID   uint64      `gorm:"column:seller_id;not null;index:idx_seller_status,priority:1;index:idx_seller_created,priority:1" json:"seller_id"`
	ProductID  uint64      `gorm:"column:product_id;not null" json:"product_id"`
	Amount     string      `gorm:"column:amount;type:decimal(15,2);not null" json:"amount"`
	Address    string      `gorm:"column:address;type:varchar(512);not null;default:''" json:"address"`
	Status     OrderStatus `gorm:"column:status;not null;default:0;index:idx_seller_status,priority:2" json:"status"`
	ExpiresAt  *time.Time  `gorm:"column:expires_at" json:"expires_at"`
	PaidAt     *time.Time  `gorm:"column:paid_at" json:"paid_at"`
	RefundedAt *time.Time  `gorm:"column:refunded_at" json:"refunded_at"`
	CreatedAt  time.Time   `gorm:"column:created_at;index:idx_seller_created,priority:2" json:"created_at"`

	// Calculated field (not stored)
	RemainingSec int64 `gorm:"-" json:"remaining_sec,omitempty"`
}

func (Order) TableName() string { return "orders" }

// SellerOrderItem is an enriched order view for the seller's order list page.
type SellerOrderItem struct {
	Order
	ProductTitle   string `json:"product_title"`
	ProductImage   string `json:"product_image"`
	BuyerNickname  string `json:"buyer_nickname"`
	BuyerAvatar    string `json:"buyer_avatar"`
}
