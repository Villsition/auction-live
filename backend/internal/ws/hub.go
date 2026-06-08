package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	redisPkg "auction/pkg/redis"

	"go.uber.org/zap"
)

const viewersKey = "auction:%d:viewers"

type Hub struct {
	rooms       map[uint64]map[*Client]struct{}
	auctionRoom map[uint64]uint64 // auctionID → roomID for room-level broadcast isolation
	userRoom    map[uint64]uint64 // userID → roomID for O(1) per-user send
	register    chan *Client
	unregister  chan *Client
	rdb         *redisPkg.Client // write
	rdbRead     *redisPkg.Client // read
	logger      *zap.Logger
	mu          sync.RWMutex
}

func NewHub(rdb, rdbRead *redisPkg.Client, logger *zap.Logger) *Hub {
	return &Hub{
		rooms:       make(map[uint64]map[*Client]struct{}),
		auctionRoom: make(map[uint64]uint64),
		userRoom:    make(map[uint64]uint64),
		register:    make(chan *Client, 512),
		unregister:  make(chan *Client, 512),
		rdb:         rdb,
		rdbRead:     rdbRead,
		logger:      logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleJoin(client)
		case client := <-h.unregister:
			h.handleLeave(client)
		}
	}
}

func (h *Hub) handleJoin(client *Client) {
	ctx := context.Background()
	key := fmt.Sprintf(viewersKey, client.roomID)

	h.mu.Lock()
	if h.rooms[client.roomID] == nil {
		h.rooms[client.roomID] = make(map[*Client]struct{})
	}
	h.rooms[client.roomID][client] = struct{}{}
	h.userRoom[client.userID] = client.roomID
	h.mu.Unlock()

	h.rdb.SAdd(ctx, key, client.userID)
	count, _ := h.rdb.SCard(ctx, key).Result()

	h.logger.Debug("ws client joined",
		zap.Uint64("room", client.roomID),
		zap.Uint64("user", client.userID),
		zap.Int64("online", count),
	)

	h.broadcastOnlineCount(client.roomID, int(count))
}

func (h *Hub) handleLeave(client *Client) {
	if client.dropped > 0 {
		h.logger.Warn("ws client disconnected with dropped messages",
			zap.Uint64("user", client.userID),
			zap.Uint64("room", client.roomID),
			zap.Int64("total_dropped", client.dropped),
		)
	}
	ctx := context.Background()
	key := fmt.Sprintf(viewersKey, client.roomID)

	h.mu.Lock()
	if clients, ok := h.rooms[client.roomID]; ok {
		delete(clients, client)
		close(client.send)
		if len(clients) == 0 {
			delete(h.rooms, client.roomID)
		}
	}
	// Only clear userRoom if no other connection from this user exists anywhere
	stillConnected := false
	for _, clients := range h.rooms {
		for c := range clients {
			if c.userID == client.userID && c != client {
				stillConnected = true
				break
			}
		}
		if stillConnected { break }
	}
	if !stillConnected {
		delete(h.userRoom, client.userID)
	}
	h.mu.Unlock()

	h.rdb.SRem(ctx, key, client.userID)
	count, _ := h.rdb.SCard(ctx, key).Result()

	h.logger.Debug("ws client left",
		zap.Uint64("room", client.roomID),
		zap.Uint64("user", client.userID),
		zap.Int64("online", count),
	)

	h.broadcastOnlineCount(client.roomID, int(count))
}

func (h *Hub) broadcastOnlineCount(roomID uint64, count int) {
	// Collect up to 3 viewers for the avatar display
	h.mu.RLock()
	clients := h.rooms[roomID]
	// Deduplicate by userID
	seen := make(map[uint64]bool)
	viewers := make([]ViewerInfo, 0, 3)
	for c := range clients {
		if seen[c.userID] || len(viewers) >= 3 {
			continue
		}
		seen[c.userID] = true
		viewers = append(viewers, ViewerInfo{
			UserID:   c.userID,
			Nickname: c.nickname,
			Avatar:   c.avatar,
		})
	}
	h.mu.RUnlock()

	h.BroadcastToRoom(roomID, &OnlineCountEvent{
		Type:    EventOnlineCount,
		RoomID:  roomID,
		Count:   count,
		Viewers: viewers,
	})
}

func (h *Hub) BroadcastToRoom(roomID uint64, msg any) {
	h.mu.RLock()
	clients := h.rooms[roomID]
	h.mu.RUnlock()

	// Marshal once, send bytes to all clients (avoids per-client JSON encode)
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	for client := range clients {
		client.SendBytes(data)
	}
}

// SetAuctionRoom records the mapping from auctionID to roomID for room-level broadcast isolation.
func (h *Hub) SetAuctionRoom(auctionID, roomID uint64) {
	h.mu.Lock()
	h.auctionRoom[auctionID] = roomID
	h.mu.Unlock()
}

// RemoveAuctionRoom removes the auction→room mapping (call when auction ends/cancels).
func (h *Hub) RemoveAuctionRoom(auctionID uint64) {
	h.mu.Lock()
	delete(h.auctionRoom, auctionID)
	h.mu.Unlock()
}

func (h *Hub) BroadcastToAuction(auctionID uint64, msg any) {
	h.mu.RLock()
	roomID, ok := h.auctionRoom[auctionID]
	if ok {
		clients := h.rooms[roomID]
		// Marshal and collect send channels while holding lock (avoids concurrent map read-write)
		data, err := json.Marshal(msg)
		if err != nil {
			h.mu.RUnlock()
			h.logger.Error("marshal broadcast", zap.Error(err))
			return
		}
		sends := make([]chan<- []byte, 0, len(clients))
		for c := range clients {
			sends = append(sends, c.send)
		}
		h.mu.RUnlock()

		for _, ch := range sends {
			select {
			case ch <- data:
			default:
			}
		}
		return
	}
	h.mu.RUnlock()

	// Fallback: no mapping found — broadcast to all rooms
	h.mu.RLock()
	defer h.mu.RUnlock()
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("marshal broadcast", zap.Error(err))
		return
	}
	for _, clients := range h.rooms {
		for c := range clients {
			select {
			case c.send <- data:
			default:
				c.dropped++
			}
		}
	}
}

// SendToUser sends a message to a specific user via O(1) user→room lookup.
// The roomID parameter is retained for backward compatibility but is ignored
// in favor of the userRoom index.
// Returns true if the user was found.
func (h *Hub) SendToUser(roomID, userID uint64, msg any) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// O(1): look up user's room directly
	actualRoomID, ok := h.userRoom[userID]
	if !ok {
		return false
	}
	_ = roomID // retained for API compatibility; userRoom is authoritative

	clients := h.rooms[actualRoomID]
	for client := range clients {
		if client.userID == userID {
			client.SendJSON(msg)
			return true
		}
	}
	return false
}

func (h *Hub) GetOnlineCount(ctx context.Context, roomID uint64) (int64, error) {
	return h.rdbRead.SCard(ctx, fmt.Sprintf(viewersKey, roomID)).Result()
}

func (h *Hub) RoomOnlineCount(roomID uint64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}
