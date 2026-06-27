package seed

import (
	"context"
	"log/slog"

	"github.com/OpenKorProject/korauth/internal/password"
	"github.com/OpenKorProject/korauth/internal/store"
)

func Run(ctx context.Context, tenants *store.TenantStore, users *store.UserStore, tenantName, adminUsername, adminPassword string) error {
	// Seed tenant
	tenant, err := tenants.GetByName(ctx, tenantName)
	if err != nil {
		return err
	}
	if tenant == nil {
		tenant, err = tenants.Create(ctx, tenantName)
		if err != nil {
			return err
		}
		slog.Info("seed: tenant created", "name", tenantName, "id", tenant.ID)
	}

	// Seed admin kullanıcı
	existing, err := users.GetByUsernameAndTenant(ctx, tenant.ID, adminUsername)
	if err != nil {
		return err
	}
	if existing != nil {
		slog.Info("seed: admin user already exists", "username", adminUsername)
		return nil
	}

	hash, err := password.Hash(adminPassword)
	if err != nil {
		return err
	}

	// force_password_change=true: ilk girişte parola değişimi zorunlu
	user, err := users.Create(ctx, tenant.ID, adminUsername, hash, true, nil, nil, nil)
	if err != nil {
		return err
	}

	if err := users.SetRoles(ctx, user.ID, []string{"admin"}); err != nil {
		return err
	}

	slog.Info("seed: admin user created", "username", adminUsername, "tenant_id", tenant.ID)
	return nil
}
