package service

import (
	"context"

	"github.com/OpenKorProject/korauth/internal/model"
	"github.com/OpenKorProject/korauth/internal/store"
)

type RoleService struct {
	roles *store.RoleStore
}

func NewRoleService(roles *store.RoleStore) *RoleService {
	return &RoleService{roles: roles}
}

func (s *RoleService) List(ctx context.Context) ([]model.Role, error) {
	return s.roles.List(ctx)
}
