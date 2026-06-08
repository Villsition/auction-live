package model

import "time"

type AuctionSession struct {
	BaseModel
	RoomID          uint64        `gorm:"column:room_id;not null" json:"room_id"`
	ProductID       uint64        `gorm:"column:product_id;not null" json:"product_id"`
	StartPrice      string        `gorm:"column:start_price;type:decimal(15,2);not null" json:"start_price"`
	CurrentPrice    string        `gorm:"column:current_price;type:decimal(15,2);not null" json:"current_price"`
	CeilingPrice    string        `gorm:"column:ceiling_price;type:decimal(15,2);not null;default:0.00" json:"ceiling_price"`
	BidIncrement    string        `gorm:"column:bid_increment;type:decimal(15,2);not null" json:"bid_increment"`
	DelaySeconds    uint          `gorm:"column:delay_seconds;not null;default:30" json:"delay_seconds"`
	StartTime       *time.Time    `gorm:"column:start_time" json:"start_time"`
	PlannedEndTime  *time.Time    `gorm:"column:planned_end_time" json:"planned_end_time"`
	ActualEndTime   *time.Time    `gorm:"column:actual_end_time" json:"actual_end_time"`
	WinnerID        *uint64       `gorm:"column:winner_id" json:"winner_id"`
	FinalPrice      *string       `gorm:"column:final_price;type:decimal(15,2)" json:"final_price"`
	BidCount        uint          `gorm:"column:bid_count;not null;default:0" json:"bid_count"`
	SortOrder       uint          `gorm:"column:sort_order;not null;default:0" json:"sort_order"`
	CancelReason    string        `gorm:"column:cancel_reason;type:varchar(512);not null;default:''" json:"cancel_reason"`
	CancelledBy     *uint64       `gorm:"column:cancelled_by" json:"cancelled_by"`
	CancelledAt     *time.Time    `gorm:"column:cancelled_at" json:"cancelled_at"`
	Status          AuctionStatus `gorm:"column:status;not null;default:0" json:"status"`

	// Associations (preload)
	Product  *Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	LiveRoom *LiveRoom `gorm:"foreignKey:RoomID" json:"live_room,omitempty"`
}

func (AuctionSession) TableName() string { return "auction_sessions" }
