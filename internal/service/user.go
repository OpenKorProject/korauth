package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/OpenKorProject/korauth/internal/model"
	"github.com/OpenKorProject/korauth/internal/password"
	"github.com/OpenKorProject/korauth/internal/store"
)

type UserService struct {
	users *store.UserStore
	roles *store.RoleStore
	auth  *AuthService
}

func NewUserService(users *store.UserStore, roles *store.RoleStore, auth *AuthService) *UserService {
	return &UserService{users: users, roles: roles, auth: auth}
}

func (s *UserService) Create(ctx context.Context, tenantID uuid.UUID, username, plain string, roleNames []string, firstName, lastName, email *string) (*model.User, error) {
	if err := password.Validate(plain); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPasswordPolicy, err)
	}

	valid, err := s.roles.IsValid(ctx, roleNames)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("%w: one or more role names are invalid", ErrNotFound)
	}

	hash, err := password.Hash(plain)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, tenantID, username, hash, false, firstName, lastName, email)
	if err != nil {
		if isPgUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}

	if err := s.users.SetRoles(ctx, user.ID, roleNames); err != nil {
		return nil, fmt.Errorf("set roles: %w", err)
	}

	return s.users.GetByID(ctx, user.ID)
}

func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

func (s *UserService) List(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]model.User, int64, error) {
	return s.users.List(ctx, tenantID, page, perPage)
}

type UpdateUserReq struct {
	Username  *string
	FirstName *string
	LastName  *string
	Email     *string
	Password  *string
}

func (s *UserService) Update(ctx context.Context, id uuid.UUID, req UpdateUserReq) (*model.User, error) {
	// Profil alanlarını güncelle
	if req.Username != nil || req.FirstName != nil || req.LastName != nil || req.Email != nil {
		if err := s.users.UpdateUser(ctx, id, req.Username, req.FirstName, req.LastName, req.Email); err != nil {
			if isPgUniqueViolation(err) {
				return nil, ErrConflict
			}
			return nil, err
		}
	}
	// Parola ayrı update
	if req.Password != nil {
		if err := password.Validate(*req.Password); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrPasswordPolicy, err)
		}
		hash, err := password.Hash(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		if err := s.users.UpdatePassword(ctx, id, hash, false); err != nil {
			return nil, err
		}
	}
	return s.GetByID(ctx, id)
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := s.users.SoftDelete(ctx, id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	// Refresh token'larını iptal et
	return s.auth.RevokeAllForUser(ctx, id)
}

func (s *UserService) AssignRoles(ctx context.Context, userID uuid.UUID, roleNames []string, callerID uuid.UUID) (*model.User, error) {
	valid, err := s.roles.IsValid(ctx, roleNames)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("%w: one or more role names are invalid", ErrNotFound)
	}

	// Admin kendi rolünü kaldıramaz
	if userID == callerID {
		hasAdmin := false
		for _, r := range roleNames {
			if r == "admin" {
				hasAdmin = true
				break
			}
		}
		if !hasAdmin {
			return nil, fmt.Errorf("%w: cannot remove your own admin role", ErrForbidden)
		}
	}

	if err := s.users.SetRoles(ctx, userID, roleNames); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, userID)
}

func isPgUniqueViolation(err error) bool {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		return pgErr.Code == "23505"
	}
	return false
}
