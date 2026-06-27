package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServiceName       string
	HTTPPort          string
	DatabaseURL       string
	DBSchema          string
	RedisURL          string
	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
	JWTIssuer         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	SeedAdminUsername string
	SeedAdminPassword string
	SeedTenantName    string
	LogLevel          string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var missing []string

	require := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := &Config{
		ServiceName:       getenv("SERVICE_NAME", "korauth"),
		HTTPPort:          getenv("HTTP_PORT", "8081"),
		DatabaseURL:       require("DATABASE_URL"),
		DBSchema:          getenv("DB_SCHEMA", "auth"),
		RedisURL:          require("REDIS_URL"),
		JWTPrivateKeyPath: require("JWT_PRIVATE_KEY_PATH"),
		JWTPublicKeyPath:  require("JWT_PUBLIC_KEY_PATH"),
		JWTIssuer:         getenv("JWT_ISSUER", "openkor-auth"),
		SeedAdminUsername: getenv("SEED_ADMIN_USERNAME", "admin"),
		SeedAdminPassword: require("SEED_ADMIN_PASSWORD"),
		SeedTenantName:    getenv("SEED_TENANT_NAME", "default"),
		LogLevel:          getenv("LOG_LEVEL", "info"),
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("required env vars not set: %s", strings.Join(missing, ", "))
	}

	var err error
	cfg.AccessTokenTTL, err = parseDuration("ACCESS_TOKEN_TTL", "15m")
	if err != nil {
		return nil, err
	}
	cfg.RefreshTokenTTL, err = parseDuration("REFRESH_TOKEN_TTL", "168h")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(key, fallback string) (time.Duration, error) {
	v := getenv(key, fallback)
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %s=%q: %w", key, v, err)
	}
	return d, nil
}
