package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/opsnexus/tenant-service/internal/adapters/mysql"
	"github.com/opsnexus/tenant-service/internal/application"
	"github.com/opsnexus/tenant-service/internal/config"
	"github.com/opsnexus/tenant-service/internal/domain"
	"github.com/opsnexus/tenant-service/internal/ports"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load config", zap.Error(err))
	}

	db, err := mysql.NewDB(cfg)
	if err != nil {
		logger.Fatal("connect to database", zap.Error(err))
	}

	// Wire up repositories and services.
	tenantRepo := mysql.NewTenantRepository(db)
	orgRepo := mysql.NewOrganizationRepository(db)
	profileRepo := mysql.NewUserProfileRepository(db)

	tenantSvc := application.NewTenantService(tenantRepo, logger)
	orgSvc := application.NewOrganizationService(orgRepo, tenantRepo, logger)
	profileSvc := application.NewUserProfileService(profileRepo, tenantRepo, logger)

	ctx := context.Background()

	// ── Seed tenant ───────────────────────────────────────────────────────────
	logger.Info("seeding demo tenant…")
	tenant, err := tenantSvc.CreateTenant(ctx, ports.CreateTenantRequest{
		Name: "Demo Corp",
		Slug: "demo",
		Plan: domain.PlanEnterprise,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create demo tenant: %v\n", err)
		os.Exit(1)
	}
	logger.Info("demo tenant created", zap.String("id", tenant.ID))

	// ── Seed organizations ────────────────────────────────────────────────────
	orgNames := []struct {
		name string
		kind string
	}{
		{"Engineering", "department"},
		{"Marketing", "department"},
		{"Operations", "department"},
	}

	orgIDs := make(map[string]string)
	for _, o := range orgNames {
		org, err := orgSvc.CreateOrganization(ctx, tenant.ID, ports.CreateOrgRequest{
			Name: o.name,
			Type: o.kind,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "create org %s: %v\n", o.name, err)
			os.Exit(1)
		}
		orgIDs[o.name] = org.ID
		logger.Info("organization created", zap.String("name", o.name), zap.String("id", org.ID))
	}

	// ── Seed user profiles ────────────────────────────────────────────────────
	engID := orgIDs["Engineering"]
	mktID := orgIDs["Marketing"]

	users := []ports.CreateProfileRequest{
		{
			UserID:         "seed-user-001",
			TenantID:       tenant.ID,
			OrganizationID: &engID,
			DisplayName:    "Alice Engineer",
			Timezone:       "America/New_York",
			Locale:         "en",
		},
		{
			UserID:         "seed-user-002",
			TenantID:       tenant.ID,
			OrganizationID: &mktID,
			DisplayName:    "Bob Marketer",
			Timezone:       "America/Los_Angeles",
			Locale:         "en",
		},
	}

	for _, u := range users {
		profile, err := profileSvc.CreateProfile(ctx, u)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create profile for %s: %v\n", u.UserID, err)
			os.Exit(1)
		}
		logger.Info("profile created", zap.String("userID", profile.UserID), zap.String("displayName", profile.DisplayName))
	}

	logger.Info("seed completed successfully")
}
