package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/community/internal/domain"
)

var ErrMemberNotFound = errors.New("member not found")

type MemberRepository interface {
	Create(ctx context.Context, m *domain.CommunityMember) error
	GetByID(ctx context.Context, id uint64) (*domain.CommunityMember, error)
	GetByCommunity(ctx context.Context, communityID uint64) ([]*domain.CommunityMember, error)
	GetByCommunityAndUser(ctx context.Context, communityID, userID uint64) (*domain.CommunityMember, error)
	ListGhostMembers(ctx context.Context, communityID uint64) ([]*domain.CommunityMember, error)
	Update(ctx context.Context, m *domain.CommunityMember) error
	UpdateRole(ctx context.Context, id uint64, role domain.MemberRole) error
	Delete(ctx context.Context, id uint64) error
	GetLeaderboard(ctx context.Context, communityID uint64, limit int) ([]*domain.CommunityMember, error)
	IncrementMatchStats(ctx context.Context, memberID uint64, won bool, newEloRating *int) error
}

type memberRepository struct {
	db *sql.DB
}

func NewMemberRepository(db *sql.DB) MemberRepository {
	return &memberRepository{db: db}
}

func (r *memberRepository) Create(ctx context.Context, m *domain.CommunityMember) error {
	query := `
		INSERT INTO community_members (community_id, user_id, display_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, joined_at, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		m.CommunityID, m.UserID, m.DisplayName, m.Role,
	).Scan(&m.ID, &m.JoinedAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *memberRepository) GetByID(ctx context.Context, id uint64) (*domain.CommunityMember, error) {
	query := `
		SELECT id, community_id, user_id, display_name, role::text, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE id = $1
	`
	m := &domain.CommunityMember{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.CommunityID, &m.UserID, &m.DisplayName, &m.Role,
		&m.EloRating, &m.RankingPoints, &m.MatchesPlayed, &m.MatchesWon,
		&m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}

	return m, nil
}

func (r *memberRepository) GetByCommunity(ctx context.Context, communityID uint64) ([]*domain.CommunityMember, error) {
	query := `
		SELECT id, community_id, user_id, display_name, role::text, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE community_id = $1
		ORDER BY role, display_name
	`
	return r.queryMembers(ctx, query, communityID)
}

func (r *memberRepository) GetByCommunityAndUser(ctx context.Context, communityID, userID uint64) (*domain.CommunityMember, error) {
	query := `
		SELECT id, community_id, user_id, display_name, role::text, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE community_id = $1 AND user_id = $2
	`
	m := &domain.CommunityMember{}
	err := r.db.QueryRowContext(ctx, query, communityID, userID).Scan(
		&m.ID, &m.CommunityID, &m.UserID, &m.DisplayName, &m.Role,
		&m.EloRating, &m.RankingPoints, &m.MatchesPlayed, &m.MatchesWon,
		&m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}

	return m, nil
}

func (r *memberRepository) ListGhostMembers(ctx context.Context, communityID uint64) ([]*domain.CommunityMember, error) {
	query := `
		SELECT id, community_id, user_id, display_name, role::text, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE community_id = $1 AND user_id IS NULL
		ORDER BY display_name
	`
	return r.queryMembers(ctx, query, communityID)
}

func (r *memberRepository) Update(ctx context.Context, m *domain.CommunityMember) error {
	query := `
		UPDATE community_members
		SET display_name = $1
		WHERE id = $2
	`
	result, err := r.db.ExecContext(ctx, query, m.DisplayName, m.ID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r *memberRepository) UpdateRole(ctx context.Context, id uint64, role domain.MemberRole) error {
	query := `
		UPDATE community_members
		SET role = $1
		WHERE id = $2
	`
	result, err := r.db.ExecContext(ctx, query, role, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r *memberRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM community_members WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r *memberRepository) GetLeaderboard(ctx context.Context, communityID uint64, limit int) ([]*domain.CommunityMember, error) {
	query := `
		SELECT id, community_id, user_id, display_name, role::text, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE community_id = $1 AND elo_rating IS NOT NULL
		ORDER BY elo_rating DESC
		LIMIT $2
	`
	return r.queryMembers(ctx, query, communityID, limit)
}

func (r *memberRepository) IncrementMatchStats(ctx context.Context, memberID uint64, won bool, newEloRating *int) error {
	var query string
	var args []any

	if won {
		query = `
			UPDATE community_members
			SET matches_played = matches_played + 1,
			    matches_won = matches_won + 1,
			    elo_rating = $1
			WHERE id = $2
		`
	} else {
		query = `
			UPDATE community_members
			SET matches_played = matches_played + 1,
			    elo_rating = $1
			WHERE id = $2
		`
	}
	args = []any{newEloRating, memberID}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMemberNotFound
	}

	return nil
}

func (r *memberRepository) queryMembers(ctx context.Context, query string, args ...any) ([]*domain.CommunityMember, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*domain.CommunityMember
	for rows.Next() {
		m := &domain.CommunityMember{}
		err := rows.Scan(
			&m.ID, &m.CommunityID, &m.UserID, &m.DisplayName, &m.Role,
			&m.EloRating, &m.RankingPoints, &m.MatchesPlayed, &m.MatchesWon,
			&m.JoinedAt, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}
