package service

import (
	"context"
	"time"

	"auction/internal/model"
	"auction/internal/repository"
	"auction/internal/ws"
)

type CommentSvc struct {
	repo     *repository.CommentRepo
	userRepo *repository.UserRepo
	hub      *ws.Hub
}

func NewCommentSvc(repo *repository.CommentRepo, userRepo *repository.UserRepo, hub *ws.Hub) *CommentSvc {
	return &CommentSvc{repo: repo, userRepo: userRepo, hub: hub}
}

// Create pushes comment to Redis cache first, broadcasts via WebSocket,
// then asynchronously persists to MySQL.
func (s *CommentSvc) Create(ctx context.Context, roomID, userID uint64, content string) (*model.Comment, error) {
	user, _ := s.userRepo.GetByID(ctx, userID)
	username := ""
	if user != nil {
		username = user.Nickname
	}

	c := &model.Comment{
		RoomID:   roomID,
		UserID:   userID,
		Content:  content,
		Username: username,
	}
	// Generate temporary ID for dedup before MySQL persist
	c.ID = uint64(time.Now().UnixNano())

	// 1. Write to Redis cache (fast path)
	if err := s.repo.PushToCache(ctx, c); err != nil {
		return nil, err
	}

	// 2. Broadcast via WebSocket immediately (room-level isolation)
	s.hub.BroadcastToRoom(roomID, map[string]any{
		"type":     "comment",
		"id":       c.ID,
		"room_id":  c.RoomID,
		"user_id":  c.UserID,
		"username": c.Username,
		"content":  c.Content,
	})

	// 3. Async persist to MySQL
	go func() {
		_ = s.repo.Create(context.Background(), c)
	}()

	return c, nil
}

// ClearRoom deletes all comments for a room.
func (s *CommentSvc) ClearRoom(ctx context.Context, roomID uint64) error {
	return s.repo.DeleteByRoom(ctx, roomID)
}

// ListByRoom reads from Redis cache first, falls back to MySQL.
func (s *CommentSvc) ListByRoom(ctx context.Context, roomID uint64, limit int) ([]model.Comment, error) {
	comments, err := s.repo.ListFromCache(ctx, roomID, limit)
	if err != nil || len(comments) == 0 {
		comments, err = s.repo.ListByRoom(ctx, roomID, limit)
		if err != nil {
			return nil, err
		}
		// Fill usernames for MySQL results
		userIDs := make(map[uint64]bool)
		for _, c := range comments {
			userIDs[c.UserID] = true
		}
		users := make(map[uint64]string)
		for uid := range userIDs {
			if u, err := s.userRepo.GetByID(ctx, uid); err == nil {
				users[uid] = u.Nickname
			}
		}
		for i := range comments {
			if name, ok := users[comments[i].UserID]; ok {
				comments[i].Username = name
			}
		}
	}
	return comments, nil
}
