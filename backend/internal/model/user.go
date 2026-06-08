package model

import "time"

type User struct {
	BaseModel
	Username     string     `gorm:"column:username;uniqueIndex:uk_username;type:varchar(32);not null;default:''" json:"username"`
	PasswordHash string     `gorm:"column:password_hash;type:varchar(256);not null;default:''" json:"-"`
	Nickname     string     `gorm:"column:nickname;type:varchar(64);not null;default:''" json:"nickname"`
	Avatar       string     `gorm:"column:avatar;type:varchar(512);not null;default:''" json:"avatar"`
	Role         UserRole   `gorm:"column:role;not null;default:0" json:"role"`
	Status       UserStatus `gorm:"column:status;not null;default:1" json:"status"`
	TokenVersion int64      `gorm:"column:token_version;not null;default:0" json:"token_version"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
}

func (User) TableName() string { return "users" }
