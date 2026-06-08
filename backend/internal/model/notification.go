package model

type Notification struct {
	BaseModel
	UserID    uint64           `gorm:"column:user_id;not null" json:"user_id"`
	Title     string           `gorm:"column:title;type:varchar(256);not null;default:''" json:"title"`
	Content   string           `gorm:"column:content;type:text" json:"content"`
	Type      NotificationType `gorm:"column:type;not null;default:0" json:"type"`
	RelatedID uint64           `gorm:"column:related_id;not null;default:0" json:"related_id"`
	IsRead    uint8            `gorm:"column:is_read;not null;default:0" json:"is_read"`
}

func (Notification) TableName() string { return "notifications" }
