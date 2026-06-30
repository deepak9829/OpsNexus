package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/opsnexus/tenant-service/internal/application"
	"github.com/opsnexus/tenant-service/internal/domain"
	"github.com/opsnexus/tenant-service/internal/ports"
)

// ─── Mock implementations ─────────────────────────────────────────────────────

type mockTenantRepo struct{ mock.Mock }

func (m *mockTenantRepo) Create(ctx context.Context, t *domain.Tenant) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTenantRepo) FindByID(ctx context.Context, id string) (*domain.Tenant, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Tenant), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockTenantRepo) FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	args := m.Called(ctx, slug)
	if v := args.Get(0); v != nil {
		return v.(*domain.Tenant), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockTenantRepo) Update(ctx context.Context, t *domain.Tenant) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTenantRepo) Deactivate(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockTenantRepo) List(ctx context.Context, page, limit int) ([]*domain.Tenant, int64, error) {
	args := m.Called(ctx, page, limit)
	return args.Get(0).([]*domain.Tenant), args.Get(1).(int64), args.Error(2)
}

type mockOrgRepo struct{ mock.Mock }

func (m *mockOrgRepo) Create(ctx context.Context, o *domain.Organization) error {
	return m.Called(ctx, o).Error(0)
}
func (m *mockOrgRepo) FindByID(ctx context.Context, id string) (*domain.Organization, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Organization), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockOrgRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Organization, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*domain.Organization), args.Error(1)
}
func (m *mockOrgRepo) Update(ctx context.Context, o *domain.Organization) error {
	return m.Called(ctx, o).Error(0)
}
func (m *mockOrgRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

type mockProfileRepo struct{ mock.Mock }

func (m *mockProfileRepo) Create(ctx context.Context, p *domain.UserProfile) error {
	return m.Called(ctx, p).Error(0)
}
func (m *mockProfileRepo) FindByUserID(ctx context.Context, userID string) (*domain.UserProfile, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.UserProfile), args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockProfileRepo) Update(ctx context.Context, p *domain.UserProfile) error {
	return m.Called(ctx, p).Error(0)
}
func (m *mockProfileRepo) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.UserProfile, int64, error) {
	args := m.Called(ctx, tenantID, page, limit)
	return args.Get(0).([]*domain.UserProfile), args.Get(1).(int64), args.Error(2)
}

// ─── TenantService tests ──────────────────────────────────────────────────────

func TestCreateTenant_Success(t *testing.T) {
	repo := new(mockTenantRepo)
	svc := application.NewTenantService(repo, zap.NewNop())

	repo.On("FindBySlug", mock.Anything, "acme").Return(nil, domain.ErrTenantNotFound)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Tenant")).Return(nil)

	tenant, err := svc.CreateTenant(context.Background(), ports.CreateTenantRequest{
		Name: "Acme Corp",
		Slug: "acme",
		Plan: domain.PlanPro,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, tenant.ID)
	assert.Equal(t, "acme", tenant.Slug)
	assert.Equal(t, domain.PlanPro, tenant.Plan)
	assert.Equal(t, domain.TenantStatusActive, tenant.Status)
	assert.Equal(t, 50, tenant.MaxUsers)
	repo.AssertExpectations(t)
}

func TestCreateTenant_DuplicateSlug(t *testing.T) {
	repo := new(mockTenantRepo)
	svc := application.NewTenantService(repo, zap.NewNop())

	existing := &domain.Tenant{ID: "existing-id", Slug: "acme"}
	repo.On("FindBySlug", mock.Anything, "acme").Return(existing, nil)

	_, err := svc.CreateTenant(context.Background(), ports.CreateTenantRequest{
		Name: "Acme Corp",
		Slug: "acme",
		Plan: domain.PlanFree,
	})

	assert.ErrorIs(t, err, domain.ErrSlugTaken)
	repo.AssertExpectations(t)
}

func TestGetTenant_NotFound(t *testing.T) {
	repo := new(mockTenantRepo)
	svc := application.NewTenantService(repo, zap.NewNop())

	repo.On("FindByID", mock.Anything, "missing-id").Return(nil, domain.ErrTenantNotFound)

	_, err := svc.GetTenant(context.Background(), "missing-id")

	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
	repo.AssertExpectations(t)
}

func TestUpdateTenant_Success(t *testing.T) {
	repo := new(mockTenantRepo)
	svc := application.NewTenantService(repo, zap.NewNop())

	tenant := &domain.Tenant{
		ID:   "tenant-id",
		Name: "Old Name",
		Plan: domain.PlanFree,
	}
	repo.On("FindByID", mock.Anything, "tenant-id").Return(tenant, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Tenant")).Return(nil)

	newName := "New Name"
	newPlan := domain.PlanEnterprise
	updated, err := svc.UpdateTenant(context.Background(), "tenant-id", ports.UpdateTenantRequest{
		Name: &newName,
		Plan: &newPlan,
	})

	assert.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, domain.PlanEnterprise, updated.Plan)
	assert.Equal(t, 1000, updated.MaxUsers)
	repo.AssertExpectations(t)
}

// ─── OrganizationService tests ────────────────────────────────────────────────

func TestCreateOrganization_Success(t *testing.T) {
	orgRepo := new(mockOrgRepo)
	tenantRepo := new(mockTenantRepo)
	svc := application.NewOrganizationService(orgRepo, tenantRepo, zap.NewNop())

	tenantRepo.On("FindByID", mock.Anything, "tenant-1").Return(&domain.Tenant{ID: "tenant-1"}, nil)
	orgRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Organization")).Return(nil)

	org, err := svc.CreateOrganization(context.Background(), "tenant-1", ports.CreateOrgRequest{
		Name: "Engineering",
		Type: "department",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, org.ID)
	assert.Equal(t, "Engineering", org.Name)
	assert.Equal(t, "tenant-1", org.TenantID)
	tenantRepo.AssertExpectations(t)
	orgRepo.AssertExpectations(t)
}

// ─── UserProfileService tests ─────────────────────────────────────────────────

func TestCreateProfile_Success(t *testing.T) {
	profileRepo := new(mockProfileRepo)
	tenantRepo := new(mockTenantRepo)
	svc := application.NewUserProfileService(profileRepo, tenantRepo, zap.NewNop())

	tenantRepo.On("FindByID", mock.Anything, "tenant-1").Return(&domain.Tenant{ID: "tenant-1"}, nil)
	profileRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.UserProfile")).Return(nil)

	profile, err := svc.CreateProfile(context.Background(), ports.CreateProfileRequest{
		UserID:      "user-1",
		TenantID:    "tenant-1",
		DisplayName: "Alice",
		Timezone:    "America/New_York",
		Locale:      "en",
	})

	assert.NoError(t, err)
	assert.Equal(t, "user-1", profile.UserID)
	assert.Equal(t, "tenant-1", profile.TenantID)
	assert.Equal(t, "Alice", profile.DisplayName)
	assert.Equal(t, "America/New_York", profile.Timezone)
	tenantRepo.AssertExpectations(t)
	profileRepo.AssertExpectations(t)
}

func TestUpdateProfile_Success(t *testing.T) {
	profileRepo := new(mockProfileRepo)
	tenantRepo := new(mockTenantRepo)
	svc := application.NewUserProfileService(profileRepo, tenantRepo, zap.NewNop())

	existing := &domain.UserProfile{
		UserID:      "user-1",
		TenantID:    "tenant-1",
		DisplayName: "Old Name",
		Timezone:    "UTC",
		Locale:      "en",
	}
	profileRepo.On("FindByUserID", mock.Anything, "user-1").Return(existing, nil)
	profileRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.UserProfile")).Return(nil)

	newName := "New Name"
	newTZ := "Europe/London"
	updated, err := svc.UpdateProfile(context.Background(), "user-1", ports.UpdateProfileRequest{
		DisplayName: &newName,
		Timezone:    &newTZ,
	})

	assert.NoError(t, err)
	assert.Equal(t, "New Name", updated.DisplayName)
	assert.Equal(t, "Europe/London", updated.Timezone)
	profileRepo.AssertExpectations(t)
}
