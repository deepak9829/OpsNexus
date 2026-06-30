package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/opsnexus/auth-service/internal/domain"
	"github.com/opsnexus/auth-service/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// --- Mock Repositories ---

type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	for _, u := range m.users {
		if u.Email == user.Email && u.TenantID == user.TenantID {
			return domain.ErrUserAlreadyExists
		}
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) FindByTenantAndEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.TenantID == tenantID && u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) ListByTenant(ctx context.Context, tenantID string, page, limit int) ([]*domain.User, int64, error) {
	var result []*domain.User
	for _, u := range m.users {
		if u.TenantID == tenantID {
			result = append(result, u)
		}
	}
	return result, int64(len(result)), nil
}

type mockRoleRepo struct {
	roles     map[string]*domain.Role
	userRoles map[string][]string // userID -> []roleID
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{
		roles:     make(map[string]*domain.Role),
		userRoles: make(map[string][]string),
	}
}

func (m *mockRoleRepo) Create(ctx context.Context, role *domain.Role) error {
	m.roles[role.ID] = role
	return nil
}

func (m *mockRoleRepo) FindByID(ctx context.Context, id string) (*domain.Role, error) {
	if r, ok := m.roles[id]; ok {
		return r, nil
	}
	return nil, domain.ErrRoleNotFound
}

func (m *mockRoleRepo) FindByName(ctx context.Context, tenantID, name string) (*domain.Role, error) {
	for _, r := range m.roles {
		if r.TenantID == tenantID && r.Name == name {
			return r, nil
		}
	}
	return nil, domain.ErrRoleNotFound
}

func (m *mockRoleRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Role, error) {
	var result []*domain.Role
	for _, r := range m.roles {
		if r.TenantID == tenantID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRoleRepo) AssignToUser(ctx context.Context, userID, roleID string) error {
	m.userRoles[userID] = append(m.userRoles[userID], roleID)
	return nil
}

func (m *mockRoleRepo) RevokeFromUser(ctx context.Context, userID, roleID string) error {
	roles := m.userRoles[userID]
	for i, r := range roles {
		if r == roleID {
			m.userRoles[userID] = append(roles[:i], roles[i+1:]...)
			return nil
		}
	}
	return nil
}

type mockTokenRepo struct {
	tokens map[string]*domain.RefreshToken
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{tokens: make(map[string]*domain.RefreshToken)}
}

func (m *mockTokenRepo) Create(ctx context.Context, token *domain.RefreshToken) error {
	m.tokens[token.Token] = token
	return nil
}

func (m *mockTokenRepo) FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	if t, ok := m.tokens[token]; ok {
		return t, nil
	}
	return nil, domain.ErrTokenInvalid
}

func (m *mockTokenRepo) RevokeByUserID(ctx context.Context, userID string) error {
	for _, t := range m.tokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}

func (m *mockTokenRepo) RevokeToken(ctx context.Context, token string) error {
	if t, ok := m.tokens[token]; ok {
		t.Revoked = true
		return nil
	}
	return nil
}

// --- Test Helpers ---

func newTestService(t *testing.T) (ports.AuthService, *mockUserRepo, *mockRoleRepo, *mockTokenRepo) {
	t.Helper()
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	tokenRepo := newMockTokenRepo()
	logger := zap.NewNop()
	svc := NewAuthService(userRepo, roleRepo, tokenRepo, "test-secret-32-characters-long!!", 15, 168, logger)
	return svc, userRepo, roleRepo, tokenRepo
}

func createTestUser(t *testing.T, userRepo *mockUserRepo, email, password string, status domain.UserStatus) *domain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	require.NoError(t, err)
	user := &domain.User{
		ID:           uuid.New().String(),
		TenantID:     "tenant-1",
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    "Test",
		LastName:     "User",
		Status:       status,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	userRepo.users[user.ID] = user
	return user
}

// --- Tests ---

func TestLogin_Success(t *testing.T) {
	svc, userRepo, _, _ := newTestService(t)
	createTestUser(t, userRepo, "test@example.com", "Password123!", domain.UserStatusActive)

	pair, err := svc.Login(context.Background(), ports.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, "Bearer", pair.TokenType)
	assert.Equal(t, int64(900), pair.ExpiresIn)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc, userRepo, _, _ := newTestService(t)
	createTestUser(t, userRepo, "test@example.com", "Password123!", domain.UserStatusActive)

	_, err := svc.Login(context.Background(), ports.LoginRequest{
		Email:    "test@example.com",
		Password: "WrongPassword!",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	_, err := svc.Login(context.Background(), ports.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "Password123!",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_InactiveUser(t *testing.T) {
	svc, userRepo, _, _ := newTestService(t)
	createTestUser(t, userRepo, "suspended@example.com", "Password123!", domain.UserStatusSuspended)

	_, err := svc.Login(context.Background(), ports.LoginRequest{
		Email:    "suspended@example.com",
		Password: "Password123!",
	})

	assert.ErrorIs(t, err, domain.ErrUserInactive)
}

func TestRegister_Success(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	user, err := svc.Register(context.Background(), ports.RegisterRequest{
		TenantID:  "tenant-1",
		Email:     "newuser@example.com",
		Password:  "Password123!",
		FirstName: "New",
		LastName:  "User",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "newuser@example.com", user.Email)
	assert.Equal(t, domain.UserStatusActive, user.Status)
	assert.NotEqual(t, "Password123!", user.PasswordHash)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, userRepo, _, _ := newTestService(t)
	createTestUser(t, userRepo, "existing@example.com", "Password123!", domain.UserStatusActive)

	_, err := svc.Register(context.Background(), ports.RegisterRequest{
		TenantID:  "tenant-1",
		Email:     "existing@example.com",
		Password:  "Password123!",
		FirstName: "Dup",
		LastName:  "User",
	})

	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)
}

func TestRefreshToken_Success(t *testing.T) {
	svc, userRepo, _, tokenRepo := newTestService(t)
	user := createTestUser(t, userRepo, "test@example.com", "Password123!", domain.UserStatusActive)

	// Store a valid refresh token
	refreshTokenStr := "valid-refresh-token-" + uuid.New().String()
	tokenRepo.tokens[refreshTokenStr] = &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TenantID:  user.TenantID,
		Token:     refreshTokenStr,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}

	pair, err := svc.RefreshToken(context.Background(), refreshTokenStr)

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.Equal(t, refreshTokenStr, pair.RefreshToken)
}

func TestRefreshToken_Expired(t *testing.T) {
	svc, _, _, tokenRepo := newTestService(t)

	expiredToken := "expired-token-" + uuid.New().String()
	tokenRepo.tokens[expiredToken] = &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    "user-1",
		TenantID:  "tenant-1",
		Token:     expiredToken,
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour), // already expired
		Revoked:   false,
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}

	_, err := svc.RefreshToken(context.Background(), expiredToken)

	assert.ErrorIs(t, err, domain.ErrTokenExpired)
}

func TestValidateToken_Success(t *testing.T) {
	svc, userRepo, _, _ := newTestService(t)
	createTestUser(t, userRepo, "test@example.com", "Password123!", domain.UserStatusActive)

	pair, err := svc.Login(context.Background(), ports.LoginRequest{
		Email:    "test@example.com",
		Password: "Password123!",
	})
	require.NoError(t, err)

	claims, err := svc.ValidateToken(context.Background(), pair.AccessToken)

	require.NoError(t, err)
	assert.NotEmpty(t, claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
}

func TestValidateToken_Invalid(t *testing.T) {
	svc, _, _, _ := newTestService(t)

	_, err := svc.ValidateToken(context.Background(), "invalid.jwt.token")

	assert.ErrorIs(t, err, domain.ErrTokenInvalid)
}
