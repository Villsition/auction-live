package handler

import (
	"net/http"
	"strconv"

	"auction/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	gorillaWs "github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = gorillaWs.Upgrader{
	ReadBufferSize:     1024,
	WriteBufferSize:    1024,
	EnableCompression:  true,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in dev
	},
}

type WSHandler struct {
	hub       *ws.Hub
	jwtSecret string
	logger    *zap.Logger
}

func NewWSHandler(hub *ws.Hub, jwtSecret string, logger *zap.Logger) *WSHandler {
	return &WSHandler{hub: hub, jwtSecret: jwtSecret, logger: logger}
}

// Connect handles WebSocket upgrade and authentication.
// URL: /api/ws?token=xxx&room_id=123
// The room_id corresponds to the auction_id the client wants to watch.
func (h *WSHandler) Connect(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "token required"})
		return
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
		return
	}

	userID := uint64(claims["user_id"].(float64))
	nickname, _ := claims["nickname"].(string)
	avatar, _ := claims["avatar"].(string)

	roomID, err := strconv.ParseUint(c.Query("room_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "room_id required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("ws upgrade failed", zap.Error(err))
		return
	}

	client := ws.NewClient(h.hub, conn, userID, roomID, nickname, avatar, h.logger)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
