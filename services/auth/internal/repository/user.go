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
		INSERT INTO users (email, display_name, avatar_url, oauth_provider, oauth_id)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		u.Email, u.DisplayName, u.AvatarURL, u.OAuthProvider, u.OAuthID,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = uint64(id)

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uint64) (*domain.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, oauth_provider, oauth_id, created_at, updated_at
		FROM users
		WHERE id = ?
	`
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL,
		&u.OAuthProvider, &u.OAuthID, &u.CreatedAt, &u.UpdatedAt,
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
		SELECT id, email, display_name, avatar_url, oauth_provider, oauth_id, created_at, updated_at
		FROM users
		WHERE oauth_provider = ? AND oauth_id = ?
	`
	u := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, provider, oauthID).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL,
		&u.OAuthProvider, &u.OAuthID, &u.CreatedAt, &u.UpdatedAt,
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
		SET email = ?, display_name = ?, avatar_url = ?
		WHERE id = ?
	`
	result, err := r.db.ExecContext(ctx, query,
		u.Email, u.DisplayName, u.AvatarURL, u.ID,
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
