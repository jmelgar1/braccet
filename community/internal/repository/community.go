package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/community/internal/domain"
)

var ErrCommunityNotFound = errors.New("community not found")

type CommunityRepository interface {
	Create(ctx context.Context, c *domain.Community) error
	GetByID(ctx context.Context, id uint64) (*domain.Community, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Community, error)
	ListByOwner(ctx context.Context, ownerID uint64) ([]*domain.Community, error)
	Update(ctx context.Context, c *domain.Community) error
	Delete(ctx context.Context, id uint64) error
}

type communityRepository struct {
	db *sql.DB
}

func NewCommunityRepository(db *sql.DB) CommunityRepository {
	return &communityRepository{db: db}
}

func (r *communityRepository) Create(ctx context.Context, c *domain.Community) error {
	query := `
		INSERT INTO communities (slug, owner_id, name, description, game, avatar_url, settings)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		c.Slug, c.OwnerID, c.Name, c.Description, c.Game, c.AvatarURL, c.Settings,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *communityRepository) GetByID(ctx context.Context, id uint64) (*domain.Community, error) {
	query := `
		SELECT id, slug, owner_id, name, description, game, avatar_url, COALESCE(settings, '{}'), created_at, updated_at
		FROM communities
		WHERE id = $1
	`
	c := &domain.Community{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Slug, &c.OwnerID, &c.Name, &c.Description, &c.Game, &c.AvatarURL, &c.Settings, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommunityNotFound
		}
		return nil, err
	}

	return c, nil
}

func (r *communityRepository) GetBySlug(ctx context.Context, slug string) (*domain.Community, error) {
	query := `
		SELECT id, slug, owner_id, name, description, game, avatar_url, COALESCE(settings, '{}'), created_at, updated_at
		FROM communities
		WHERE LOWER(slug) = LOWER($1)
	`
	c := &domain.Community{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&c.ID, &c.Slug, &c.OwnerID, &c.Name, &c.Description, &c.Game, &c.AvatarURL, &c.Settings, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommunityNotFound
		}
		return nil, err
	}

	return c, nil
}

func (r *communityRepository) ListByOwner(ctx context.Context, ownerID uint64) ([]*domain.Community, error) {
	query := `
		SELECT id, slug, owner_id, name, description, game, avatar_url, COALESCE(settings, '{}'), created_at, updated_at
		FROM communities
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`
	return r.queryCommunities(ctx, query, ownerID)
}

func (r *communityRepository) Update(ctx context.Context, c *domain.Community) error {
	query := `
		UPDATE communities
		SET name = $1, description = $2, game = $3, avatar_url = $4, settings = $5
		WHERE id = $6
	`
	result, err := r.db.ExecContext(ctx, query,
		c.Name, c.Description, c.Game, c.AvatarURL, c.Settings, c.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCommunityNotFound
	}

	return nil
}

func (r *communityRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM communities WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCommunityNotFound
	}

	return nil
}

func (r *communityRepository) queryCommunities(ctx context.Context, query string, args ...any) ([]*domain.Community, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var communities []*domain.Community
	for rows.Next() {
		c := &domain.Community{}
		err := rows.Scan(
			&c.ID, &c.Slug, &c.OwnerID, &c.Name, &c.Description, &c.Game, &c.AvatarURL, &c.Settings, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		communities = append(communities, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return communities, nil
}
