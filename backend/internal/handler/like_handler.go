package handler

import (
	"context"
	"fmt"
	"strconv"

	redisPkg "auction/pkg/redis"
	"auction/internal/ws"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

const likesKey = "room:%d:likes"

type LikeHandler struct {
	rdb *redisPkg.Client
	hub *ws.Hub
}

func NewLikeHandler(rdb *redisPkg.Client, hub *ws.Hub) *LikeHandler {
	return &LikeHandler{rdb: rdb, hub: hub}
}

// Send increments the like counter and broadcasts to the room.
func (h *LikeHandler) Send(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf(likesKey, roomID)
	total, _ := h.rdb.Incr(ctx, key).Result()

	userID, _ := c.Get("user_id")

	h.hub.BroadcastToRoom(roomID, map[string]any{
		"type":    ws.EventLike,
		"room_id": roomID,
		"user_id": userID,
		"total":   total,
	})

	response.Success(c, gin.H{"room_id": roomID, "total": total})
}

// Total returns the current like count.
func (h *LikeHandler) Total(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}

	ctx := context.Background()
	val, _ := h.rdb.Get(ctx, fmt.Sprintf(likesKey, roomID)).Int64()

	response.Success(c, gin.H{"room_id": roomID, "total": val})
}
