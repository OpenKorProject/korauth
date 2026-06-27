package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/OpenKorProject/korauth/internal/token"
)

type JWKSHandler struct {
	tokenSvc *token.Service
}

func NewJWKSHandler(tokenSvc *token.Service) *JWKSHandler {
	return &JWKSHandler{tokenSvc: tokenSvc}
}

func (h *JWKSHandler) Get(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, h.tokenSvc.JWKS())
}
