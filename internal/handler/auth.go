package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/OpenKorProject/korauth/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	TenantID string `json:"tenant_id" binding:"required,uuid"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	tenantID, _ := uuid.Parse(req.TenantID)
	pair, err := h.svc.Login(c.Request.Context(), tenantID, req.Username, req.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			respondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password")
		case service.ErrAccountLocked:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": gin.H{
				"code":       "ACCOUNT_LOCKED",
				"message":    "too many failed attempts, try again in 15 minutes",
				"details":    gin.H{"retry_after_seconds": 900},
				"request_id": reqID(c),
			}})
		default:
			respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
		}
		return
	}
	c.JSON(http.StatusOK, pair)
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if err == service.ErrInvalidRefreshToken {
			respondError(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "refresh token is invalid or expired")
			return
		}
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
		return
	}
	c.JSON(http.StatusOK, pair)
}

type logoutReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
		return
	}
	c.Status(http.StatusNoContent)
}
