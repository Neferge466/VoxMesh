package service

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/voxmesh/pkg/auth"
	voxerrors "github.com/voxmesh/pkg/errors"
	"github.com/voxmesh/pkg/model"
	"golang.org/x/crypto/bcrypt"

	"github.com/voxmesh/auth/internal/repository"
)

type AuthService struct {
	repo *repository.UserRepo
	pool *pgxpool.Pool
}

func NewAuthService(pool *pgxpool.Pool) *AuthService {
	return &AuthService{
		repo: repository.NewUserRepo(pool),
		pool: pool,
	}
}

func (s *AuthService) Register(ctx context.Context, req model.RegisterRequest) (*model.TokenPair, error) {
	// Check existing
	_, err := s.repo.FindByEmail(ctx, req.Email)
	if err == nil {
		return nil, voxerrors.ErrEmailTaken
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, voxerrors.ErrInternal
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, voxerrors.ErrInternal
	}
	req.Password = string(hashed)

	user, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, voxerrors.ErrInternal
	}

	return s.generateTokenPair(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (*model.TokenPair, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, voxerrors.ErrInvalidCredentials
		}
		return nil, voxerrors.ErrInternal
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, voxerrors.ErrInvalidCredentials
	}

	s.repo.UpdateLastLogin(ctx, user.ID)

	return s.generateTokenPair(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.TokenPair, error) {
	hash := auth.HashAPIKey(refreshToken)

	revoked, err := s.repo.IsTokenRevoked(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, voxerrors.ErrInvalidRefreshToken
		}
		return nil, voxerrors.ErrInternal
	}
	if revoked {
		return nil, voxerrors.ErrTokenRevoked
	}

	// Revoke old refresh token
	s.repo.RevokeRefreshToken(ctx, hash)

	// Get user for new token pair
	// We need user ID from refresh_tokens table
	var userID string
	err = s.pool.QueryRow(ctx, `SELECT user_id FROM refresh_tokens WHERE token_hash = $1`, hash).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, voxerrors.ErrInvalidRefreshToken
		}
		return nil, voxerrors.ErrInternal
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, voxerrors.ErrInternal
	}

	return s.generateTokenPair(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash := auth.HashAPIKey(refreshToken)
	return s.repo.RevokeRefreshToken(ctx, hash)
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*model.User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, voxerrors.ErrInvalidCredentials
	}
	return user, nil
}

func (s *AuthService) generateTokenPair(ctx context.Context, user *model.User) (*model.TokenPair, error) {
	accessToken, err := auth.GenerateAccessToken(user.ID, user.Username, user.Roles)
	if err != nil {
		return nil, voxerrors.ErrInternal
	}

	raw, hash := auth.GenerateRefreshToken()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if err := s.repo.StoreRefreshToken(ctx, user.ID, hash, expiresAt); err != nil {
		return nil, voxerrors.ErrInternal
	}

	return &model.TokenPair{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: raw,
		ExpiresIn:    3600,
	}, nil
}
