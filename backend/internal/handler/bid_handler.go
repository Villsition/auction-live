package handler

import (
	"strconv"
	"time"

	"auction/internal/model"
	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type BidHandler struct {
	svc *service.BidSvc
}

func NewBidHandler(svc *service.BidSvc) *BidHandler {
	return &BidHandler{svc: svc}
}

func (h *BidHandler) Create(c *gin.Context) {
	var req struct {
		AuctionID      uint64 `json:"auction_id" binding:"required"`
		Amount         string `json:"amount" binding:"required"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}

	if !isValidBidAmount(req.Amount) {
		response.Error(c, errcode.ErrInvalidParam, "amount must be a positive integer, e.g. 100")
		return
	}

	userID, _ := c.Get("user_id")
	nickname, _ := c.Get("nickname")
	avatar, _ := c.Get("avatar")
	bid := model.Bid{
		AuctionID:      req.AuctionID,
		UserID:         userID.(uint64),
		Amount:         req.Amount,
		BidTime:        time.Now(),
		ClientIP:       c.ClientIP(),
		Nickname:       strVal(nickname),
		Avatar:         strVal(avatar),
		IdempotencyKey: req.IdempotencyKey,
	}

	result, err := h.svc.PlaceBid(c.Request.Context(), &bid)
	if err != nil {
		if err == service.ErrBidTooFast {
			response.Error(c, errcode.ErrTooManyRequest, err.Error())
			return
		}
		if err == service.ErrDuplicateBid {
			response.Error(c, errcode.ErrConflict, err.Error())
			return
		}
		response.Error(c, errcode.ErrConflict, err.Error())
		return
	}
	response.Success(c, gin.H{
		"rank":         result.Rank,
		"ceiling_deal":      result.CeilingDeal,
		"delay_extend":      result.DelayExtend,
		"new_end_timestamp": result.NewEndTimestamp,
		"final_delay":       result.FinalDelay,
		"bid":               bid,
	})
}

func (h *BidHandler) ListByAuction(c *gin.Context) {
	auctionID, err := strconv.ParseUint(c.Query("auction_id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid auction_id")
		return
	}
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	bids, total, err := h.svc.ListByAuction(c.Request.Context(), auctionID, page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.Bid]{
		List: bids, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

// isValidBidAmount checks that s is a positive integer-only string (no decimals, no signs).
func isValidBidAmount(s string) bool {
	if s == "" || s == "0" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// strVal safely extracts a string from a context value (may be nil).
func strVal(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
