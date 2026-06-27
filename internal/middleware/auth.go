package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/OpenKorProject/korauth/internal/token"
)

const (
	CtxUserID   = "user_id"
	CtxTenantID = "tenant_id"
	CtxRoles    = "roles"
)

func Auth(tokenSvc *token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, unauthorizedResp(c))
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		claims, err := tokenSvc.Parse(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, unauthorizedResp(c))
			return
		}

		c.Set(CtxUserID, claims.Subject)
		c.Set(CtxTenantID, claims.TenantID)
		c.Set(CtxRoles, claims.Roles)
		c.Next()
	}
}

func unauthorizedResp(c *gin.Context) gin.H {
	reqID, _ := c.Get(CtxRequestID)
	return gin.H{"error": gin.H{
		"code":       "UNAUTHORIZED",
		"message":    "Missing or invalid token",
		"request_id": reqID,
	}}
}
