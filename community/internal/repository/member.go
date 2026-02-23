package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/braccet/community/internal/domain"
)

var ErrMemberNotFound = errors.New("member not found")

type MemberRepository interface {
	Create(ctx context.Context, m *domain.CommunityMember) error
	GetByID(ctx context.Context, id uint64) (*domain.CommunityMember, error)
	GetByCommunity(ctx context.Context, communityID uint64) ([]*domain.CommunityMember, error)
	GetByCommunityAndUser(ctx context.Context, communityID, userID uint64) (*domain.CommunityMember, error)
	GetByCommunityAndDisplayName(ctx context.Context, communityID uint64, displayName string) (*domain.CommunityMember, error)
	ListGhostMembers(ctx context.Context, communityID uint64) ([]*domain.CommunityMember, error)
	Update(ctx context.Context, m *domain.CommunityMember) error
	UpdateRole(ctx context.Context, id uint64, role domain.MemberRole) error
	Delete(ctx context.Context, id uint64) error
	DeleteMany(ctx context.Context, memberIDs []uint64) (int64, error)
	GetLeaderboard(ctx context.Context, communityID uint64, limit int) ([]*domain.CommunityMember, error)
	IncrementMatchStats(ctx context.Context, memberID uint64, won bool, newEloRating *int) error
	DecrementMatchStats(ctx context.Context, memberID uint64, won bool, newEloRating *int) error
	SearchByDisplayNamePrefix(ctx context.Context, communityID uint64, prefix string, limit int) ([]*domain.CommunityMember, error)
	GetBulkIconURLs(ctx context.Context, memberIDs []uint64) (map[uint64]string, error)
	GetBulkMemberData(ctx context.Context, memberIDs []uint64) (map[uint64]MemberData, error)
	GetDistinctRegions(ctx context.Context, communityID uint64) ([]string, error)
}

// MemberData holds icon and region data for bulk fetches
type MemberData struct {
	IconURL *string
	Region  *string
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
		SELECT id, community_id, user_id, display_name, role::text, icon_url, region, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE id = $1
	`
	m := &domain.CommunityMember{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.CommunityID, &m.UserID, &m.DisplayName, &m.Role, &m.IconURL, &m.Region,
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
		SELECT id, community_id, user_id, display_name, role::text, icon_url, region, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE community_id = $1
		ORDER BY role, display_name
	`
	return r.queryMembers(ctx, query, communityID)
}

func (r *memberRepository) GetByCommunityAndUser(ctx context.Context, communityID, userID uint64) (*domain.CommunityMember, error) {
	query := `
		SELECT id, community_id, user_id, display_name, role::text, icon_url, region, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE community_id = $1 AND user_id = $2
	`
	m := &domain.CommunityMember{}
	err := r.db.QueryRowContext(ctx, query, communityID, userID).Scan(
		&m.ID, &m.CommunityID, &m.UserID, &m.DisplayName, &m.Role, &m.IconURL, &m.Region,
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

func (r *memberRepository) GetByCommunityAndDisplayName(ctx context.Context, communityID uint64, displayName string) (*domain.CommunityMember, error) {
	query := `
		SELECT id, community_id, user_id, display_name, role::text, icon_url, region, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE community_id = $1 AND display_name = $2
		LIMIT 1
	`
	m := &domain.CommunityMember{}
	err := r.db.QueryRowContext(ctx, query, communityID, displayName).Scan(
		&m.ID, &m.CommunityID, &m.UserID, &m.DisplayName, &m.Role, &m.IconURL, &m.Region,
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
		SELECT id, community_id, user_id, display_name, role::text, icon_url, region, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE community_id = $1 AND user_id IS NULL
		ORDER BY display_name
	`
	return r.queryMembers(ctx, query, communityID)
}

func (r *memberRepository) Update(ctx context.Context, m *domain.CommunityMember) error {
	query := `
		UPDATE community_members
		SET display_name = $1, icon_url = $2, region = $3
		WHERE id = $4
	`
	result, err := r.db.ExecContext(ctx, query, m.DisplayName, m.IconURL, m.Region, m.ID)
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

// DeleteMany deletes multiple community members by their IDs.
// Returns the number of rows deleted.
func (r *memberRepository) DeleteMany(ctx context.Context, memberIDs []uint64) (int64, error) {
	if len(memberIDs) == 0 {
		return 0, nil
	}

	// Build placeholders: $1, $2, $3, ...
	placeholders := make([]string, len(memberIDs))
	args := make([]any, len(memberIDs))
	for i, id := range memberIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM community_members WHERE id IN (%s)",
		strings.Join(placeholders, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *memberRepository) GetLeaderboard(ctx context.Context, communityID uint64, limit int) ([]*domain.CommunityMember, error) {
	query := `
		SELECT id, community_id, user_id, display_name, role::text, icon_url, region, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
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

func (r *memberRepository) DecrementMatchStats(ctx context.Context, memberID uint64, won bool, newEloRating *int) error {
	var query string
	var args []any

	if won {
		query = `
			UPDATE community_members
			SET matches_played = GREATEST(0, matches_played - 1),
			    matches_won = GREATEST(0, matches_won - 1),
			    elo_rating = $1
			WHERE id = $2
		`
	} else {
		query = `
			UPDATE community_members
			SET matches_played = GREATEST(0, matches_played - 1),
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

func (r *memberRepository) SearchByDisplayNamePrefix(ctx context.Context, communityID uint64, prefix string, limit int) ([]*domain.CommunityMember, error) {
	query := `
		SELECT id, community_id, user_id, display_name, role::text, icon_url, region, elo_rating, ranking_points, matches_played, matches_won, joined_at, created_at, updated_at
		FROM community_members
		WHERE community_id = $1 AND LOWER(display_name) LIKE LOWER($2 || '%')
		ORDER BY display_name
		LIMIT $3
	`
	return r.queryMembers(ctx, query, communityID, prefix, limit)
}

func (r *memberRepository) GetBulkIconURLs(ctx context.Context, memberIDs []uint64) (map[uint64]string, error) {
	if len(memberIDs) == 0 {
		return make(map[uint64]string), nil
	}

	// Build placeholders: $1, $2, $3, ...
	placeholders := make([]string, len(memberIDs))
	args := make([]any, len(memberIDs))
	for i, id := range memberIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, icon_url
		FROM community_members
		WHERE id IN (%s) AND icon_url IS NOT NULL
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uint64]string)
	for rows.Next() {
		var id uint64
		var iconURL string
		if err := rows.Scan(&id, &iconURL); err != nil {
			return nil, err
		}
		result[id] = iconURL
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *memberRepository) GetBulkMemberData(ctx context.Context, memberIDs []uint64) (map[uint64]MemberData, error) {
	if len(memberIDs) == 0 {
		return make(map[uint64]MemberData), nil
	}

	// Build placeholders: $1, $2, $3, ...
	placeholders := make([]string, len(memberIDs))
	args := make([]any, len(memberIDs))
	for i, id := range memberIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, icon_url, region
		FROM community_members
		WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uint64]MemberData)
	for rows.Next() {
		var id uint64
		var data MemberData
		if err := rows.Scan(&id, &data.IconURL, &data.Region); err != nil {
			return nil, err
		}
		result[id] = data
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
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
			&m.ID, &m.CommunityID, &m.UserID, &m.DisplayName, &m.Role, &m.IconURL, &m.Region,
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

func (r *memberRepository) GetDistinctRegions(ctx context.Context, communityID uint64) ([]string, error) {
	query := `
		SELECT DISTINCT region
		FROM community_members
		WHERE community_id = $1 AND region IS NOT NULL AND region != ''
		ORDER BY region
	`
	rows, err := r.db.QueryContext(ctx, query, communityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []string
	for rows.Next() {
		var region string
		if err := rows.Scan(&region); err != nil {
			return nil, err
		}
		regions = append(regions, region)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return regions, nil
}
