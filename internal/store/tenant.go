package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OpenKorProject/korauth/internal/model"
)

type TenantStore struct {
	db *pgxpool.Pool
}

func NewTenantStore(db *pgxpool.Pool) *TenantStore {
	return &TenantStore{db: db}
}

func (s *TenantStore) Create(ctx context.Context, name string) (*model.Tenant, error) {
	t := &model.Tenant{}
	err := s.db.QueryRow(ctx,
		`INSERT INTO auth.tenants (name) VALUES ($1)
		 RETURNING id, name, created_at, updated_at`,
		name,
	).Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return t, nil
}

func (s *TenantStore) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	t := &model.Tenant{}
	err := s.db.QueryRow(ctx,
		`SELECT id, name, created_at, updated_at FROM auth.tenants WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return t, nil
}

func (s *TenantStore) GetByName(ctx context.Context, name string) (*model.Tenant, error) {
	t := &model.Tenant{}
	err := s.db.QueryRow(ctx,
		`SELECT id, name, created_at, updated_at FROM auth.tenants WHERE name = $1`,
		name,
	).Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by name: %w", err)
	}
	return t, nil
}

func (s *TenantStore) List(ctx context.Context, page, perPage int) ([]model.Tenant, int64, error) {
	offset := (page - 1) * perPage

	var total int64
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM auth.tenants`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, name, created_at, updated_at FROM auth.tenants
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		perPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	tenants, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Tenant, error) {
		var t model.Tenant
		return t, row.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	})
	return tenants, total, err
}
