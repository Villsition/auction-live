package handler

import (
	"strconv"

	"auction/internal/model"
	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type PaymentRecordHandler struct {
	svc *service.PaymentRecordSvc
}

func NewPaymentRecordHandler(svc *service.PaymentRecordSvc) *PaymentRecordHandler {
	return &PaymentRecordHandler{svc: svc}
}

func (h *PaymentRecordHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid payment id")
		return
	}
	record, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "payment record not found")
		return
	}
	response.Success(c, record)
}

func (h *PaymentRecordHandler) List(c *gin.Context) {
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	records, total, err := h.svc.List(c.Request.Context(), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.PaymentRecord]{
		List: records, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}
