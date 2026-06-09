package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"auction/internal/model"
	"auction/internal/repository"
	redisPkg "auction/pkg/redis"
)

type LiveRoomSvc struct {
	repo *repository.LiveRoomRepo
	rdb  *redisPkg.Client
}

func NewLiveRoomSvc(repo *repository.LiveRoomRepo, rdb *redisPkg.Client) *LiveRoomSvc {
	return &LiveRoomSvc{repo: repo, rdb: rdb}
}

func (s *LiveRoomSvc) Create(ctx context.Context, room *model.LiveRoom) error {
	return s.repo.Create(ctx, room)
}

func (s *LiveRoomSvc) GetByID(ctx context.Context, id uint64) (*model.LiveRoom, error) {
	room, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Enrich with real-time Redis online count
	if room.Status == model.LiveRoomStatusLive {
		key := fmt.Sprintf("auction:%d:viewers", id)
		if count, err := s.rdb.SCard(ctx, key).Result(); err == nil {
			room.OnlineCount = uint(count)
		}
	}
	return room, nil
}

func (s *LiveRoomSvc) Update(ctx context.Context, id uint64, updates map[string]any) error {
	return s.repo.Update(ctx, id, updates)
}

func (s *LiveRoomSvc) List(ctx context.Context, page model.PageRequest) ([]model.LiveRoom, int64, error) {
	return s.repo.List(ctx, page)
}

func (s *LiveRoomSvc) ListLive(ctx context.Context, page model.PageRequest) ([]model.LiveRoom, int64, error) {
	return s.repo.ListLive(ctx, page)
}

func (s *LiveRoomSvc) ListBySeller(ctx context.Context, sellerID uint64, page model.PageRequest) ([]model.LiveRoom, int64, error) {
	return s.repo.ListBySeller(ctx, sellerID, page)
}

func (s *LiveRoomSvc) ListAllLive(ctx context.Context) ([]model.LiveRoom, error) {
	return s.repo.ListAllLive(ctx)
}

func (s *LiveRoomSvc) CountBySellerID(ctx context.Context, sellerID uint64) (int64, error) {
	return s.repo.CountBySellerID(ctx, sellerID)
}

const searchCachePrefix = "search:live:"
const searchCacheTTL = 10 * time.Second

func (s *LiveRoomSvc) SearchLive(ctx context.Context, keyword string) ([]model.LiveRoom, error) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return s.repo.ListAllLive(ctx)
	}

	cacheKey := fmt.Sprintf("%s%s", searchCachePrefix, kw)

	// 1. Try Redis cache
	if cached, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
		var rooms []model.LiveRoom
		if json.Unmarshal([]byte(cached), &rooms) == nil {
			return rooms, nil
		}
	}

	// 2. Fallback to MySQL
	rooms, err := s.repo.SearchLive(ctx, kw)
	if err != nil {
		return nil, err
	}

	// 3. Write cache (async, don't block response)
	go func() {
		if data, err := json.Marshal(rooms); err == nil {
			s.rdb.Set(context.Background(), cacheKey, data, searchCacheTTL)
		}
	}()

	return rooms, nil
}
