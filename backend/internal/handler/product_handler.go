package handler

import (
	"strconv"

	"auction/internal/model"
	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	svc *service.ProductSvc
}

func NewProductHandler(svc *service.ProductSvc) *ProductHandler {
	return &ProductHandler{svc: svc}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var p model.Product
	if err := c.ShouldBindJSON(&p); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	// Validate: ceiling_price must be >= start_price (0 = no ceiling)
	cp, _ := strconv.ParseFloat(p.CeilingPrice, 64)
	sp, _ := strconv.ParseFloat(p.StartPrice, 64)
	if cp > 0 && cp <= sp {
		response.Error(c, errcode.ErrInvalidParam, "封顶价不能低于起拍价")
		return
	}
	if err := h.svc.Create(c.Request.Context(), &p); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, p)
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid product id")
		return
	}
	product, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, errcode.ErrNotFound, "product not found")
		return
	}
	response.Success(c, product)
}

func (h *ProductHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParam, "invalid product id")
		return
	}
	var updates map[string]any
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	// Validate: if both ceiling_price and start_price are provided, ceiling >= start
	if cpRaw, ok := updates["ceiling_price"]; ok {
		cp, _ := strconv.ParseFloat(asString(cpRaw), 64)
		if spRaw, ok2 := updates["start_price"]; ok2 {
			sp, _ := strconv.ParseFloat(asString(spRaw), 64)
			if cp > 0 && cp <= sp {
				response.Error(c, errcode.ErrInvalidParam, "封顶价不能低于起拍价")
				return
			}
		}
	}
	if err := h.svc.Update(c.Request.Context(), id, updates); err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.Success(c, nil)
}

func (h *ProductHandler) List(c *gin.Context) {
	var page model.PageRequest
	if err := c.ShouldBindQuery(&page); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}
	page.Normalize()
	products, total, err := h.svc.List(c.Request.Context(), page)
	if err != nil {
		response.Error(c, errcode.ErrDatabase, err.Error())
		return
	}
	response.SuccessPage(c, model.PageResult[model.Product]{
		List: products, Total: total, Page: page.Page, PageSize: page.PageSize,
	})
}

// asString converts any value from JSON unmarshal to string.
func asString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64)
	default:
		return ""
	}
}
