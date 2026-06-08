package handler

import (
	"errors"

	"auction/internal/service"
	"auction/pkg/errcode"
	"auction/pkg/response"
	"auction/pkg/upload"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc      *service.AuthSvc
	uploader *upload.Uploader
}

func NewAuthHandler(svc *service.AuthSvc, uploader *upload.Uploader) *AuthHandler {
	return &AuthHandler{svc: svc, uploader: uploader}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input service.RegisterInput
	// Try multipart form first (with avatar), fallback to JSON
	contentType := c.GetHeader("Content-Type")
	if len(contentType) > 9 && contentType[:9] == "multipart" {
		if err := c.ShouldBind(&input); err != nil {
			response.Error(c, errcode.ErrInvalidParam, err.Error())
			return
		}
		// Handle avatar upload
		file, err := c.FormFile("avatar")
		if err == nil {
			url, saveErr := h.uploader.SaveImage(file)
			if saveErr == nil {
				input.Avatar = url
			}
		}
	} else {
		if err := c.ShouldBindJSON(&input); err != nil {
			response.Error(c, errcode.ErrInvalidParam, err.Error())
			return
		}
	}

	result, err := h.svc.Register(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrUsernameTaken) {
			response.Error(c, errcode.ErrConflict, err.Error())
			return
		}
		response.Error(c, errcode.ErrInternal, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input service.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, errcode.ErrInvalidParam, err.Error())
		return
	}

	result, err := h.svc.Login(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, service.ErrAlreadyLoggedIn) {
			response.Error(c, errcode.ErrConflict, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidLogin) || errors.Is(err, service.ErrUserDisabled) {
			response.Error(c, errcode.ErrUnauthorized, err.Error())
			return
		}
		response.Error(c, errcode.ErrInternal, err.Error())
		return
	}

	response.Success(c, result)
}
