package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRole erişimi belirtilen rollerden en az birine sahip kullanıcıyla kısıtlar.
func RequireRole(allowed ...string) gin.HandlerFunc {
	set := make(map[string]bool, len(allowed))
	for _, r := range allowed {
		set[r] = true
	}
	return func(c *gin.Context) {
		rawRoles, _ := c.Get(CtxRoles)
		roles, _ := rawRoles.([]string)
		for _, r := range roles {
			if set[r] {
				c.Next()
				return
			}
		}
		reqID, _ := c.Get(CtxRequestID)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{
			"code":       "FORBIDDEN",
			"message":    "Insufficient permissions",
			"request_id": reqID,
		}})
	}
}
