package handler

import (
	"strconv"
	"strings"

	"auction/internal/model"
	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	svc *service.OrderSvc
}

func NewOrderHandler(svc *service.OrderSvc) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid order id")
		return
	}
	order, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "order not found")
		return
	}
	response.Success(c, order)
}

func (h *OrderHandler) ListMyOrders(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	items, total, err := h.svc.ListByBuyerWithDetails(c.Request.Context(), userID.(uint64), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.SellerOrderItem]{
		List: items, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

func (h *OrderHandler) ListSellerOrders(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()

	// Parse optional status filter (comma-separated: "0,2")
	statusStr := c.Query("status")
	var statusFilter []int
	if statusStr != "" {
		for _, s := range strings.Split(statusStr, ",") {
			if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				statusFilter = append(statusFilter, v)
			}
		}
	}

	// If status filter is provided, return enriched data with product/buyer info
	if len(statusFilter) > 0 {
		items, total, err := h.svc.ListBySellerWithDetails(c.Request.Context(), userID.(uint64), page, statusFilter...)
		if err != nil {
			response.Error(c, errcode.ErrDatabase, err.Error())
			return
		}
		response.SuccessPage(c, model.PageResult[model.SellerOrderItem]{
			List: items, Total: total, Page: page.Page, PageSize: page.PageSize,
		})
		return
	}

	orders, total, err := h.svc.ListBySeller(c.Request.Context(), userID.(uint64), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.Order]{
		List: orders, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

// ConfirmAddress sets the shipping address for an unpaid order.
func (h *OrderHandler) ConfirmAddress(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid order id")
		return
	}

	var req struct {
		Address string `json:"address" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}

	if err := h.svc.ConfirmAddress(c.Request.Context(), id, req.Address); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, nil)
}

// Pay simulates a payment.
func (h *OrderHandler) Pay(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid order id")
		return
	}

	if err := h.svc.Pay(c.Request.Context(), id); err != nil {
		if err.Error() == "order payment deadline has expired" {
			response.Error(c, errcode.ErrConflict, "订单已过期，请重新竞拍")
			return
		}
		response.Error(c, errcode.ErrConflict, err.Error())
		return
	}

	// Reload to show updated status
	order, _ := h.svc.GetByID(c.Request.Context(), id)
	response.Success(c, order)
}

// ShipOrder marks a paid order as shipped (seller action).
func (h *OrderHandler) ShipOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid order id")
		return
	}
	if err := h.svc.Ship(c.Request.Context(), id); err != nil {
		response.Error(c, errcode.ErrConflict, err.Error())
		return
	}
	response.Success(c, nil)
}

// ConfirmReceipt marks a shipped order as completed (buyer action).
func (h *OrderHandler) ConfirmReceipt(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid order id")
		return
	}
	if err := h.svc.ConfirmReceipt(c.Request.Context(), id); err != nil {
		response.Error(c, errcode.ErrConflict, err.Error())
		return
	}
	response.Success(c, nil)
}
