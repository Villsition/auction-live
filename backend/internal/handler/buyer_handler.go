package handler

import (
	"encoding/json"
	"context"
	"fmt"
	"strconv"
	"time"

	"auction/internal/model"
	"auction/internal/repository"
	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BuyerHandler struct {
	userRepo          *repository.UserRepo
	liveRoomSvc       *service.LiveRoomSvc
	auctionSessionSvc *service.AuctionSessionSvc
	bidSvc            *service.BidSvc
	productSvc        *service.ProductSvc
	db                *gorm.DB
}

func NewBuyerHandler(
	userRepo *repository.UserRepo,
	liveRoomSvc *service.LiveRoomSvc,
	auctionSessionSvc *service.AuctionSessionSvc,
	bidSvc *service.BidSvc,
	productSvc *service.ProductSvc,
	db *gorm.DB,
) *BuyerHandler {
	return &BuyerHandler{
		userRepo:          userRepo,
		liveRoomSvc:       liveRoomSvc,
		auctionSessionSvc: auctionSessionSvc,
		bidSvc:            bidSvc,
		productSvc:        productSvc,
		db:                db,
	}
}

// roomAuctionResp is the aggregated response for GET /api/live-rooms/:id/auction
type roomAuctionResp struct {
	LiveRoom       *model.LiveRoom       `json:"live_room"`
	AuctionSession *model.AuctionSession `json:"auction_session"`
	Product        *model.Product        `json:"product"`
	CurrentPrice   string                `json:"current_price"`
	BidCount       int64                 `json:"bid_count"`
	EndTimestamp   int64                 `json:"end_timestamp_ms"` // planned end, Unix ms
	ServerTime     int64                 `json:"server_time_ms"`   // current server Unix ms
	RemainingMs    int64                 `json:"remaining_ms"`     // ms left, 0 if ended
	NextBid        string                `json:"next_bid"`         // suggested next bid amount
}

// GetCurrentAuction returns the active auction + product + real-time data for a live room.
func (h *BuyerHandler) GetCurrentAuction(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}

	room, err := h.liveRoomSvc.GetByID(c.Request.Context(), roomID)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "room not found")
		return
	}

	// Scope to current live session
	var sessions []model.AuctionSession
	if room.StartedAt != nil {
		sessions, err = h.auctionSessionSvc.ListByRoomSince(c.Request.Context(), roomID, *room.StartedAt)
	} else {
		sessions, _, err = h.auctionSessionSvc.ListByRoom(c.Request.Context(), roomID, model.PageRequest{Page: 1, PageSize: 500})
	}
	if err != nil || len(sessions) == 0 {
		response.Success(c, gin.H{"live_room": room, "auction_session": nil})
		return
	}
	// Priority: active (any) → highest auction_id (most recent)
	var session model.AuctionSession
	var bestActive model.AuctionSession
	var bestAny model.AuctionSession
	for _, s := range sessions {
		if s.ID > bestAny.ID { bestAny = s }
		if s.Status == model.AuctionStatusActive && s.ID > bestActive.ID {
			bestActive = s
		}
	}
	if bestActive.ID != 0 {
		session = bestActive
	} else {
		session = bestAny
	}
	if session.Status != model.AuctionStatusActive {
		product, _ := h.productSvc.GetByID(c.Request.Context(), session.ProductID)
		if product == nil && session.Product != nil {
			product = session.Product
		}
		var endTs int64
		if session.PlannedEndTime != nil {
			endTs = session.PlannedEndTime.UnixMilli()
		}
		response.Success(c, roomAuctionResp{
			LiveRoom:       room,
			AuctionSession: &session,
			Product:        product,
			CurrentPrice:   session.CurrentPrice,
			EndTimestamp:   endTs,
			ServerTime:     time.Now().UnixMilli(),
			NextBid:        session.CurrentPrice,
		})
		return
	}

	product, _ := h.productSvc.GetByID(c.Request.Context(), session.ProductID)
	// Fallback to embedded product if preloaded
	if product == nil && session.Product != nil {
		product = session.Product
	}

	currentPrice, _ := h.bidSvc.GetCurrentPrice(c.Request.Context(), session.ID)
	if currentPrice == "" {
		currentPrice = session.CurrentPrice
	}
	bidCount, _ := h.bidSvc.GetBidCount(c.Request.Context(), session.ID)

	// Calculate remaining time with ms precision
	nowMs := time.Now().UnixMilli()
	var endTimestamp int64
	var remainingMs int64
	// Prefer Redis end_timestamp (may have been extended by delay)
	if rules, err := h.bidSvc.GetAuctionRules(c.Request.Context(), session.ID); err == nil {
		if et, ok := rules["end_timestamp"]; ok && et != "" {
			var redisEnd int64
			if _, parseErr := fmt.Sscanf(et, "%d", &redisEnd); parseErr == nil && redisEnd > 0 {
				endTimestamp = redisEnd
			}
		}
	}
	// Fallback to MySQL
	if endTimestamp == 0 && session.PlannedEndTime != nil {
		endTimestamp = session.PlannedEndTime.UnixMilli()
	}
	remainingMs = endTimestamp - nowMs
	if remainingMs < 0 {
		remainingMs = 0
	}

	// Suggested next bid
	nextBid := currentPrice
	if session.BidIncrement != "" {
		currentCents := parseCents(currentPrice)
		incrementCents := parseCents(session.BidIncrement)
		nextBid = formatCents(currentCents + incrementCents)
	}

	response.Success(c, roomAuctionResp{
		LiveRoom:       room,
		AuctionSession: &session,
		Product:        product,
		CurrentPrice:   currentPrice,
		BidCount:       bidCount,
		EndTimestamp:   endTimestamp,
		ServerTime:     nowMs,
		RemainingMs:    remainingMs,
		NextBid:        nextBid,
	})
}


// ListRoomProducts returns all auction sessions with products for a room (the "橱窗/showcase").
func (h *BuyerHandler) ListRoomProducts(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}

	// Only return products from the current live session (since room was started)
	room, err := h.liveRoomSvc.GetByID(c.Request.Context(), roomID)
	var since time.Time
	if err == nil && room.StartedAt != nil {
		since = *room.StartedAt
	}

	var sessions []model.AuctionSession
	if since.IsZero() {
		sessions, err = h.auctionSessionSvc.ListByRoomWithProducts(c.Request.Context(), roomID)
	} else {
		sessions, err = h.auctionSessionSvc.ListByRoomSince(c.Request.Context(), roomID, since)
	}
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}

	type productItem struct {
		AuctionID    uint64 `json:"auction_id"`
		ProductID    uint64 `json:"product_id"`
		Title        string `json:"title"`
		CoverImage   string `json:"cover_image"`
		Description  string `json:"description"`
		StartPrice   string `json:"start_price"`
		CurrentPrice string `json:"current_price"`
		BidIncrement string `json:"bid_increment"`
		CeilingPrice string `json:"ceiling_price"`
		BidCount     uint   `json:"bid_count"`
		DurationMin  int    `json:"duration_min"`
		DelaySeconds uint   `json:"delay_seconds"`
		Status         int     `json:"status"`
	FinalPrice     string  `json:"final_price,omitempty"`
	WinnerID       *uint64 `json:"winner_id,omitempty"`
	WinnerNickname string  `json:"winner_nickname,omitempty"`
	WinnerAvatar   string  `json:"winner_avatar,omitempty"`
	}

	// Collect winner user IDs
	winnerIDs := make(map[uint64]bool)
	for _, s := range sessions {
		if s.WinnerID != nil { winnerIDs[*s.WinnerID] = true }
	}
	winnerUsers := make(map[uint64]*model.User)
	for uid := range winnerIDs {
		if u, err := h.userRepo.GetByID(context.Background(), uid); err == nil { winnerUsers[uid] = u }
	}

	items := make([]productItem, 0, len(sessions))
	for _, s := range sessions {
		item := productItem{
			AuctionID:    s.ID,
			Status:       int(s.Status),
			BidCount:     s.BidCount,
			CurrentPrice: s.CurrentPrice,
			FinalPrice:   stringPtr(s.FinalPrice),
			DelaySeconds: s.DelaySeconds,
		}
		if s.WinnerID != nil {
			item.WinnerID = s.WinnerID
			if u, ok := winnerUsers[*s.WinnerID]; ok {
				item.WinnerNickname = u.Nickname
				item.WinnerAvatar = u.Avatar
			}
		}
		if s.Product != nil {
			item.ProductID = s.Product.ID
			item.Title = s.Product.Title
			item.CoverImage = s.Product.CoverImage
			item.Description = s.Product.Description
			item.StartPrice = s.Product.StartPrice
			item.BidIncrement = s.Product.BidIncrement
			item.CeilingPrice = s.Product.CeilingPrice
			item.DurationMin = s.Product.DurationMin
		}
		items = append(items, item)
	}

	response.Success(c, gin.H{"list": items})
}
// bidRankItem is one entry in the ranking.
type bidRankItem struct {
	Rank     uint64 `json:"rank"`
	UserID   uint64 `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Amount   string `json:"amount"`
	Time     string `json:"time"`
}

type rankingResp struct {
	Ranking []bidRankItem `json:"ranking"`
	MyBid   *bidRankItem  `json:"my_bid"` // null if not placed
}

// GetBidRanking returns top N bids + optional user's own rank.
func (h *BuyerHandler) GetBidRanking(c *gin.Context) {
	auctionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid session id")
		return
	}

	topN := int64(20)
	if n, err := strconv.ParseInt(c.DefaultQuery("top", "20"), 10, 64); err == nil && n > 0 && n <= 100 {
		topN = n
	}

	results, err := h.bidSvc.GetBidRanking(c.Request.Context(), auctionID, topN)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}

	ranking := make([]bidRankItem, 0, len(results))
	for i, z := range results {
		member, _ := z.Member.(string)
		var snap struct {
			UserID   uint64 `json:"user_id"`
			Nickname string `json:"nickname"`
			Avatar   string `json:"avatar"`
			Time     string `json:"bid_time"`
		}
		json.Unmarshal([]byte(member), &snap)

		ranking = append(ranking, bidRankItem{
			Rank:     uint64(i + 1),
			UserID:   snap.UserID,
			Nickname: snap.Nickname,
			Avatar:   snap.Avatar,
			Amount:   formatCents(int64(z.Score * 100)),
			Time:     snap.Time,
		})
	}

	// User's own bid
	var myBid *bidRankItem
	if userID, exists := c.Get("user_id"); exists {
		if ub, err := h.bidSvc.GetUserBidInAuction(c.Request.Context(), auctionID, userID.(uint64)); err == nil && ub != nil {
			myBid = &bidRankItem{
				Rank:     uint64(ub.Rank),
				UserID:   userID.(uint64),
				Nickname: ub.Nickname,
				Avatar:   ub.Avatar,
				Amount:   ub.Amount,
			}
		}
	}

	response.Success(c, rankingResp{Ranking: ranking, MyBid: myBid})
}

// ListMyBids returns the current user's bid history across all auctions.
func (h *BuyerHandler) ListMyBids(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	bids, total, err := h.bidSvc.ListByUser(c.Request.Context(), userID.(uint64), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.Bid]{
		List: bids, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

// parseCents converts a decimal string like "12.50" to cents (int64). Returns 0 on error.
func parseCents(s string) int64 {
	if s == "" {
		return 0
	}
	var dollars, cents int64
	// Simple parse: split on dot
	for i, c := range s {
		if c == '.' {
			dollars, _ = strconv.ParseInt(s[:i], 10, 64)
			frac := s[i+1:]
			if len(frac) > 2 {
				frac = frac[:2]
			}
			cents, _ = strconv.ParseInt(frac, 10, 64)
			return dollars*100 + cents
		}
	}
	dollars, _ = strconv.ParseInt(s, 10, 64)
	return dollars * 100
}

// formatCents converts cents back to a decimal string.
func formatCents(c int64) string {
	dollars := c / 100
	cents := c % 100
	return strconv.FormatInt(dollars, 10) + "." + pad2(cents)
}

func pad2(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func stringPtr(s *string) string {
	if s == nil { return "" }
	return *s
}

// BidHistoryItem is a joined row for the bid history page.
type bidHistoryItem struct {
	BidID          uint64  `json:"bid_id"`
	AuctionID      uint64  `json:"auction_id"`
	BidAmount      string  `json:"bid_amount"`
	BidTime        string  `json:"bid_time"`
	ProductID      uint64  `json:"product_id"`
	ProductTitle   string  `json:"product_title"`
	ProductImage   string  `json:"product_image"`
	FinalPrice     *string `json:"final_price"`
	AuctionStatus  int     `json:"auction_status"`
	WinnerID       *uint64 `json:"winner_id"`
	RoomID         uint64  `json:"room_id"`
	SellerNickname string  `json:"seller_nickname"`
	SellerAvatar   string  `json:"seller_avatar"`
}

func (h *BuyerHandler) BidHistory(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()

	// Raw SQL join to get full history in one query
	type row struct {
		BidID          uint64  `gorm:"column:bid_id"`
		AuctionID      uint64  `gorm:"column:auction_id"`
		BidAmount      string  `gorm:"column:bid_amount"`
		BidTime        string  `gorm:"column:bid_time"`
		ProductID      uint64  `gorm:"column:product_id"`
		ProductTitle   string  `gorm:"column:product_title"`
		ProductImage   string  `gorm:"column:product_image"`
		FinalPrice     *string `gorm:"column:final_price"`
		AuctionStatus  int     `gorm:"column:auction_status"`
		WinnerID       *uint64 `gorm:"column:winner_id"`
		RoomID         uint64  `gorm:"column:room_id"`
		SellerNickname string  `gorm:"column:seller_nickname"`
		SellerAvatar   string  `gorm:"column:seller_avatar"`
		AuctionStart   string  `gorm:"column:auction_start"`
	}

	var total int64
	h.db.Table("bids b").
		Select("b.id as bid_id, b.auction_id, b.amount as bid_amount, b.bid_time, p.id as product_id, p.title as product_title, COALESCE(p.cover_image,'') as product_image, auc.final_price, auc.status as auction_status, auc.winner_id, lr.id as room_id, u.nickname as seller_nickname, COALESCE(u.avatar,'') as seller_avatar, auc.start_time as auction_start").
		Joins("JOIN auction_sessions auc ON auc.id = b.auction_id").
		Joins("JOIN products p ON p.id = auc.product_id").
		Joins("JOIN live_rooms lr ON lr.id = auc.room_id").
		Joins("JOIN users u ON u.id = lr.seller_id").
		Where("b.id IN (SELECT MAX(id) FROM bids WHERE user_id = ? AND is_valid = 1 GROUP BY auction_id)", userID).
		Count(&total)

	var rows []row
	err := h.db.Table("bids b").
		Select("b.id as bid_id, b.auction_id, b.amount as bid_amount, b.bid_time, p.id as product_id, p.title as product_title, COALESCE(p.cover_image,'') as product_image, auc.final_price, auc.status as auction_status, auc.winner_id, lr.id as room_id, u.nickname as seller_nickname, COALESCE(u.avatar,'') as seller_avatar, auc.start_time as auction_start").
		Joins("JOIN auction_sessions auc ON auc.id = b.auction_id").
		Joins("JOIN products p ON p.id = auc.product_id").
		Joins("JOIN live_rooms lr ON lr.id = auc.room_id").
		Joins("JOIN users u ON u.id = lr.seller_id").
		Where("b.id IN (SELECT MAX(id) FROM bids WHERE user_id = ? AND is_valid = 1 GROUP BY auction_id)", userID).
		Order("b.bid_time DESC").
		Offset(page.Offset()).Limit(page.PageSize).
		Find(&rows).Error

	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}

	// Convert to response items
	type item struct {
		BidID          uint64  `json:"bid_id"`
		AuctionID      uint64  `json:"auction_id"`
		BidAmount      string  `json:"bid_amount"`
		BidTime        string  `json:"bid_time"`
		ProductID      uint64  `json:"product_id"`
		ProductTitle   string  `json:"product_title"`
		ProductImage   string  `json:"product_image"`
		FinalPrice     *string `json:"final_price"`
		AuctionStatus  int     `json:"auction_status"`
		WinnerID       *uint64 `json:"winner_id"`
		RoomID         uint64  `json:"room_id"`
		SellerNickname string  `json:"seller_nickname"`
		SellerAvatar   string  `json:"seller_avatar"`
			AuctionStart   string  `json:"auction_start"`
	}
	items := make([]item, len(rows))
	for i, r := range rows {
		items[i] = item(r)
	}

	response.Success(c, gin.H{"list": items, "total": total})
}
