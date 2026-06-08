package model

type AuctionLog struct {
	BaseModel
	AuctionID  uint64  `gorm:"column:auction_id;not null" json:"auction_id"`
	OperatorID uint64  `gorm:"column:operator_id;not null" json:"operator_id"`
	Action     string  `gorm:"column:action;type:varchar(32);not null;default:''" json:"action"`
	Detail     JSONMap `gorm:"column:detail;type:json" json:"detail"`
}

func (AuctionLog) TableName() string { return "auction_logs" }
