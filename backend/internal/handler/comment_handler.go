package handler

import (
	"strconv"

	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	svc *service.CommentSvc
}

func NewCommentHandler(svc *service.CommentSvc) *CommentHandler {
	return &CommentHandler{svc: svc}
}

func (h *CommentHandler) Send(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}

	var req struct {
		Content string `json:"content" binding:"required,max=500"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	comment, err := h.svc.Create(c.Request.Context(), roomID, userID.(uint64), req.Content)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, comment)
}

func (h *CommentHandler) List(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid room id")
		return
	}

	limit := 50
	if n, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && n > 0 && n <= 200 {
		limit = n
	}

	comments, err := h.svc.ListByRoom(c.Request.Context(), roomID, limit)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, comments)
}
