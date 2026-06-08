package model

import "time"

type LiveRoom struct {
	BaseModel
	SellerID       uint64         `gorm:"column:seller_id;not null" json:"seller_id"`
	Title          string         `gorm:"column:title;type:varchar(256);not null;default:''" json:"title"`
	CoverImage     string         `gorm:"column:cover_image;type:varchar(512);not null;default:''" json:"cover_image"`
	StreamURL      string         `gorm:"column:stream_url;type:varchar(512);not null;default:''" json:"stream_url"`
	Status         LiveRoomStatus `gorm:"column:status;not null;default:0" json:"status"`
	OnlineCount    uint           `gorm:"column:online_count;not null;default:0" json:"online_count"`
	TotalLikes     uint64         `gorm:"column:total_likes;not null;default:0" json:"total_likes"`
	StartedAt      *time.Time     `gorm:"column:started_at" json:"started_at"`
	EndedAt        *time.Time     `gorm:"column:ended_at" json:"ended_at"`
	// Transient (joined from users)
	SellerNickname string `gorm:"-" json:"seller_nickname,omitempty"`
	SellerAvatar   string `gorm:"-" json:"seller_avatar,omitempty"`
}

func (LiveRoom) TableName() string { return "live_rooms" }
