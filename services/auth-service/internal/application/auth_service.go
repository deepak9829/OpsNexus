package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/opsnexus/auth-service/internal/domain"
	"github.com/opsnexus/auth-service/internal/ports"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo   ports.UserRepository
	roleRepo   ports.RoleRepository
	tokenRepo  ports.RefreshTokenRepository
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
	logger     *zap.Logger
}

func NewAuthService(
	userRepo ports.UserRepository,
	roleRepo ports.RoleRepository,
	tokenRepo ports.RefreshTokenRepository,
	jwtSecret string,
	accessTTLMinutes int,
	refreshTTLHours int,
	logger *zap.Logger,
) ports.AuthService {
	return &authService{
		userRepo:   userRepo,
		roleRepo:   roleRepo,
		tokenRepo:  tokenRepo,
		jwtSecret:  jwtSecret,
		accessTTL:  time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshTTLHours) * time.Hour,
		logger:     logger,
	}
}

func (s *authService) Register(ctx context.Context, req ports.RegisterRequest) (*domain.User, error) {
	if req.Email == "" || req.Password == "" || req.FirstName == "" || req.LastName == "" {
		return nil, domain.ErrInvalidInput
	}
	if req.TenantID == "" {
		return nil, domain.ErrInvalidInput
	}

	existing, err := s.userRepo.FindByTenantAndEmail(ctx, req.TenantID, req.Email)
	if err == nil && existing != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		TenantID:     req.TenantID,
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: string(hash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Status:       domain.UserStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, fmt.Errorf("creating user: %w", err)
	}

	s.logger.Info("user registered", zap.String("user_id", user.ID), zap.String("email", user.Email))
	return user, nil
}

func (s *authService) Login(ctx context.Context, req ports.LoginRequest) (*domain.TokenPair, error) {
	if req.Email == "" || req.Password == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.userRepo.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !user.IsActive() {
		return nil, domain.ErrUserInactive
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	refreshTokenStr := uuid.New().String() + "-" + uuid.New().String()
	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TenantID:  user.TenantID,
		Token:     refreshTokenStr,
		ExpiresAt: time.Now().UTC().Add(s.refreshTTL),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.tokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	s.logger.Info("user logged in", zap.String("user_id", user.ID))
	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	if refreshToken == "" {
		return nil, domain.ErrInvalidInput
	}

	stored, err := s.tokenRepo.FindByToken(ctx, refreshToken)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	if !stored.IsValid() {
		if stored.Revoked {
			return nil, domain.ErrTokenRevoked
		}
		return nil, domain.ErrTokenExpired
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	if !user.IsActive() {
		return nil, domain.ErrUserInactive
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return domain.ErrInvalidInput
	}
	return s.tokenRepo.RevokeToken(ctx, refreshToken)
}

func (s *authService) ValidateToken(ctx context.Context, accessToken string) (*ports.Claims, error) {
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, domain.ErrTokenInvalid
	}

	userID, _ := claims["sub"].(string)
	tenantID, _ := claims["tid"].(string)
	email, _ := claims["email"].(string)

	var roles []string
	if rolesRaw, ok := claims["roles"].([]interface{}); ok {
		for _, r := range rolesRaw {
			if rs, ok := r.(string); ok {
				roles = append(roles, rs)
			}
		}
	}

	return &ports.Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Roles:    roles,
	}, nil
}

func (s *authService) GetCurrentUser(ctx context.Context, userID string) (*domain.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

func (s *authService) AssignRole(ctx context.Context, userID, roleID string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return domain.ErrUserNotFound
	}

	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return domain.ErrRoleNotFound
	}

	if user.TenantID != role.TenantID {
		return domain.ErrTenantMismatch
	}

	return s.roleRepo.AssignToUser(ctx, userID, roleID)
}

func (s *authService) generateAccessToken(user *domain.User) (string, error) {
	roleNames := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roleNames = append(roleNames, r.Name)
	}

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"tid":   user.TenantID,
		"email": user.Email,
		"roles": roleNames,
		"iss":   "opsnexus-auth",
		"iat":   now.Unix(),
		"exp":   now.Add(s.accessTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *authService) ListUsers(ctx context.Context, tenantID string, page, limit int) ([]*domain.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.userRepo.ListByTenant(ctx, tenantID, page, limit)
}

func (s *authService) UpdateUserStatus(ctx context.Context, userID, status string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("finding user: %w", err)
	}
	switch status {
	case "active":
		user.Status = domain.UserStatusActive
	case "inactive":
		user.Status = domain.UserStatusInactive
	case "suspended":
		user.Status = domain.UserStatusSuspended
	default:
		return domain.ErrInvalidInput
	}
	user.UpdatedAt = time.Now().UTC()
	return s.userRepo.Update(ctx, user)
}
