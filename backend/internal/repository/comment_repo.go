package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"auction/internal/model"

	redisPkg "auction/pkg/redis"

	"gorm.io/gorm"
)

const commentListKey = "room:%d:comments"
const maxCachedComments = 200

type CommentRepo struct {
	db      *gorm.DB
	rdb     *redisPkg.Client
	rdbRead *redisPkg.Client
}

func NewCommentRepo(db *gorm.DB, rdb, rdbRead *redisPkg.Client) *CommentRepo {
	return &CommentRepo{db: db, rdb: rdb, rdbRead: rdbRead}
}

// PushToCache LPUSHes a comment JSON into Redis list and trims to max size.
func (r *CommentRepo) PushToCache(ctx context.Context, c *model.Comment) error {
	key := fmt.Sprintf(commentListKey, c.RoomID)

	data, _ := json.Marshal(map[string]any{
		"id":       c.ID,
		"user_id":  c.UserID,
		"username": c.Username,
		"content":  c.Content,
		"time":     time.Now().Format("2006-01-02 15:04:05"),
	})

	pipe := r.rdb.Pipeline()
	pipe.LPush(ctx, key, string(data))
	pipe.LTrim(ctx, key, 0, maxCachedComments-1)
	_, err := pipe.Exec(ctx)
	return err
}

// ListFromCache returns recent comments from Redis. Newest last (chronological).
func (r *CommentRepo) ListFromCache(ctx context.Context, roomID uint64, limit int) ([]model.Comment, error) {
	key := fmt.Sprintf(commentListKey, roomID)
	results, err := r.rdbRead.LRange(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}

	comments := make([]model.Comment, 0, len(results))
	// LRange returns newest-first, reverse to chronological
	for i := len(results) - 1; i >= 0; i-- {
		var m map[string]any
		if json.Unmarshal([]byte(results[i]), &m) != nil {
			continue
		}
		c := model.Comment{
			Content: str(m, "content"),
			Username: str(m, "username"),
		}
		if uid, ok := m["user_id"].(float64); ok {
			c.UserID = uint64(uid)
		}
		comments = append(comments, c)
	}
	return comments, nil
}

// Create persists a comment to MySQL.
func (r *CommentRepo) Create(ctx context.Context, c *model.Comment) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// DeleteByRoom clears all comments for a room from MySQL and Redis cache.
func (r *CommentRepo) DeleteByRoom(ctx context.Context, roomID uint64) error {
	// Clear Redis cache
	key := fmt.Sprintf(commentListKey, roomID)
	r.rdb.Del(ctx, key)
	// Clear MySQL
	return r.db.WithContext(ctx).Where("room_id = ?", roomID).Delete(&model.Comment{}).Error
}

// ListByRoom reads from MySQL, newest last.
func (r *CommentRepo) ListByRoom(ctx context.Context, roomID uint64, limit int) ([]model.Comment, error) {
	var comments []model.Comment
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("created_at DESC").
		Limit(limit).
		Find(&comments).Error
	for i, j := 0, len(comments)-1; i < j; i, j = i+1, j-1 {
		comments[i], comments[j] = comments[j], comments[i]
	}
	return comments, err
}

func str(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
