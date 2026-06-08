package handler

import (
	"strconv"

	"auction/internal/model"
	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc *service.NotificationSvc
}

func NewNotificationHandler(svc *service.NotificationSvc) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) ListByUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, errcode.ErrUnauthorized, "login required")
		return
	}
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	list, total, err := h.svc.ListByUser(c.Request.Context(), userID.(uint64), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.Notification]{
		List: list, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid notification id")
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, errcode.ErrUnauthorized, "login required")
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), id, userID.(uint64)); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, nil)
}
