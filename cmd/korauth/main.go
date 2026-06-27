package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/OpenKorProject/korauth/internal/config"
	appdb "github.com/OpenKorProject/korauth/internal/db"
	"github.com/OpenKorProject/korauth/internal/handler"
	appredis "github.com/OpenKorProject/korauth/internal/redis"
	"github.com/OpenKorProject/korauth/internal/seed"
	"github.com/OpenKorProject/korauth/internal/server"
	"github.com/OpenKorProject/korauth/internal/service"
	"github.com/OpenKorProject/korauth/internal/store"
	"github.com/OpenKorProject/korauth/internal/token"
)

func main() {
	setupLogger()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// PostgreSQL
	db, err := appdb.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database connected")

	// Migration
	if err := appdb.Migrate(ctx, db); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// Redis
	rdb, err := appredis.New(ctx, cfg.RedisURL)
	if err != nil {
		slog.Error("redis connection failed", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()
	slog.Info("redis connected")

	// Token servisi
	tokenSvc, err := token.NewService(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath, cfg.JWTIssuer, cfg.AccessTokenTTL)
	if err != nil {
		slog.Error("token service init failed", "err", err)
		os.Exit(1)
	}

	// Store'lar
	tenantStore := store.NewTenantStore(db)
	userStore := store.NewUserStore(db)
	roleStore := store.NewRoleStore(db)

	// Seed
	if err := seed.Run(ctx, tenantStore, userStore, cfg.SeedTenantName, cfg.SeedAdminUsername, cfg.SeedAdminPassword); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}

	// Credentials banner
	tenant, _ := tenantStore.GetByName(ctx, cfg.SeedTenantName)
	line := strings.Repeat("=", 50)
	fmt.Printf("\n%s\n", line)
	fmt.Println("✓ KORAUTH INITIALIZED")
	fmt.Printf("%s\n", line)
	fmt.Printf("Admin Username: %s\n", cfg.SeedAdminUsername)
	fmt.Printf("Admin Password: %s\n", cfg.SeedAdminPassword)
	if tenant != nil {
		fmt.Printf("Tenant ID:      %s\n", tenant.ID)
	}
	fmt.Printf("%s\n", line)
	fmt.Printf("Login: POST http://localhost:%s/v1/auth/login\n", cfg.HTTPPort)
	fmt.Printf("Docs:  http://localhost:%s/docs\n", cfg.HTTPPort)
	fmt.Printf("%s\n\n", line)

	// Servisler
	authSvc := service.NewAuthService(userStore, tokenSvc, rdb, cfg.RefreshTokenTTL)
	userSvc := service.NewUserService(userStore, roleStore, authSvc)
	tenantSvc := service.NewTenantService(tenantStore)
	roleSvc := service.NewRoleService(roleStore)

	// Handler'lar
	h := server.Handlers{
		Auth:   handler.NewAuthHandler(authSvc),
		JWKS:   handler.NewJWKSHandler(tokenSvc),
		User:   handler.NewUserHandler(userSvc),
		Tenant: handler.NewTenantHandler(tenantSvc),
		Role:   handler.NewRoleHandler(roleSvc),
	}

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      server.New(tokenSvc, h),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("korauth started", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

func setupLogger() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
}
