package handler

import (
	"strconv"

	"auction/internal/model"
	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuctionLogHandler struct {
	svc *service.AuctionLogSvc
}

func NewAuctionLogHandler(svc *service.AuctionLogSvc) *AuctionLogHandler {
	return &AuctionLogHandler{svc: svc}
}

func (h *AuctionLogHandler) ListByAuction(c *gin.Context) {
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
	logs, total, err := h.svc.ListByAuction(c.Request.Context(), auctionID, page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.AuctionLog]{
		List: logs, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}
