package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/OpenKorProject/korauth/internal/middleware"
	"github.com/OpenKorProject/korauth/internal/service"
)

type paginationResp struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

func reqID(c *gin.Context) string {
	id, _ := c.Get(middleware.CtxRequestID)
	s, _ := id.(string)
	return s
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{
		"code":       code,
		"message":    message,
		"request_id": reqID(c),
	}})
}

func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		respondError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, service.ErrConflict):
		respondError(c, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, service.ErrForbidden):
		respondError(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, service.ErrPasswordPolicy):
		respondError(c, http.StatusBadRequest, "PASSWORD_POLICY_VIOLATION", err.Error())
	default:
		respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}

func paginationQuery(c *gin.Context) (page, perPage int) {
	page = intQuery(c, "page", 1)
	perPage = intQuery(c, "per_page", 50)
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	return
}

func intQuery(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscan(v, &n); err != nil || n < 1 {
		return def
	}
	return n
}
