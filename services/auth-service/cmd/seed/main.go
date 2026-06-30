package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	mysqladapter "github.com/opsnexus/auth-service/internal/adapters/mysql"
	"github.com/opsnexus/auth-service/internal/config"
	"github.com/opsnexus/auth-service/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	db, err := mysqladapter.NewConnection(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	if err := mysqladapter.AutoMigrate(db); err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	systemTenantID := "00000000-0000-0000-0000-000000000001"

	roleRepo := mysqladapter.NewRoleRepository(db)
	userRepo := mysqladapter.NewUserRepository(db)

	// Create permissions
	allPerms := []domain.Permission{
		{ID: uuid.New().String(), Resource: "*", Action: "admin"},
		{ID: uuid.New().String(), Resource: "users", Action: "read"},
		{ID: uuid.New().String(), Resource: "users", Action: "write"},
		{ID: uuid.New().String(), Resource: "users", Action: "delete"},
		{ID: uuid.New().String(), Resource: "roles", Action: "read"},
		{ID: uuid.New().String(), Resource: "roles", Action: "write"},
		{ID: uuid.New().String(), Resource: "roles", Action: "delete"},
		{ID: uuid.New().String(), Resource: "tenants", Action: "read"},
		{ID: uuid.New().String(), Resource: "tenants", Action: "write"},
	}

	rwPerms := []domain.Permission{
		{ID: uuid.New().String(), Resource: "users", Action: "read"},
		{ID: uuid.New().String(), Resource: "users", Action: "write"},
		{ID: uuid.New().String(), Resource: "roles", Action: "read"},
	}

	readPerms := []domain.Permission{
		{ID: uuid.New().String(), Resource: "users", Action: "read"},
		{ID: uuid.New().String(), Resource: "roles", Action: "read"},
		{ID: uuid.New().String(), Resource: "tenants", Action: "read"},
	}

	now := time.Now().UTC()

	superAdminRole := &domain.Role{
		ID:          uuid.New().String(),
		TenantID:    systemTenantID,
		Name:        "super_admin",
		Description: "Full system access",
		Permissions: allPerms,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	adminRole := &domain.Role{
		ID:          uuid.New().String(),
		TenantID:    systemTenantID,
		Name:        "admin",
		Description: "Administrative access with read/write permissions",
		Permissions: rwPerms,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	viewerRole := &domain.Role{
		ID:          uuid.New().String(),
		TenantID:    systemTenantID,
		Name:        "viewer",
		Description: "Read-only access",
		Permissions: readPerms,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	for _, role := range []*domain.Role{superAdminRole, adminRole, viewerRole} {
		existing, _ := roleRepo.FindByName(ctx, systemTenantID, role.Name)
		if existing != nil {
			fmt.Printf("Role %s already exists, skipping.\n", role.Name)
			continue
		}
		if err := roleRepo.Create(ctx, role); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create role %s: %v\n", role.Name, err)
			os.Exit(1)
		}
		fmt.Printf("Created role: %s\n", role.Name)
	}

	// Create test user
	hash, err := bcrypt.GenerateFromPassword([]byte("Admin123!"), 12)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to hash password: %v\n", err)
		os.Exit(1)
	}

	testUser := &domain.User{
		ID:           uuid.New().String(),
		TenantID:     systemTenantID,
		Email:        "admin@opsnexus.com",
		PasswordHash: string(hash),
		FirstName:    "System",
		LastName:     "Admin",
		Status:       domain.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	existing, _ := userRepo.FindByTenantAndEmail(ctx, systemTenantID, testUser.Email)
	if existing != nil {
		fmt.Println("Test user already exists, skipping.")
	} else {
		if err := userRepo.Create(ctx, testUser); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create test user: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created test user: %s\n", testUser.Email)

		// Assign super_admin role
		sa, err := roleRepo.FindByName(ctx, systemTenantID, "super_admin")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to find super_admin role: %v\n", err)
			os.Exit(1)
		}
		if err := roleRepo.AssignToUser(ctx, testUser.ID, sa.ID); err != nil {
			fmt.Fprintf(os.Stderr, "failed to assign role: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Assigned super_admin role to test user.")
	}

	fmt.Println("Seeding complete.")
}
