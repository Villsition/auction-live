package handler

import (
	"strconv"

	"auction/internal/model"
	"auction/internal/service"
	"auction/internal/ws"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuctionSessionHandler struct {
	svc *service.AuctionSessionSvc
	hub *ws.Hub
}

func NewAuctionSessionHandler(svc *service.AuctionSessionSvc, hub *ws.Hub) *AuctionSessionHandler {
	return &AuctionSessionHandler{svc: svc, hub: hub}
}

func (h *AuctionSessionHandler) Create(c *gin.Context) {
	var session model.AuctionSession
	if err := c.ShouldBindJSON(&session); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	if err := h.svc.Create(c.Request.Context(), &session); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, session)
}

func (h *AuctionSessionHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid session id")
		return
	}
	session, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "session not found")
		return
	}
	response.Success(c, session)
}

func (h *AuctionSessionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid session id")
		return
	}
	var updates map[string]any
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, updates); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *AuctionSessionHandler) List(c *gin.Context) {
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	sessions, total, err := h.svc.List(c.Request.Context(), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.AuctionSession]{
		List: sessions, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

// GetOnlineCount returns the number of online viewers for an auction.
func (h *AuctionSessionHandler) GetOnlineCount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid session id")
		return
	}
	count, _ := h.hub.GetOnlineCount(c.Request.Context(), id)
	response.Success(c, gin.H{"auction_id": id, "online": count})
}
