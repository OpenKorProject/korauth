package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/OpenKorProject/korauth/internal/middleware"
	"github.com/OpenKorProject/korauth/internal/model"
	"github.com/OpenKorProject/korauth/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userIDStr, _ := c.Get(middleware.CtxUserID)
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token subject")
		return
	}
	user, err := h.svc.GetByID(c.Request.Context(), userID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) List(c *gin.Context) {
	tenantIDStr, _ := c.Get(middleware.CtxTenantID)

	// Admin başka tenant'ı sorgulayabilir
	tenantIDQuery := c.Query("tenant_id")
	var tenantID uuid.UUID
	if tenantIDQuery != "" {
		var err error
		tenantID, err = uuid.Parse(tenantIDQuery)
		if err != nil {
			respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid tenant_id")
			return
		}
	} else {
		tenantID, _ = uuid.Parse(tenantIDStr.(string))
	}

	page, perPage := paginationQuery(c)
	users, total, err := h.svc.List(c.Request.Context(), tenantID, page, perPage)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	if users == nil {
		users = []model.User{}
	}
	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"pagination": paginationResp{
			Page:    page,
			PerPage: perPage,
			Total:   total,
		},
	})
}

type createUserReq struct {
	Username  string   `json:"username" binding:"required,min=3,max=64"`
	Password  string   `json:"password" binding:"required,min=8"`
	TenantID  string   `json:"tenant_id" binding:"required,uuid"`
	FirstName *string  `json:"first_name"`
	LastName  *string  `json:"last_name"`
	Email     *string  `json:"email"`
	Roles     []string `json:"roles" binding:"required,min=1"`
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	tenantID, _ := uuid.Parse(req.TenantID)

	user, err := h.svc.Create(c.Request.Context(), tenantID, req.Username, req.Password, req.Roles, req.FirstName, req.LastName, req.Email)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return
	}
	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

type updateUserReq struct {
	Username  *string `json:"username"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	Password  *string `json:"password"`
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return
	}
	var req updateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Username == nil && req.FirstName == nil && req.LastName == nil && req.Email == nil && req.Password == nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "at least one field must be provided")
		return
	}

	user, err := h.svc.Update(c.Request.Context(), id, service.UpdateUserReq{
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		respondServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type assignRolesReq struct {
	Roles []string `json:"roles" binding:"required,min=1"`
}

func (h *UserHandler) AssignRoles(c *gin.Context) {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return
	}
	var req assignRolesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	callerIDStr, _ := c.Get(middleware.CtxUserID)
	callerID, _ := uuid.Parse(callerIDStr.(string))

	user, err := h.svc.AssignRoles(c.Request.Context(), id, req.Roles, callerID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

func parseUUIDParam(c *gin.Context, param string) (uuid.UUID, error) {
	s := c.Param(param)
	id, err := uuid.Parse(s)
	if err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid "+param)
		return uuid.UUID{}, err
	}
	return id, nil
}
