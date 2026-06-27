package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/OpenKorProject/korauth/internal/model"
	"github.com/OpenKorProject/korauth/internal/service"
)

type RoleHandler struct {
	svc *service.RoleService
}

func NewRoleHandler(svc *service.RoleService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

func (h *RoleHandler) List(c *gin.Context) {
	roles, err := h.svc.List(c.Request.Context())
	if err != nil {
		respondServiceError(c, err)
		return
	}
	if roles == nil {
		roles = []model.Role{}
	}
	c.JSON(http.StatusOK, gin.H{"data": roles})
}
