package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"auction/internal/model"
	"auction/internal/service"
	"auction/internal/ws"
	redisPkg "auction/pkg/redis"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type LiveRoomHandler struct {
	svc               *service.LiveRoomSvc
	rdb               *redisPkg.Client
	commentSvc        *service.CommentSvc
	auctionSessionSvc *service.AuctionSessionSvc
	hub               *ws.Hub
}

func NewLiveRoomHandler(svc *service.LiveRoomSvc, rdb *redisPkg.Client, commentSvc *service.CommentSvc, auctionSessionSvc *service.AuctionSessionSvc, hub *ws.Hub) *LiveRoomHandler {
	return &LiveRoomHandler{svc: svc, rdb: rdb, commentSvc: commentSvc, auctionSessionSvc: auctionSessionSvc, hub: hub}
}

// ---- public ----

func (h *LiveRoomHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}
	room, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "room not found")
		return
	}
	response.Success(c, room)
}

func (h *LiveRoomHandler) List(c *gin.Context) {
	keyword := c.Query("keyword")
	var rooms []model.LiveRoom
	var err error
	if keyword != "" {
		rooms, err = h.svc.SearchLive(c.Request.Context(), keyword)
	} else {
		rooms, err = h.svc.ListAllLive(c.Request.Context())
	}
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}

	// Enrich with real-time online count from Redis
	for i := range rooms {
		if rooms[i].Status == model.LiveRoomStatusLive {
			key := fmt.Sprintf("auction:%d:viewers", rooms[i].ID)
			if count, err := h.rdb.SCard(c.Request.Context(), key).Result(); err == nil {
				rooms[i].OnlineCount = uint(count)
			}
		}
	}

	response.Success(c, gin.H{"list": rooms, "total": len(rooms)})
}

// ---- seller ----

type createRoomReq struct {
	Title      string `json:"title" binding:"required"`
	CoverImage string `json:"cover_image"`
	StreamURL  string `json:"stream_url"`
}

func (h *LiveRoomHandler) Create(c *gin.Context) {
	var req createRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}

	sellerID, _ := c.Get("user_id")

	// One room per seller
	count, err := h.svc.CountBySellerID(c.Request.Context(), sellerID.(uint64))
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	if count > 0 {
		response.Error(c, errcode.ErrConflict, "each seller can only have one live room")
		return
	}

	room := model.LiveRoom{
		SellerID:   sellerID.(uint64),
		Title:      req.Title,
		CoverImage: req.CoverImage,
		StreamURL:  req.StreamURL,
		Status:     model.LiveRoomStatusOffline,
	}

	if err := h.svc.Create(c.Request.Context(), &room); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, room)
}

func (h *LiveRoomHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}
	var updates map[string]any
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	// Prevent tampering with seller_id or online_count
	delete(updates, "seller_id")
	delete(updates, "online_count")

	if err := h.svc.Update(c.Request.Context(), id, updates); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *LiveRoomHandler) ListMyRooms(c *gin.Context) {
	sellerID, _ := c.Get("user_id")
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	rooms, total, err := h.svc.ListBySeller(c.Request.Context(), sellerID.(uint64), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.LiveRoom]{
		List: rooms, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

func (h *LiveRoomHandler) StartLive(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}

	room, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "room not found")
		return
	}
	if room.Status != model.LiveRoomStatusOffline && room.Status != model.LiveRoomStatusEnded {
		response.Error(c, errcode.ErrConflict, "room is not offline or ended")
		return
	}

	now := time.Now()
	updates := map[string]any{
		"status":     model.LiveRoomStatusLive,
		"started_at": now,
	}
	if err := h.svc.Update(c.Request.Context(), id, updates); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}

	room.Status = model.LiveRoomStatusLive
	room.StartedAt = &now
	response.Success(c, room)
}

func (h *LiveRoomHandler) EndLive(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}

	room, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "room not found")
		return
	}
	if room.Status != model.LiveRoomStatusLive {
		response.Error(c, errcode.ErrConflict, "room is not live")
		return
	}

	now := time.Now()
	// Clear comments from this session
	_ = h.commentSvc.ClearRoom(context.Background(), id)
	// Cancel all auction sessions for this room
	_ = h.auctionSessionSvc.CancelByRoom(context.Background(), id)
	// Save likes from Redis to DB, then clear Redis key
	likesKey := fmt.Sprintf("room:%d:likes", id)
	likes, _ := h.rdb.Get(context.Background(), likesKey).Uint64()
	h.rdb.Del(context.Background(), likesKey)
	updates := map[string]any{
		"status":      model.LiveRoomStatusOffline,
		"ended_at":    now,
		"started_at":  nil,
		"total_likes": likes,
	}
	if err := h.svc.Update(c.Request.Context(), id, updates); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}

	room.Status = model.LiveRoomStatusOffline
	room.EndedAt = &now
	room.StartedAt = nil
	room.TotalLikes = likes

	// Broadcast live_end to all viewers
	h.hub.BroadcastToRoom(id, map[string]any{
		"type":    ws.EventLiveEnd,
		"room_id": id,
	})
	response.Success(c, room)
}
