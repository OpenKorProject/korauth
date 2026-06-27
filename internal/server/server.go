package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/OpenKorProject/korauth/internal/handler"
	"github.com/OpenKorProject/korauth/internal/middleware"
	"github.com/OpenKorProject/korauth/internal/token"
)

type Handlers struct {
	Auth   *handler.AuthHandler
	JWKS   *handler.JWKSHandler
	User   *handler.UserHandler
	Tenant *handler.TenantHandler
	Role   *handler.RoleHandler
}

func New(tokenSvc *token.Service, h Handlers) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(slogMiddleware())

	// Dokümantasyon
	r.GET("/docs", handler.Docs)
	r.GET("/openapi.yaml", handler.OpenAPISpec)
	r.GET("/openapi.yml", handler.OpenAPISpec) // Scalar .yml arama yapıyor

	v1 := r.Group("/v1/auth")

	// Herkese açık
	v1.POST("/login", h.Auth.Login)
	v1.POST("/refresh", h.Auth.Refresh)
	v1.GET("/.well-known/jwks.json", h.JWKS.Get)

	// Kimlik doğrulama gerekli
	authed := v1.Group("", middleware.Auth(tokenSvc))
	authed.POST("/logout", h.Auth.Logout)
	authed.GET("/me", h.User.GetMe)
	authed.GET("/roles", h.Role.List)

	// Yalnız admin
	admin := authed.Group("", middleware.RequireRole("admin"))
	admin.GET("/users", h.User.List)
	admin.POST("/users", h.User.Create)
	admin.GET("/users/:id", h.User.Get)
	admin.PATCH("/users/:id", h.User.Update)
	admin.DELETE("/users/:id", h.User.Delete)
	admin.PUT("/users/:id/roles", h.User.AssignRoles)
	admin.GET("/tenants", h.Tenant.List)
	admin.POST("/tenants", h.Tenant.Create)

	return r
}

func slogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		reqID, _ := c.Get(middleware.CtxRequestID)
		slog.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"request_id", reqID,
		)
	}
}
