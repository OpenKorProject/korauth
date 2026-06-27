package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/OpenKorProject/korauth/internal/model"
	"github.com/OpenKorProject/korauth/internal/store"
)

type TenantService struct {
	tenants *store.TenantStore
}

func NewTenantService(tenants *store.TenantStore) *TenantService {
	return &TenantService{tenants: tenants}
}

func (s *TenantService) Create(ctx context.Context, name string) (*model.Tenant, error) {
	existing, err := s.tenants.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrConflict
	}
	t, err := s.tenants.Create(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return t, nil
}

func (s *TenantService) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	t, err := s.tenants.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *TenantService) List(ctx context.Context, page, perPage int) ([]model.Tenant, int64, error) {
	return s.tenants.List(ctx, page, perPage)
}
