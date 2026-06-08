package model

type Comment struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	RoomID    uint64 `gorm:"column:room_id;not null" json:"room_id"`
	UserID    uint64 `gorm:"column:user_id;not null" json:"user_id"`
	Content   string `gorm:"column:content;type:varchar(500);not null" json:"content"`
	CreatedAt string `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// Joined fields (not stored)
	Username string `gorm:"-" json:"username,omitempty"`
}

func (Comment) TableName() string { return "comments" }
