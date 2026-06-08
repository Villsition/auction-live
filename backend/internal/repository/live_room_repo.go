package repository

import (
	"context"

	"auction/internal/model"

	"gorm.io/gorm"
)

type LiveRoomRepo struct {
	*BaseRepo[model.LiveRoom]
}

func NewLiveRoomRepo(db *gorm.DB) *LiveRoomRepo {
	return &LiveRoomRepo{BaseRepo: NewBaseRepo[model.LiveRoom](db)}
}

// GetByID joins the users table to populate seller nickname and avatar.
func (r *LiveRoomRepo) GetByID(ctx context.Context, id uint64) (*model.LiveRoom, error) {
	type row struct {
		model.LiveRoom
		Nickname string `gorm:"column:nickname"`
		Avatar   string `gorm:"column:avatar"`
	}
	var result row
	err := r.DB.WithContext(ctx).Table("live_rooms").
		Select("live_rooms.*, users.nickname, users.avatar").
		Joins("LEFT JOIN users ON users.id = live_rooms.seller_id").
		Where("live_rooms.id = ?", id).First(&result).Error
	if err != nil {
		return nil, err
	}
	result.LiveRoom.SellerNickname = result.Nickname
	result.LiveRoom.SellerAvatar = result.Avatar
	return &result.LiveRoom, nil
}

func (r *LiveRoomRepo) ListLive(ctx context.Context, page model.PageRequest) ([]model.LiveRoom, int64, error) {
	var total int64
	r.DB.WithContext(ctx).Model(&model.LiveRoom{}).Where("status = ?", model.LiveRoomStatusLive).Count(&total)

	type row struct {
		model.LiveRoom
		Nickname string `gorm:"column:nickname"`
		Avatar   string `gorm:"column:avatar"`
	}
	var rows []row
	err := r.DB.WithContext(ctx).Table("live_rooms").
		Select("live_rooms.*, users.nickname, users.avatar").
		Joins("LEFT JOIN users ON users.id = live_rooms.seller_id").
		Where("live_rooms.status = ?", model.LiveRoomStatusLive).
		Order("live_rooms.online_count DESC").
		Offset(page.Offset()).Limit(page.PageSize).Find(&rows).Error

	rooms := make([]model.LiveRoom, len(rows))
	for i, r := range rows {
		r.LiveRoom.SellerNickname = r.Nickname
		r.LiveRoom.SellerAvatar = r.Avatar
		rooms[i] = r.LiveRoom
	}
	return rooms, total, err
}

func (r *LiveRoomRepo) ListBySeller(ctx context.Context, sellerID uint64, page model.PageRequest) ([]model.LiveRoom, int64, error) {
	var rooms []model.LiveRoom
	var total int64

	db := r.DB.WithContext(ctx).Model(&model.LiveRoom{}).Where("seller_id = ?", sellerID)
	db.Count(&total)

	err := db.Offset(page.Offset()).Limit(page.PageSize).Order("created_at DESC").Find(&rooms).Error
	return rooms, total, err
}

// CountBySellerID returns the number of live rooms owned by a seller (any status).
func (r *LiveRoomRepo) CountBySellerID(ctx context.Context, sellerID uint64) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).Model(&model.LiveRoom{}).Where("seller_id = ?", sellerID).Count(&count).Error
	return count, err
}

// SearchLive returns live rooms whose title contains keyword.
func (r *LiveRoomRepo) SearchLive(ctx context.Context, keyword string) ([]model.LiveRoom, error) {
	type row struct {
		model.LiveRoom
		Nickname string `gorm:"column:nickname"`
		Avatar   string `gorm:"column:avatar"`
	}
	var rows []row
	err := r.DB.WithContext(ctx).Table("live_rooms").
		Select("live_rooms.*, users.nickname, users.avatar").
		Joins("LEFT JOIN users ON users.id = live_rooms.seller_id").
		Where("live_rooms.status = ? AND (live_rooms.title LIKE ? OR users.nickname LIKE ?)", model.LiveRoomStatusLive, "%"+keyword+"%", "%"+keyword+"%").
		Order("live_rooms.online_count DESC").Limit(50).Find(&rows).Error

	rooms := make([]model.LiveRoom, len(rows))
	for i, r := range rows {
		r.LiveRoom.SellerNickname = r.Nickname
		r.LiveRoom.SellerAvatar = r.Avatar
		rooms[i] = r.LiveRoom
	}
	return rooms, err
}

// ListAllLive returns all live rooms with seller info.
func (r *LiveRoomRepo) ListAllLive(ctx context.Context) ([]model.LiveRoom, error) {
	type row struct {
		model.LiveRoom
		Nickname string `gorm:"column:nickname"`
		Avatar   string `gorm:"column:avatar"`
	}
	var rows []row
	err := r.DB.WithContext(ctx).Table("live_rooms").
		Select("live_rooms.*, users.nickname, users.avatar").
		Joins("LEFT JOIN users ON users.id = live_rooms.seller_id").
		Where("live_rooms.status = ?", model.LiveRoomStatusLive).
		Order("live_rooms.online_count DESC").Limit(50).Find(&rows).Error

	rooms := make([]model.LiveRoom, len(rows))
	for i, r := range rows {
		r.LiveRoom.SellerNickname = r.Nickname
		r.LiveRoom.SellerAvatar = r.Avatar
		rooms[i] = r.LiveRoom
	}
	return rooms, err
}
