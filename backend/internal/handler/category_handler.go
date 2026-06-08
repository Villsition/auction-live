package handler

import (
	"strconv"

	"auction/internal/model"
	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	svc *service.CategorySvc
}

func NewCategoryHandler(svc *service.CategorySvc) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

func (h *CategoryHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid category id")
		return
	}
	cat, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "category not found")
		return
	}
	response.Success(c, cat)
}

func (h *CategoryHandler) List(c *gin.Context) {
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	list, total, err := h.svc.List(c.Request.Context(), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.Category]{
		List: list, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}
