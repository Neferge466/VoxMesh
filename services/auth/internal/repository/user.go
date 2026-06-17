package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voxmesh/pkg/model"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var user model.User
	err = tx.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash, display_name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, username, email, display_name, is_active, created_at, updated_at`,
		req.Username, req.Email, req.Password, req.Username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Assign default "user" role
	_, err = tx.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE name = 'user'`,
		user.ID)
	if err != nil {
		return nil, fmt.Errorf("assign default role: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	user.Roles = []string{"user"}
	return &user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.username, u.email, u.password_hash,
			COALESCE(u.display_name, '') AS display_name,
			COALESCE(u.avatar_url, '') AS avatar_url,
			u.is_active, u.created_at, u.updated_at, u.last_login_at,
			COALESCE(array_agg(r.name ORDER BY r.name) FILTER (WHERE r.name IS NOT NULL), '{}') AS roles
		 FROM users u
		 LEFT JOIN user_roles ur ON u.id = ur.user_id
		 LEFT JOIN roles r ON ur.role_id = r.id
		 WHERE u.email = $1
		 GROUP BY u.id`, email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.DisplayName, &user.AvatarURL, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt, &user.Roles)
	if err != nil {
		return nil, fmt.Errorf("find by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.username, u.email, u.password_hash,
			COALESCE(u.display_name, '') AS display_name,
			COALESCE(u.avatar_url, '') AS avatar_url,
			u.is_active, u.created_at, u.updated_at, u.last_login_at,
			COALESCE(array_agg(r.name ORDER BY r.name) FILTER (WHERE r.name IS NOT NULL), '{}') AS roles
		 FROM users u
		 LEFT JOIN user_roles ur ON u.id = ur.user_id
		 LEFT JOIN roles r ON ur.role_id = r.id
		 WHERE u.id = $1
		 GROUP BY u.id`, id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.DisplayName, &user.AvatarURL, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt, &user.Roles)
	if err != nil {
		return nil, fmt.Errorf("find by id: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = $1, updated_at = $1 WHERE id = $2`, time.Now(), userID)
	return err
}

// --- Refresh Token operations ---

func (r *UserRepo) StoreRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt)
	return err
}

func (r *UserRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = $1 WHERE token_hash = $2 AND revoked_at IS NULL`, time.Now(), tokenHash)
	return err
}

func (r *UserRepo) IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	var revoked bool
	err := r.pool.QueryRow(ctx,
		`SELECT revoked_at IS NOT NULL FROM refresh_tokens WHERE token_hash = $1`, tokenHash).Scan(&revoked)
	if err != nil {
		return false, err
	}
	return revoked, nil
}
