package ws

const (
	EventBid           = "bid"
	EventDelayExtend   = "delay_extend"
	EventCeilingDeal   = "ceiling_deal"
	EventAuctionEnd    = "auction_end"
	EventAuctionStart  = "auction_start"
	EventAuctionCancel = "auction_cancel"
	EventOnlineCount   = "online_count"
	EventOutbid        = "outbid"      // personal: you've been outbid
	EventNewLeader     = "new_leader"  // room: top bidder changed
	EventLike          = "like"        // someone liked the streamer
	EventLiveEnd       = "live_end"    // streamer ended the live
)

type BidEvent struct {
	Type        string `json:"type"`
	AuctionID   uint64 `json:"auction_id"`
	Amount      string `json:"amount"`
	UserID      uint64 `json:"user_id,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	Rank        int64  `json:"rank"`
	BidCount    int64  `json:"bid_count"`
	CeilingDeal bool  `json:"ceiling_deal"`
	DelayExtend  bool  `json:"delay_extend"`
	FinalDelay   bool  `json:"final_delay"`                 // true on 5th (last) extension
	NewEndTime   int64 `json:"new_end_time_ms,omitempty"`   // Unix ms
	ServerTimeMs int64 `json:"server_time_ms,omitempty"`    // Server Unix ms at bid time
}

type AuctionEvent struct {
	Type       string `json:"type"`
	AuctionID  uint64 `json:"auction_id"`
	Status     string `json:"status"`
	WinnerID     uint64 `json:"winner_id,omitempty"`
	WinnerName   string `json:"winner_name,omitempty"`
	WinnerAvatar string `json:"winner_avatar,omitempty"`
	FinalPrice   string `json:"final_price,omitempty"`
	Message    string `json:"message"`
}

type OnlineCountEvent struct {
	Type     string       `json:"type"`
	RoomID   uint64       `json:"room_id"`
	Count    int          `json:"count"`
	Viewers  []ViewerInfo `json:"viewers"` // current top viewers (up to 3)
}

type ViewerInfo struct {
	UserID   uint64 `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// OutbidEvent is sent when a user loses the top spot.
type OutbidEvent struct {
	Type      string `json:"type"`
	AuctionID uint64 `json:"auction_id"`
	UserID    uint64 `json:"user_id"`    // who was outbid
	NewAmount string `json:"new_amount"` // the new highest price
	MyRank    int64  `json:"my_rank"`    // their new rank
}

// NewLeaderEvent is broadcast when the top bidder changes.
type NewLeaderEvent struct {
	Type        string `json:"type"`
	AuctionID   uint64 `json:"auction_id"`
	OldLeaderID uint64 `json:"old_leader_id,omitempty"` // 0 if first bid
	NewLeaderID uint64 `json:"new_leader_id"`
	Amount      string `json:"amount"`
	Message     string `json:"message"` // "xxx成为新榜首！"
}
