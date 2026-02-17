package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/auth/internal/domain"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uint64) (*domain.User, error)
	GetByOAuth(ctx context.Context, provider domain.OAuthProvider, oauthID string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	Update(ctx context.Context, u *domain.User) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (email, display_name, avatar_url, oauth_provider, oauth_id, username, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		u.Email, u.DisplayName, u.AvatarURL, u.OAuthProvider, u.OAuthID, u.Username, u.PasswordHash,
	).Scan(&u.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uint64) (*domain.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, oauth_provider, oauth_id, username, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL,
		&u.OAuthProvider, &u.OAuthID, &u.Username, &u.PasswordHash,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return u, nil
}

func (r *userRepository) GetByOAuth(ctx context.Context, provider domain.OAuthProvider, oauthID string) (*domain.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, oauth_provider, oauth_id, username, password_hash, created_at, updated_at
		FROM users
		WHERE oauth_provider = $1 AND oauth_id = $2
	`
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, provider, oauthID).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL,
		&u.OAuthProvider, &u.OAuthID, &u.Username, &u.PasswordHash,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return u, nil
}

func (r *userRepository) Update(ctx context.Context, u *domain.User) error {
	query := `
		UPDATE users
		SET email = $1, display_name = $2, avatar_url = $3, username = $4, password_hash = $5
		WHERE id = $6
	`
	result, err := r.db.ExecContext(ctx, query,
		u.Email, u.DisplayName, u.AvatarURL, u.Username, u.PasswordHash, u.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, oauth_provider, oauth_id, username, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL,
		&u.OAuthProvider, &u.OAuthID, &u.Username, &u.PasswordHash,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return u, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, oauth_provider, oauth_id, username, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL,
		&u.OAuthProvider, &u.OAuthID, &u.Username, &u.PasswordHash,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return u, nil
}
