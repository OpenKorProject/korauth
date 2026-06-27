package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/OpenKorProject/korauth/internal/config"
	appdb "github.com/OpenKorProject/korauth/internal/db"
	"github.com/OpenKorProject/korauth/internal/password"
	"github.com/OpenKorProject/korauth/internal/store"
	"github.com/google/uuid"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "reset-admin-password":
		resetAdminPassword()
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func resetAdminPassword() {
	fs := flag.NewFlagSet("reset-admin-password", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: korauth-cli reset-admin-password <tenant-id> <new-password>\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  korauth-cli reset-admin-password 6ec83570-ee9d-46b1-8a8c-f52a01ce987d newpass123\n")
	}
	fs.Parse(os.Args[2:])

	args := fs.Args()
	if len(args) != 2 {
		fs.Usage()
		os.Exit(1)
	}

	tenantIDStr, newPassword := args[0], args[1]

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid tenant ID: %v\n", err)
		os.Exit(1)
	}

	if err := password.Validate(newPassword); err != nil {
		fmt.Fprintf(os.Stderr, "password validation failed: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := appdb.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	userStore := store.NewUserStore(db)

	user, err := userStore.GetByUsernameAndTenant(ctx, tenantID, "admin")
	if err != nil {
		fmt.Fprintf(os.Stderr, "user lookup failed: %v\n", err)
		os.Exit(1)
	}
	if user == nil {
		fmt.Fprintf(os.Stderr, "admin user not found in tenant %s\n", tenantID)
		os.Exit(1)
	}

	hashedPassword, err := password.Hash(newPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "password hashing failed: %v\n", err)
		os.Exit(1)
	}

	if err := userStore.UpdatePassword(ctx, user.ID, hashedPassword, false); err != nil {
		fmt.Fprintf(os.Stderr, "password update failed: %v\n", err)
		os.Exit(1)
	}

	slog.Info("admin password reset successfully", "user_id", user.ID, "tenant_id", tenantID)
	fmt.Printf("✓ Admin password reset successfully\n")
	fmt.Printf("  User ID:   %s\n", user.ID)
	fmt.Printf("  Tenant ID: %s\n", tenantID)
}

func usage() {
	fmt.Fprintf(os.Stderr, "korauth-cli — Admin utility for korauth\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  reset-admin-password <tenant-id> <new-password>\n")
	fmt.Fprintf(os.Stderr, "                       Reset admin user password\n")
	fmt.Fprintf(os.Stderr, "  help                 Show this message\n\n")
	fmt.Fprintf(os.Stderr, "Example:\n")
	fmt.Fprintf(os.Stderr, "  korauth-cli reset-admin-password 6ec83570-ee9d-46b1-8a8c-f52a01ce987d newpass123\n")
}
