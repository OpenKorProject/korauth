package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/OpenKorProject/korauth/internal/model"
	"github.com/OpenKorProject/korauth/internal/password"
	"github.com/OpenKorProject/korauth/internal/store"
	"github.com/OpenKorProject/korauth/internal/token"
)

const (
	bruteForceLimit  = 5
	bruteForceTTL    = 15 * time.Minute
	bruteKeyPrefix   = "brute:"
	refreshKeyPrefix = "refresh:"
	userTokensPrefix = "user_tokens:"
)

type refreshPayload struct {
	UserID    string   `json:"user_id"`
	TenantID  string   `json:"tenant_id"`
	Roles     []string `json:"roles"`
	IssuedAt  int64    `json:"issued_at"`
	ExpiresAt int64    `json:"expires_at"`
}

type AuthService struct {
	users      *store.UserStore
	tokenSvc   *token.Service
	rdb        *redis.Client
	refreshTTL time.Duration
}

func NewAuthService(users *store.UserStore, tokenSvc *token.Service, rdb *redis.Client, refreshTTL time.Duration) *AuthService {
	return &AuthService{users: users, tokenSvc: tokenSvc, rdb: rdb, refreshTTL: refreshTTL}
}

func (s *AuthService) Login(ctx context.Context, tenantID uuid.UUID, username, plain string) (*model.TokenPair, error) {
	bruteKey := fmt.Sprintf("%s%s:%s", bruteKeyPrefix, tenantID, username)

	count, err := s.rdb.Get(ctx, bruteKey).Int()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("check brute force: %w", err)
	}
	if count >= bruteForceLimit {
		return nil, ErrAccountLocked
	}

	user, err := s.users.GetByUsernameAndTenant(ctx, tenantID, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		s.incBrute(ctx, bruteKey)
		return nil, ErrInvalidCredentials
	}

	ok, err := password.Verify(plain, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		s.incBrute(ctx, bruteKey)
		return nil, ErrInvalidCredentials
	}

	s.rdb.Del(ctx, bruteKey)
	return s.makePair(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	key := refreshKeyPrefix + refreshToken
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	var rp refreshPayload
	if err := json.Unmarshal(data, &rp); err != nil {
		return nil, fmt.Errorf("unmarshal refresh payload: %w", err)
	}

	// Token rotation: eskiyi sil
	s.rdb.Del(ctx, key)
	s.rdb.SRem(ctx, userTokensPrefix+rp.UserID, refreshToken)

	userID, err := uuid.Parse(rp.UserID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}
	tenantID, err := uuid.Parse(rp.TenantID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Güncel kullanıcı durumu (silinmiş veya rol değişmiş olabilir)
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenantID {
		return nil, ErrInvalidRefreshToken
	}

	return s.makePair(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	key := refreshKeyPrefix + refreshToken
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil // zaten yok, idempotent
	}
	if err != nil {
		return fmt.Errorf("get refresh token: %w", err)
	}

	var rp refreshPayload
	if err := json.Unmarshal(data, &rp); err == nil {
		s.rdb.SRem(ctx, userTokensPrefix+rp.UserID, refreshToken)
	}

	return s.rdb.Del(ctx, key).Err()
}

// RevokeAllForUser kullanıcı silindiğinde tüm refresh token'larını iptal eder.
func (s *AuthService) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	setKey := userTokensPrefix + userID.String()
	jtis, err := s.rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return fmt.Errorf("get user tokens: %w", err)
	}
	for _, jti := range jtis {
		s.rdb.Del(ctx, refreshKeyPrefix+jti)
	}
	return s.rdb.Del(ctx, setKey).Err()
}

func (s *AuthService) makePair(ctx context.Context, user *model.User) (*model.TokenPair, error) {
	accessToken, err := s.tokenSvc.Generate(user.ID, user.TenantID, user.Roles)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	jti, err := newJTI()
	if err != nil {
		return nil, fmt.Errorf("generate jti: %w", err)
	}

	now := time.Now()
	rp := refreshPayload{
		UserID:    user.ID.String(),
		TenantID:  user.TenantID.String(),
		Roles:     user.Roles,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(s.refreshTTL).Unix(),
	}
	data, err := json.Marshal(rp)
	if err != nil {
		return nil, fmt.Errorf("marshal refresh payload: %w", err)
	}

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, refreshKeyPrefix+jti, data, s.refreshTTL)
	pipe.SAdd(ctx, userTokensPrefix+user.ID.String(), jti)
	pipe.Expire(ctx, userTokensPrefix+user.ID.String(), s.refreshTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: jti,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.tokenSvc.AccessTTL().Seconds()),
	}, nil
}

func (s *AuthService) incBrute(ctx context.Context, key string) {
	n, _ := s.rdb.Incr(ctx, key).Result()
	if n == 1 {
		s.rdb.Expire(ctx, key, bruteForceTTL)
	}
}

func newJTI() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
