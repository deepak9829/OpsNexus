package application

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/opsnexus/tenant-service/internal/domain"
	"github.com/opsnexus/tenant-service/internal/ports"
)

var slugRegexp = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ─── TenantService ────────────────────────────────────────────────────────────

type tenantServiceImpl struct {
	repo   ports.TenantRepository
	logger *zap.Logger
}

// NewTenantService constructs a TenantService backed by the given repository.
func NewTenantService(repo ports.TenantRepository, logger *zap.Logger) ports.TenantService {
	return &tenantServiceImpl{repo: repo, logger: logger}
}

func (s *tenantServiceImpl) CreateTenant(ctx context.Context, req ports.CreateTenantRequest) (*domain.Tenant, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}
	if !slugRegexp.MatchString(req.Slug) {
		return nil, fmt.Errorf("%w: slug must be lowercase alphanumeric with hyphens", domain.ErrInvalidInput)
	}

	// Check slug uniqueness.
	existing, err := s.repo.FindBySlug(ctx, req.Slug)
	if err != nil && err != domain.ErrTenantNotFound {
		return nil, fmt.Errorf("check slug: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrSlugTaken
	}

	plan := req.Plan
	if plan == "" {
		plan = domain.PlanFree
	}

	maxUsers := maxUsersForPlan(plan)

	tenant := &domain.Tenant{
		ID:     uuid.NewString(),
		Name:   req.Name,
		Slug:   req.Slug,
		Plan:   plan,
		Status: domain.TenantStatusActive,
		Settings: domain.TenantSettings{
			MaxUsers:       maxUsers,
			AllowedDomains: []string{},
			Features:       defaultFeatures(plan),
			NotificationPrefs: domain.NotificationPreferences{
				EmailEnabled: true,
				SMSEnabled:   false,
				InAppEnabled: true,
			},
		},
		MaxUsers:       maxUsers,
		AllowedDomains: []string{},
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	s.logger.Info("tenant created", zap.String("id", tenant.ID), zap.String("slug", tenant.Slug))
	return tenant, nil
}

func (s *tenantServiceImpl) GetTenant(ctx context.Context, id string) (*domain.Tenant, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *tenantServiceImpl) UpdateTenant(ctx context.Context, id string, req ports.UpdateTenantRequest) (*domain.Tenant, error) {
	t, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		if *req.Name == "" {
			return nil, fmt.Errorf("%w: name cannot be empty", domain.ErrInvalidInput)
		}
		t.Name = *req.Name
	}
	if req.Plan != nil {
		t.Plan = *req.Plan
		t.MaxUsers = maxUsersForPlan(*req.Plan)
		t.Settings.MaxUsers = t.MaxUsers
		t.Settings.Features = defaultFeatures(*req.Plan)
	}
	if req.Status != nil {
		t.Status = *req.Status
	}

	t.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	return t, nil
}

func (s *tenantServiceImpl) DeactivateTenant(ctx context.Context, id string) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Deactivate(ctx, id)
}

func (s *tenantServiceImpl) ListTenants(ctx context.Context, page, limit int) ([]*domain.Tenant, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, page, limit)
}

func (s *tenantServiceImpl) GetSettings(ctx context.Context, tenantID string) (*domain.TenantSettings, error) {
	t, err := s.repo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &t.Settings, nil
}

func (s *tenantServiceImpl) UpdateSettings(ctx context.Context, tenantID string, settings domain.TenantSettings) error {
	t, err := s.repo.FindByID(ctx, tenantID)
	if err != nil {
		return err
	}

	t.Settings = settings
	t.AllowedDomains = settings.AllowedDomains
	t.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, t)
}

// ─── OrganizationService ──────────────────────────────────────────────────────

type organizationServiceImpl struct {
	repo       ports.OrganizationRepository
	tenantRepo ports.TenantRepository
	logger     *zap.Logger
}

// NewOrganizationService constructs an OrganizationService.
func NewOrganizationService(repo ports.OrganizationRepository, tenantRepo ports.TenantRepository, logger *zap.Logger) ports.OrganizationService {
	return &organizationServiceImpl{repo: repo, tenantRepo: tenantRepo, logger: logger}
}

func (s *organizationServiceImpl) CreateOrganization(ctx context.Context, tenantID string, req ports.CreateOrgRequest) (*domain.Organization, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrInvalidInput)
	}
	// Verify tenant exists.
	if _, err := s.tenantRepo.FindByID(ctx, tenantID); err != nil {
		return nil, err
	}

	org := &domain.Organization{
		ID:        uuid.NewString(),
		TenantID:  tenantID,
		Name:      req.Name,
		Type:      req.Type,
		ParentID:  req.ParentID,
		Metadata:  map[string]any{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, org); err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}

	s.logger.Info("organization created", zap.String("id", org.ID), zap.String("tenantID", tenantID))
	return org, nil
}

func (s *organizationServiceImpl) GetOrganization(ctx context.Context, id string) (*domain.Organization, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *organizationServiceImpl) ListOrganizations(ctx context.Context, tenantID string) ([]*domain.Organization, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *organizationServiceImpl) UpdateOrganization(ctx context.Context, id string, req ports.UpdateOrgRequest) (*domain.Organization, error) {
	org, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		if *req.Name == "" {
			return nil, fmt.Errorf("%w: name cannot be empty", domain.ErrInvalidInput)
		}
		org.Name = *req.Name
	}
	if req.Type != nil {
		org.Type = *req.Type
	}
	if req.ParentID != nil {
		org.ParentID = req.ParentID
	}
	org.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, org); err != nil {
		return nil, fmt.Errorf("update organization: %w", err)
	}

	return org, nil
}

// ─── UserProfileService ───────────────────────────────────────────────────────

type userProfileServiceImpl struct {
	repo       ports.UserProfileRepository
	tenantRepo ports.TenantRepository
	logger     *zap.Logger
}

// NewUserProfileService constructs a UserProfileService.
func NewUserProfileService(repo ports.UserProfileRepository, tenantRepo ports.TenantRepository, logger *zap.Logger) ports.UserProfileService {
	return &userProfileServiceImpl{repo: repo, tenantRepo: tenantRepo, logger: logger}
}

func (s *userProfileServiceImpl) CreateProfile(ctx context.Context, req ports.CreateProfileRequest) (*domain.UserProfile, error) {
	if req.UserID == "" {
		return nil, fmt.Errorf("%w: userID is required", domain.ErrInvalidInput)
	}
	if req.TenantID == "" {
		return nil, fmt.Errorf("%w: tenantID is required", domain.ErrInvalidInput)
	}

	// Verify tenant exists.
	if _, err := s.tenantRepo.FindByID(ctx, req.TenantID); err != nil {
		return nil, err
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}
	locale := req.Locale
	if locale == "" {
		locale = "en"
	}

	profile := &domain.UserProfile{
		UserID:         req.UserID,
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		DisplayName:    req.DisplayName,
		AvatarURL:      "",
		Timezone:       timezone,
		Locale:         locale,
		Metadata:       map[string]any{},
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, profile); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}

	s.logger.Info("profile created", zap.String("userID", profile.UserID))
	return profile, nil
}

func (s *userProfileServiceImpl) GetProfile(ctx context.Context, userID string) (*domain.UserProfile, error) {
	return s.repo.FindByUserID(ctx, userID)
}

func (s *userProfileServiceImpl) UpdateProfile(ctx context.Context, userID string, req ports.UpdateProfileRequest) (*domain.UserProfile, error) {
	profile, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.DisplayName != nil {
		profile.DisplayName = *req.DisplayName
	}
	if req.AvatarURL != nil {
		profile.AvatarURL = *req.AvatarURL
	}
	if req.Timezone != nil {
		profile.Timezone = *req.Timezone
	}
	if req.Locale != nil {
		profile.Locale = *req.Locale
	}
	if req.OrganizationID != nil {
		profile.OrganizationID = req.OrganizationID
	}
	profile.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, profile); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}

	return profile, nil
}

func (s *userProfileServiceImpl) ListProfiles(ctx context.Context, tenantID string, page, limit int) ([]*domain.UserProfile, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.ListByTenant(ctx, tenantID, page, limit)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func maxUsersForPlan(plan domain.TenantPlan) int {
	switch plan {
	case domain.PlanPro:
		return 50
	case domain.PlanEnterprise:
		return 1000
	default:
		return 10
	}
}

func defaultFeatures(plan domain.TenantPlan) map[string]bool {
	base := map[string]bool{
		"audit_logs":       false,
		"sso":              false,
		"custom_roles":     false,
		"api_access":       true,
		"email_alerts":     true,
		"advanced_reports": false,
	}
	switch plan {
	case domain.PlanPro:
		base["audit_logs"] = true
		base["custom_roles"] = true
		base["advanced_reports"] = true
	case domain.PlanEnterprise:
		for k := range base {
			base[k] = true
		}
	}
	return base
}
