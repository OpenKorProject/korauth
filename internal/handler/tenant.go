package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/OpenKorProject/korauth/internal/model"
	"github.com/OpenKorProject/korauth/internal/service"
)

type TenantHandler struct {
	svc *service.TenantService
}

func NewTenantHandler(svc *service.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

func (h *TenantHandler) List(c *gin.Context) {
	page, perPage := paginationQuery(c)
	tenants, total, err := h.svc.List(c.Request.Context(), page, perPage)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	if tenants == nil {
		tenants = []model.Tenant{}
	}
	c.JSON(http.StatusOK, gin.H{
		"data": tenants,
		"pagination": paginationResp{
			Page:    page,
			PerPage: perPage,
			Total:   total,
		},
	})
}

type createTenantReq struct {
	Name string `json:"name" binding:"required,min=1,max=128"`
}

func (h *TenantHandler) Create(c *gin.Context) {
	var req createTenantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	tenant, err := h.svc.Create(c.Request.Context(), req.Name)
	if err != nil {
		if err == service.ErrConflict {
			respondError(c, http.StatusConflict, "TENANT_NAME_TAKEN", "a tenant with this name already exists")
			return
		}
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, tenant)
}
