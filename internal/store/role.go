package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OpenKorProject/korauth/internal/model"
)

type RoleStore struct {
	db *pgxpool.Pool
}

func NewRoleStore(db *pgxpool.Pool) *RoleStore {
	return &RoleStore{db: db}
}

func (s *RoleStore) List(ctx context.Context) ([]model.Role, error) {
	rows, err := s.db.Query(ctx, `SELECT id, name FROM auth.roles ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	roles, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Role, error) {
		var r model.Role
		return r, row.Scan(&r.ID, &r.Name)
	})
	return roles, err
}

func (s *RoleStore) IsValid(ctx context.Context, names []string) (bool, error) {
	var count int
	err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM auth.roles WHERE name = ANY($1::text[])`,
		names,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("validate roles: %w", err)
	}
	return count == len(names), nil
}
