package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

// Client represents a single WebSocket connection.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   uint64
	roomID   uint64 // the room/auction this client is watching
	nickname string
	avatar   string
	logger   *zap.Logger
	mu       sync.Mutex
	dropped  int64 // count of messages dropped due to full buffer
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, roomID uint64, nickname, avatar string, logger *zap.Logger) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 1024),
		userID:   userID,
		roomID:   roomID,
		nickname: nickname,
		avatar:   avatar,
		logger:   logger,
	}
}

func (c *Client) UserID() uint64  { return c.userID }
func (c *Client) RoomID() uint64  { return c.roomID }

// ReadPump reads messages from the WebSocket (ping/pong/close only).
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Debug("ws client disconnected", zap.Error(err))
			}
			break
		}
	}
}

// WritePump writes messages from the send channel to the WebSocket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendBytes enqueues pre-serialized bytes to the client's send channel.
// Uses a blocking send with 2s timeout to avoid data loss under burst traffic.
func (c *Client) SendBytes(data []byte) {
	select {
	case c.send <- data:
	case <-time.After(2 * time.Second):
		c.dropped++
		c.logger.Warn("ws send buffer full, dropping message",
			zap.Uint64("user", c.userID),
			zap.Uint64("room", c.roomID),
			zap.Int64("total_dropped", c.dropped),
		)
	}
}

// SendJSON marshals and enqueues a message to the client's send channel.
// Prefer SendBytes when broadcasting to multiple clients.
func (c *Client) SendJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.SendBytes(data)
}
