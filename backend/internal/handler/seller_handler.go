package handler

import (
	"strconv"
	"sync"
	"time"

	"auction/internal/model"
	"auction/internal/service"
	"auction/internal/ws"
	"auction/pkg/errcode"
	"auction/pkg/response"
	"auction/pkg/upload"

	"github.com/gin-gonic/gin"
)

type SellerHandler struct {
	productSvc        *service.ProductSvc
	auctionSessionSvc *service.AuctionSessionSvc
	bidSvc            *service.BidSvc
	orderSvc          *service.OrderSvc
	uploader          *upload.Uploader
	hub               *ws.Hub
}

func NewSellerHandler(
	productSvc *service.ProductSvc,
	auctionSessionSvc *service.AuctionSessionSvc,
	bidSvc *service.BidSvc,
	orderSvc *service.OrderSvc,
	uploader *upload.Uploader,
	hub *ws.Hub,
) *SellerHandler {
	return &SellerHandler{
		productSvc:        productSvc,
		auctionSessionSvc: auctionSessionSvc,
		bidSvc:            bidSvc,
		orderSvc:          orderSvc,
		uploader:          uploader,
		hub:               hub,
	}
}

// ============================================================
// Image upload
// ============================================================

func (h *SellerHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "missing file")
		return
	}
	url, err := h.uploader.SaveImage(file)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	response.Success(c, gin.H{"url": url})
}

func (h *SellerHandler) UploadVideo(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "missing file")
		return
	}
	url, err := h.uploader.SaveVideo(file)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	response.Success(c, gin.H{"url": url})
}

// ============================================================
// Product management
// ============================================================

type createProductReq struct {
	Title        string   `json:"title" binding:"required"`
	Description  string   `json:"description"`
	CoverImage   string   `json:"cover_image"`
	Images       []string `json:"images"`
	CategoryID   uint64   `json:"category_id"`
	StartPrice   string   `json:"start_price"`
	BidIncrement string   `json:"bid_increment"`
	CeilingPrice string   `json:"ceiling_price"`
	DurationMin  int      `json:"duration_min"`
	DelaySeconds uint     `json:"delay_seconds"`
}

func (h *SellerHandler) CreateProduct(c *gin.Context) {
	var req createProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}

	if req.StartPrice == "" {
		req.StartPrice = "0"
	}
	if req.BidIncrement == "" {
		req.BidIncrement = "10"
	}
	if req.CeilingPrice == "" {
		req.CeilingPrice = "0"
	}
	if req.DurationMin <= 0 {
		req.DurationMin = 5
	}
	if req.DelaySeconds == 0 {
		req.DelaySeconds = 30
	}
	if req.DelaySeconds < 10 || req.DelaySeconds > 30 {
		response.Error(c, errcode.ErrInvalidParam, "延长时间请设置在10s-30s内")
		return
	}

	sellerID, _ := c.Get("user_id")
	product := model.Product{
		SellerID:     sellerID.(uint64),
		CategoryID:   req.CategoryID,
		Title:        req.Title,
		Description:  req.Description,
		CoverImage:   req.CoverImage,
		Images:       req.Images,
		StartPrice:   req.StartPrice,
		BidIncrement: req.BidIncrement,
		CeilingPrice: req.CeilingPrice,
		DurationMin:  req.DurationMin,
		DelaySeconds: req.DelaySeconds,
		Status:       model.ProductStatusDraft,
	}

	if err := h.productSvc.Create(c.Request.Context(), &product); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, product)
}

type updateProductReq struct {
	Title        string              `json:"title"`
	Description  string              `json:"description"`
	CoverImage   string              `json:"cover_image"`
	Images       []string            `json:"images"`
	CategoryID   uint64              `json:"category_id"`
	StartPrice   string              `json:"start_price"`
	BidIncrement string              `json:"bid_increment"`
	CeilingPrice string              `json:"ceiling_price"`
	DurationMin  *int                `json:"duration_min"`
	DelaySeconds *uint               `json:"delay_seconds"`
	Status       model.ProductStatus `json:"status"`
}

func (h *SellerHandler) UpdateProduct(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid product id")
		return
	}
	var req updateProductReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}

	updates := map[string]any{}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.CoverImage != "" {
		updates["cover_image"] = req.CoverImage
	}
	if req.Images != nil {
		updates["images"] = model.StringArray(req.Images)
	}
	if req.CategoryID > 0 {
		updates["category_id"] = req.CategoryID
	}
	if req.Status >= model.ProductStatusDraft && req.Status <= model.ProductStatusCancelled {
		if req.StartPrice != "" {
			updates["start_price"] = req.StartPrice
		}
		if req.BidIncrement != "" {
			updates["bid_increment"] = req.BidIncrement
		}
		if req.CeilingPrice != "" {
			updates["ceiling_price"] = req.CeilingPrice
		}
		if req.DurationMin != nil {
			updates["duration_min"] = *req.DurationMin
		}
		if req.DelaySeconds != nil {
			v := *req.DelaySeconds
			if v < 10 || v > 30 {
				response.Error(c, errcode.ErrInvalidParam, "延长时间请设置在10s-30s内")
				return
			}
			updates["delay_seconds"] = v
		}
		updates["status"] = req.Status
	}

	if err := h.productSvc.Update(c.Request.Context(), id, updates); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *SellerHandler) GetProduct(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid product id")
		return
	}
	product, err := h.productSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "product not found")
		return
	}
	response.Success(c, product)
}

func (h *SellerHandler) DeleteProduct(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid product id")
		return
	}
	// Soft-delete via status
	if err := h.productSvc.Update(c.Request.Context(), id, map[string]any{"status": model.ProductStatusCancelled}); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, nil)
}

// ============================================================
// Auction session management
// ============================================================

type createSessionReq struct {
	RoomID       uint64 `json:"room_id" binding:"required"`
	ProductID    uint64 `json:"product_id" binding:"required"`
	StartPrice   string `json:"start_price" binding:"required"`
	BidIncrement string `json:"bid_increment" binding:"required"`
	CeilingPrice string `json:"ceiling_price"`
	DelaySeconds uint   `json:"delay_seconds"`
	DurationMin  int    `json:"duration_min" binding:"required"` // auction duration in minutes
	SortOrder    uint   `json:"sort_order"`
}

func (h *SellerHandler) CreateAuctionSession(c *gin.Context) {
	var req createSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}

	if req.DelaySeconds < 10 || req.DelaySeconds > 30 {
		response.Error(c, errcode.ErrInvalidParam, "延长时间请设置在10s-30s内")
		return
	}
	if req.CeilingPrice == "" {
		req.CeilingPrice = "0"
	}

	session := model.AuctionSession{
		RoomID:       req.RoomID,
		ProductID:    req.ProductID,
		StartPrice:   req.StartPrice,
		CurrentPrice: req.StartPrice,
		BidIncrement: req.BidIncrement,
		CeilingPrice: req.CeilingPrice,
		DelaySeconds: req.DelaySeconds,
		SortOrder:    req.SortOrder,
		Status:       model.AuctionStatusPending,
	}

	plannedEnd := time.Now().Add(time.Duration(req.DurationMin) * time.Minute)
	session.PlannedEndTime = &plannedEnd

	if err := h.auctionSessionSvc.Create(c.Request.Context(), &session); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, session)
}

type updateSessionReq struct {
	StartPrice   string `json:"start_price"`
	BidIncrement string `json:"bid_increment"`
	CeilingPrice string `json:"ceiling_price"`
	DelaySeconds *uint  `json:"delay_seconds"`
	DurationMin  *int   `json:"duration_min"`
	SortOrder    *uint  `json:"sort_order"`
}

func (h *SellerHandler) UpdateAuctionSession(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid session id")
		return
	}

	session, err := h.auctionSessionSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "session not found")
		return
	}

	// Only allow editing pending sessions
	if session.Status != model.AuctionStatusPending {
		response.Error(c, errcode.ErrConflict, "only pending sessions can be modified")
		return
	}

	var req updateSessionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}

	updates := map[string]any{}
	if req.StartPrice != "" {
		updates["start_price"] = req.StartPrice
		updates["current_price"] = req.StartPrice
	}
	if req.BidIncrement != "" {
		updates["bid_increment"] = req.BidIncrement
	}
	if req.CeilingPrice != "" {
		updates["ceiling_price"] = req.CeilingPrice
	}
	if req.DelaySeconds != nil {
		v := *req.DelaySeconds
		if v < 10 || v > 30 {
			response.Error(c, errcode.ErrInvalidParam, "延长时间请设置在10s-30s内")
			return
  }
		updates["delay_seconds"] = v
	}
	if req.DurationMin != nil {
		plannedEnd := time.Now().Add(time.Duration(*req.DurationMin) * time.Minute)
		updates["planned_end_time"] = plannedEnd
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}

	if err := h.auctionSessionSvc.Update(c.Request.Context(), id, updates); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *SellerHandler) ListAuctionSessions(c *gin.Context) {
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	sessions, total, err := h.auctionSessionSvc.List(c.Request.Context(), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.AuctionSession]{
		List: sessions, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

// StartAuction activates the auction and caches rules to Redis.
func (h *SellerHandler) StartAuction(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid session id")
		return
	}

	session, err := h.auctionSessionSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "session not found")
		return
	}

	if session.Status != model.AuctionStatusPending {
		response.Error(c, errcode.ErrConflict, "session is not in pending status")
		return
	}

	now := time.Now()
	updates := map[string]any{
		"status":     model.AuctionStatusActive,
		"start_time": now,
	}
	if err := h.auctionSessionSvc.Update(c.Request.Context(), id, updates); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}

	// Cache rules to Redis for real-time bidding
	session.Status = model.AuctionStatusActive
	session.StartTime = &now
	if err := h.bidSvc.CacheAuctionRules(c.Request.Context(), session); err != nil {
		response.Error(c, errcode.ErrDatabase, "failed to cache auction rules: "+err.Error())
		return
	}

	// Register auction→room mapping for room-level broadcast isolation
	h.hub.SetAuctionRoom(id, session.RoomID)

	// Broadcast start event to WebSocket room
	h.hub.BroadcastToAuction(id, &ws.AuctionEvent{
		Type:      ws.EventAuctionStart,
		AuctionID: id,
		Status:    "started",
		Message:   "竞拍开始",
	})

	response.Success(c, session)
}

// CancelAuction cancels an active auction.
func (h *SellerHandler) CancelAuction(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid session id")
		return
	}

	session, err := h.auctionSessionSvc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "session not found")
		return
	}

	if session.Status != model.AuctionStatusActive && session.Status != model.AuctionStatusPending {
		response.Error(c, errcode.ErrConflict, "cannot cancel in current status")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	operatorID, _ := c.Get("user_id")
	now := time.Now()
	updates := map[string]any{
		"status":        model.AuctionStatusCancelled,
		"cancel_reason": req.Reason,
		"cancelled_by":  operatorID,
		"cancelled_at":  now,
		"actual_end_time": now,
	}
	if err := h.auctionSessionSvc.Update(c.Request.Context(), id, updates); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}

	h.bidSvc.SetAuctionStatus(c.Request.Context(), id, model.AuctionStatusCancelled)

	// Clean up auction→room mapping
	h.hub.RemoveAuctionRoom(id)

	h.hub.BroadcastToAuction(id, &ws.AuctionEvent{
		Type:      ws.EventAuctionCancel,
		AuctionID: id,
		Status:    "cancelled",
		Message:   "竞拍已取消: " + req.Reason,
	})

	response.Success(c, nil)
}

// ============================================================
// Dashboard
// ============================================================

type dashboardResp struct {
	ProductStats  map[string]int64         `json:"product_stats"`
	AuctionStats  map[string]int64         `json:"auction_stats"`
	OrderStats    map[string]int64         `json:"order_stats"`
	RevenueTotal  string                   `json:"revenue_total"`
	ActiveBidding []dashboardActiveAuction `json:"active_bidding"`
}

type dashboardActiveAuction struct {
	ID           uint64 `json:"id"`
	ProductTitle string `json:"product_title"`
	CoverImage   string `json:"cover_image"`
	CurrentPrice string `json:"current_price"`
	BidCount     int64  `json:"bid_count"`
		AuctionStart  *time.Time `json:"auction_start"`
	RemainingMs  int64  `json:"remaining_ms"`
}

func (h *SellerHandler) Dashboard(c *gin.Context) {
	sellerID, _ := c.Get("user_id")
	ctx := c.Request.Context()
	uid := sellerID.(uint64)

	var (
		productStats map[int]int64
		auctionStats map[int]int64
		orderStats   map[int]int64
		revenue      string
	)
	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); productStats, _ = h.productSvc.CountByStatus(ctx, uid) }()
	go func() { defer wg.Done(); auctionStats, _ = h.auctionSessionSvc.CountBySeller(ctx, uid) }()
	go func() { defer wg.Done(); orderStats, _ = h.orderSvc.CountByStatus(ctx, uid) }()
	go func() { defer wg.Done(); revenue, _ = h.orderSvc.RevenueBySeller(ctx, uid) }()
	wg.Wait()

	activeAuctions, _ := h.auctionSessionSvc.ListActiveBySeller(ctx, uid)
	nowMs := time.Now().UnixMilli()
	bidding := make([]dashboardActiveAuction, 0, len(activeAuctions))
	for _, auc := range activeAuctions {
		price, _ := h.bidSvc.GetCurrentPrice(ctx, auc.ID)
		if price == "" {
			price = auc.StartPrice
		}
		count, _ := h.bidSvc.GetBidCount(ctx, auc.ID)
		var remainingMs int64
		if auc.PlannedEndTime != nil {
			remainingMs = auc.PlannedEndTime.UnixMilli() - nowMs
			if remainingMs < 0 {
				remainingMs = 0
			}
		}
		title := ""
		cover := ""
		if auc.Product != nil {
			title = auc.Product.Title
			cover = auc.Product.CoverImage
		}
		bidding = append(bidding, dashboardActiveAuction{
			ID: auc.ID, ProductTitle: title, CoverImage: cover,
			CurrentPrice: price, BidCount: count, RemainingMs: remainingMs,
		})
	}

	response.Success(c, dashboardResp{
		ProductStats:  formatStats(productStats, productStatusNames),
		AuctionStats:  formatStats(auctionStats, auctionStatusNames),
		OrderStats:    formatStats(orderStats, orderStatusNames),
		RevenueTotal:  revenue,
		ActiveBidding: bidding,
	})
}

var productStatusNames = map[int]string{0: "draft", 1: "listed", 2: "bidding", 3: "sold", 4: "unsold", 5: "cancelled"}
var auctionStatusNames = map[int]string{0: "pending", 1: "active", 2: "sold", 3: "unsold", 4: "cancelled"}
var orderStatusNames = map[int]string{0: "unpaid", 1: "paid", 2: "shipped", 3: "completed", 4: "cancelled", 5: "refunded"}

func formatStats(m map[int]int64, names map[int]string) map[string]int64 {
	out := make(map[string]int64, len(names))
	for k, name := range names {
		out[name] = m[k]
	}
	return out
}

// ============================================================
// Enhanced product list with auction info, search & filter
// ============================================================

type productWithAuctionResp struct {
	ID            uint64  `json:"id"`
	Title         string  `json:"title"`
	CoverImage    string  `json:"cover_image"`
	StartPrice    string  `json:"start_price"`
	BidIncrement  string  `json:"bid_increment"`
	CeilingPrice  string  `json:"ceiling_price"`
		DurationMin   int     `json:"duration_min"`
		DelaySeconds  uint    `json:"delay_seconds"`
	Status        int     `json:"status"`
	StatusName    string  `json:"status_name"`
	AuctionID     *uint64 `json:"auction_id"`
	AuctionStatus *uint8  `json:"auction_status"`
	CurrentPrice  *string `json:"current_price"`
	FinalPrice    *string `json:"final_price"`
	BidCount      *uint   `json:"bid_count"`
		AuctionStart  *time.Time `json:"auction_start"`
}

func (h *SellerHandler) ListProducts(c *gin.Context) {
	sellerID, _ := c.Get("user_id")
	uid := sellerID.(uint64)
	ctx := c.Request.Context()

	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()

	keyword := c.Query("keyword")
	statusStr := c.Query("status")
	status := -1
	if statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil && s >= 0 {
			status = s
		}
	}

	rows, total, err := h.productSvc.ListWithAuction(ctx, uid, keyword, status, page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}

	list := make([]productWithAuctionResp, 0, len(rows))
	for _, r := range rows {
		list = append(list, productWithAuctionResp{
			ID: r.ID, Title: r.Title, CoverImage: r.CoverImage,
			StartPrice: r.StartPrice, BidIncrement: r.BidIncrement, CeilingPrice: r.CeilingPrice,
				DurationMin: r.DurationMin, DelaySeconds: r.DelaySeconds,
			Status: int(r.Status), StatusName: productStatusNames[int(r.Status)],
			AuctionID: r.AuctionID, AuctionStatus: r.AuctionStatus,
			CurrentPrice: r.CurrentPrice, FinalPrice: r.FinalPrice, BidCount: r.BidCount,
				AuctionStart: r.AuctionStart,
		})
	}

	response.SuccessPage(c, model.PageResult[productWithAuctionResp]{
		List: list, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}
