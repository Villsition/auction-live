package model

type Product struct {
	BaseModel
	SellerID      uint64        `gorm:"column:seller_id;not null" json:"seller_id"`
	CategoryID    uint64        `gorm:"column:category_id;not null;default:0" json:"category_id"`
	Title         string        `gorm:"column:title;type:varchar(256);not null;default:''" json:"title"`
	Description   string        `gorm:"column:description;type:text" json:"description"`
	CoverImage    string        `gorm:"column:cover_image;type:varchar(512);not null;default:''" json:"cover_image"`
	Images        StringArray   `gorm:"column:images;type:json" json:"images"`
	StartPrice    string        `gorm:"column:start_price;type:decimal(15,2);not null;default:0.00" json:"start_price"`
	ReservePrice  string        `gorm:"column:reserve_price;type:decimal(15,2);not null;default:0.00" json:"reserve_price"`
	CeilingPrice  string        `gorm:"column:ceiling_price;type:decimal(15,2);not null;default:0.00" json:"ceiling_price"`
	BidIncrement  string        `gorm:"column:bid_increment;type:decimal(15,2);not null;default:1.00" json:"bid_increment"`
	DelaySeconds  uint          `gorm:"column:delay_seconds;not null;default:30" json:"delay_seconds"`
	DurationMin   int           `gorm:"column:duration_min;not null;default:5" json:"duration_min"`
	Status        ProductStatus `gorm:"column:status;not null;default:0" json:"status"`
}

func (Product) TableName() string { return "products" }
