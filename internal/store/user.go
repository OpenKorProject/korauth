package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/OpenKorProject/korauth/internal/model"
)

type UserStore struct {
	db *pgxpool.Pool
}

func NewUserStore(db *pgxpool.Pool) *UserStore {
	return &UserStore{db: db}
}

// Roller string_agg ile birleştirilir; pgx native array'den daha taşınabilir.
const userCols = `
	u.id, u.tenant_id, u.username, u.first_name, u.last_name, u.email, u.password_hash, u.force_password_change,
	u.created_at, u.updated_at, u.deleted_at,
	COALESCE(string_agg(r.name, ',' ORDER BY r.name) FILTER (WHERE r.name IS NOT NULL), '') AS roles
`
const userJoin = `
	FROM auth.users u
	LEFT JOIN auth.user_roles ur ON ur.user_id = u.id
	LEFT JOIN auth.roles r ON r.id = ur.role_id
`

func scanUser(row pgx.Row) (*model.User, error) {
	u := &model.User{}
	var rolesStr string
	err := row.Scan(
		&u.ID, &u.TenantID, &u.Username, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash, &u.ForcePasswordChange,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
		&rolesStr,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if rolesStr != "" {
		u.Roles = strings.Split(rolesStr, ",")
	} else {
		u.Roles = []string{}
	}
	return u, nil
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+userCols+userJoin+`WHERE u.id = $1 AND u.deleted_at IS NULL GROUP BY u.id`,
		id,
	)
	u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (s *UserStore) GetByUsernameAndTenant(ctx context.Context, tenantID uuid.UUID, username string) (*model.User, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+userCols+userJoin+
			`WHERE u.tenant_id = $1 AND u.username = $2 AND u.deleted_at IS NULL GROUP BY u.id`,
		tenantID, username,
	)
	u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
}

func (s *UserStore) Create(ctx context.Context, tenantID uuid.UUID, username, passwordHash string, forcePasswordChange bool, firstName, lastName, email *string) (*model.User, error) {
	var id uuid.UUID
	err := s.db.QueryRow(ctx,
		`INSERT INTO auth.users (tenant_id, username, password_hash, force_password_change, first_name, last_name, email)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		tenantID, username, passwordHash, forcePasswordChange, firstName, lastName, email,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return s.GetByID(ctx, id)
}

func (s *UserStore) List(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]model.User, int64, error) {
	offset := (page - 1) * perPage

	var total int64
	if err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM auth.users WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	rows, err := s.db.Query(ctx,
		`SELECT `+userCols+userJoin+
			`WHERE u.tenant_id = $1 AND u.deleted_at IS NULL
			 GROUP BY u.id ORDER BY u.created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, perPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		u := model.User{}
		var rolesStr string
		if err := rows.Scan(
			&u.ID, &u.TenantID, &u.Username, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash, &u.ForcePasswordChange,
			&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
			&rolesStr,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		if rolesStr != "" {
			u.Roles = strings.Split(rolesStr, ",")
		} else {
			u.Roles = []string{}
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (s *UserStore) UpdateUser(ctx context.Context, id uuid.UUID, username, firstName, lastName, email *string) error {
	// Boş olan alanları skip et (CASE WHEN kullan)
	_, err := s.db.Exec(ctx,
		`UPDATE auth.users SET
			username = COALESCE($1, username),
			first_name = COALESCE($2, first_name),
			last_name = COALESCE($3, last_name),
			email = COALESCE($4, email),
			updated_at = NOW()
		 WHERE id = $5 AND deleted_at IS NULL`,
		username, firstName, lastName, email, id,
	)
	return err
}

func (s *UserStore) UpdatePassword(ctx context.Context, id uuid.UUID, hash string, forceChange bool) error {
	_, err := s.db.Exec(ctx,
		`UPDATE auth.users SET password_hash = $1, force_password_change = $2, updated_at = NOW()
		 WHERE id = $3 AND deleted_at IS NULL`,
		hash, forceChange, id,
	)
	return err
}

func (s *UserStore) SoftDelete(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := s.db.Exec(ctx,
		`UPDATE auth.users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return false, fmt.Errorf("soft delete user: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (s *UserStore) SetRoles(ctx context.Context, userID uuid.UUID, roleNames []string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM auth.user_roles WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("clear roles: %w", err)
	}
	for _, name := range roleNames {
		if _, err := tx.Exec(ctx,
			`INSERT INTO auth.user_roles (user_id, role_id)
			 SELECT $1, id FROM auth.roles WHERE name = $2`,
			userID, name,
		); err != nil {
			return fmt.Errorf("insert role %s: %w", name, err)
		}
	}
	return tx.Commit(ctx)
}

// AddRefreshTokenRef ve DeleteAllRefreshTokenRefs kullanıcı silme sırasında
// Redis token temizliği için kullanılır (bkz. service/auth.go).
func (s *UserStore) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1 AND deleted_at IS NULL)`, id,
	).Scan(&exists)
	return exists, err
}
