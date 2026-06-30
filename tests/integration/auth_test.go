//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ---------------------------------------------------------------------------
// GORM models (mirror auth-service domain models)
// ---------------------------------------------------------------------------

type User struct {
	ID        string    `gorm:"primaryKey;type:char(36)"`
	TenantID  string    `gorm:"type:char(36);not null;index:idx_tenant_email,unique"`
	Email     string    `gorm:"type:varchar(255);not null;index:idx_tenant_email,unique"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Password  string    `gorm:"type:varchar(255);not null"`
	Status    string    `gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type RefreshToken struct {
	ID        string    `gorm:"primaryKey;type:char(36)"`
	UserID    string    `gorm:"type:char(36);not null;index"`
	TenantID  string    `gorm:"type:char(36);not null;index"`
	TokenHash string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"not null;default:false"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type Role struct {
	ID          string    `gorm:"primaryKey;type:char(36)"`
	TenantID    string    `gorm:"type:char(36);not null;index:idx_tenant_role,unique"`
	Name        string    `gorm:"type:varchar(128);not null;index:idx_tenant_role,unique"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

type UserRole struct {
	UserID    string    `gorm:"primaryKey;type:char(36)"`
	RoleID    string    `gorm:"primaryKey;type:char(36)"`
	TenantID  string    `gorm:"type:char(36);not null;index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// ---------------------------------------------------------------------------
// Package-level DB handle
// ---------------------------------------------------------------------------

var testDB *gorm.DB

// TestMain sets up the DB connection and auto-migrates tables for all
// integration tests in this package.
//
// Required env var: TEST_MYSQL_DSN
// Example: auth_user:auth_pass@tcp(localhost:3306)/auth_db?charset=utf8mb4&parseTime=True&loc=UTC
func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		dsn = "auth_user:auth_pass@tcp(localhost:3306)/auth_db?charset=utf8mb4&parseTime=True&loc=UTC"
	}

	var err error
	testDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: cannot connect to MySQL at DSN %q: %v\n", dsn, err)
		os.Exit(1)
	}

	// Auto-migrate the test schema
	if err = testDB.AutoMigrate(&User{}, &RefreshToken{}, &Role{}, &UserRole{}); err != nil {
		fmt.Fprintf(os.Stderr, "integration: AutoMigrate failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newUserID() string  { return uuid.New().String() }
func newTenantID() string { return uuid.New().String() }

func createTestUser(t *testing.T, db *gorm.DB, tenantID, email string) User {
	t.Helper()
	u := User{
		ID:       newUserID(),
		TenantID: tenantID,
		Email:    email,
		Name:     "Test User",
		Password: "$2a$10$hashedpasswordplaceholder",
		Status:   "active",
	}
	require.NoError(t, db.WithContext(context.Background()).Create(&u).Error)
	t.Cleanup(func() {
		db.Unscoped().Delete(&u)
	})
	return u
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestUserRepository_CreateAndFind creates a user row, then retrieves it by
// email and verifies every field round-trips correctly.
func TestUserRepository_CreateAndFind(t *testing.T) {
	ctx := context.Background()
	tenantID := newTenantID()
	email := fmt.Sprintf("create-find-%s@example.com", uuid.New().String()[:8])

	// Create
	want := User{
		ID:       newUserID(),
		TenantID: tenantID,
		Email:    email,
		Name:     "Alice Tester",
		Password: "$2a$10$examplehashedpassword",
		Status:   "active",
	}
	require.NoError(t, testDB.WithContext(ctx).Create(&want).Error)
	t.Cleanup(func() { testDB.Unscoped().Delete(&want) })

	// Find by email
	var got User
	err := testDB.WithContext(ctx).
		Where("email = ? AND tenant_id = ?", email, tenantID).
		First(&got).Error
	require.NoError(t, err)

	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.TenantID, got.TenantID)
	assert.Equal(t, want.Email, got.Email)
	assert.Equal(t, want.Name, got.Name)
	assert.Equal(t, want.Password, got.Password)
	assert.Equal(t, want.Status, got.Status)
	assert.False(t, got.CreatedAt.IsZero(), "CreatedAt should be set by DB")
	assert.False(t, got.UpdatedAt.IsZero(), "UpdatedAt should be set by DB")
}

// TestUserRepository_FindByTenantAndEmail verifies that two tenants can have
// users with the same email address without colliding, and that a lookup
// scoped to one tenant does not return the other tenant's user.
func TestUserRepository_FindByTenantAndEmail(t *testing.T) {
	ctx := context.Background()
	sharedEmail := fmt.Sprintf("shared-%s@example.com", uuid.New().String()[:8])

	tenantA := newTenantID()
	tenantB := newTenantID()

	userA := createTestUser(t, testDB, tenantA, sharedEmail)
	userB := createTestUser(t, testDB, tenantB, sharedEmail)

	// Query for tenant A
	var gotA User
	err := testDB.WithContext(ctx).
		Where("email = ? AND tenant_id = ?", sharedEmail, tenantA).
		First(&gotA).Error
	require.NoError(t, err)
	assert.Equal(t, userA.ID, gotA.ID, "tenant A query must return tenant A user")

	// Query for tenant B
	var gotB User
	err = testDB.WithContext(ctx).
		Where("email = ? AND tenant_id = ?", sharedEmail, tenantB).
		First(&gotB).Error
	require.NoError(t, err)
	assert.Equal(t, userB.ID, gotB.ID, "tenant B query must return tenant B user")

	// Querying tenant A must NOT return tenant B's row
	assert.NotEqual(t, gotA.ID, gotB.ID)
}

// TestUserRepository_UpdateStatus creates a user with status "active", updates
// it to "suspended", and verifies the new value is persisted.
func TestUserRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	tenantID := newTenantID()
	email := fmt.Sprintf("status-%s@example.com", uuid.New().String()[:8])

	u := createTestUser(t, testDB, tenantID, email)
	assert.Equal(t, "active", u.Status)

	// Update status
	err := testDB.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", u.ID).
		Update("status", "suspended").Error
	require.NoError(t, err)

	// Re-fetch and verify
	var updated User
	require.NoError(t, testDB.WithContext(ctx).First(&updated, "id = ?", u.ID).Error)
	assert.Equal(t, "suspended", updated.Status)
	assert.True(t, updated.UpdatedAt.After(u.UpdatedAt) || updated.UpdatedAt.Equal(u.UpdatedAt),
		"UpdatedAt should not move backwards")
}

// TestRefreshTokenRepository_CreateAndRevoke creates a refresh token, verifies
// it can be fetched by its hash, then revokes it and verifies the revoked flag.
func TestRefreshTokenRepository_CreateAndRevoke(t *testing.T) {
	ctx := context.Background()
	tenantID := newTenantID()
	userID := newUserID()
	tokenHash := fmt.Sprintf("sha256-hash-%s", uuid.New().String())

	token := RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TenantID:  tenantID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
	}
	require.NoError(t, testDB.WithContext(ctx).Create(&token).Error)
	t.Cleanup(func() { testDB.Unscoped().Delete(&token) })

	// Verify found and not revoked
	var found RefreshToken
	err := testDB.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&found).Error
	require.NoError(t, err)
	assert.Equal(t, token.ID, found.ID)
	assert.False(t, found.Revoked, "token should not be revoked yet")

	// Revoke
	err = testDB.WithContext(ctx).
		Model(&RefreshToken{}).
		Where("id = ?", token.ID).
		Update("revoked", true).Error
	require.NoError(t, err)

	// Verify revoked
	var revoked RefreshToken
	require.NoError(t, testDB.WithContext(ctx).First(&revoked, "id = ?", token.ID).Error)
	assert.True(t, revoked.Revoked, "token should now be revoked")
}

// TestRoleRepository_AssignAndRevoke creates a role, assigns it to a user,
// verifies the assignment, then revokes it and verifies removal.
func TestRoleRepository_AssignAndRevoke(t *testing.T) {
	ctx := context.Background()
	tenantID := newTenantID()
	email := fmt.Sprintf("role-%s@example.com", uuid.New().String()[:8])

	user := createTestUser(t, testDB, tenantID, email)

	role := Role{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        fmt.Sprintf("engineer-%s", uuid.New().String()[:6]),
		Description: "Integration test role",
	}
	require.NoError(t, testDB.WithContext(ctx).Create(&role).Error)
	t.Cleanup(func() { testDB.Unscoped().Delete(&role) })

	// Assign role to user
	assignment := UserRole{
		UserID:   user.ID,
		RoleID:   role.ID,
		TenantID: tenantID,
	}
	require.NoError(t, testDB.WithContext(ctx).Create(&assignment).Error)
	t.Cleanup(func() { testDB.Unscoped().Delete(&assignment) })

	// Verify user has the role
	var count int64
	err := testDB.WithContext(ctx).
		Model(&UserRole{}).
		Where("user_id = ? AND role_id = ? AND tenant_id = ?", user.ID, role.ID, tenantID).
		Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "user should have the assigned role")

	// Revoke (hard delete the assignment)
	err = testDB.WithContext(ctx).
		Where("user_id = ? AND role_id = ? AND tenant_id = ?", user.ID, role.ID, tenantID).
		Delete(&UserRole{}).Error
	require.NoError(t, err)

	// Verify removed
	var afterRevoke int64
	err = testDB.WithContext(ctx).
		Model(&UserRole{}).
		Where("user_id = ? AND role_id = ? AND tenant_id = ?", user.ID, role.ID, tenantID).
		Count(&afterRevoke).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), afterRevoke, "role assignment should be removed after revoke")
}
