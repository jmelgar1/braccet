package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/braccet/community/internal/domain"
)

var ErrMemberEloRatingNotFound = errors.New("member elo rating not found")

type MemberEloRatingRepository interface {
	Create(ctx context.Context, r *domain.MemberEloRating) error
	GetByID(ctx context.Context, id uint64) (*domain.MemberEloRating, error)
	GetByMemberAndSystem(ctx context.Context, memberID, systemID uint64) (*domain.MemberEloRating, error)
	GetByMembersAndSystem(ctx context.Context, memberIDs []uint64, systemID uint64) ([]*domain.MemberEloRating, error)
	GetByMember(ctx context.Context, memberID uint64) ([]*domain.MemberEloRating, error)
	GetLeaderboard(ctx context.Context, systemID uint64, limit int) ([]*domain.MemberEloRating, error)
	Update(ctx context.Context, r *domain.MemberEloRating) error
	Delete(ctx context.Context, id uint64) error
	DeleteByMemberAndSystem(ctx context.Context, memberID, systemID uint64) error
	ApplyRatingReversion(ctx context.Context, memberID, systemID uint64, ratingChange int, wasWinner bool, floorRating int) error
}

type memberEloRatingRepository struct {
	db *sql.DB
}

func NewMemberEloRatingRepository(db *sql.DB) MemberEloRatingRepository {
	return &memberEloRatingRepository{db: db}
}

func (r *memberEloRatingRepository) Create(ctx context.Context, rating *domain.MemberEloRating) error {
	query := `
		INSERT INTO member_elo_ratings (
			member_id, elo_system_id, rating, games_played, games_won,
			current_win_streak, highest_rating, lowest_rating, last_game_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		rating.MemberID, rating.EloSystemID, rating.Rating, rating.GamesPlayed, rating.GamesWon,
		rating.CurrentWinStreak, rating.HighestRating, rating.LowestRating, rating.LastGameAt,
	).Scan(&rating.ID, &rating.CreatedAt, &rating.UpdatedAt)

	return err
}

func (r *memberEloRatingRepository) GetByID(ctx context.Context, id uint64) (*domain.MemberEloRating, error) {
	query := `
		SELECT mer.id, mer.member_id, mer.elo_system_id, mer.rating, mer.games_played, mer.games_won,
			mer.current_win_streak, mer.highest_rating, mer.lowest_rating, mer.last_game_at,
			mer.created_at, mer.updated_at, cm.display_name, cm.region, cm.icon_url
		FROM member_elo_ratings mer
		JOIN community_members cm ON cm.id = mer.member_id
		WHERE mer.id = $1
	`
	rating := &domain.MemberEloRating{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rating.ID, &rating.MemberID, &rating.EloSystemID, &rating.Rating, &rating.GamesPlayed, &rating.GamesWon,
		&rating.CurrentWinStreak, &rating.HighestRating, &rating.LowestRating, &rating.LastGameAt,
		&rating.CreatedAt, &rating.UpdatedAt, &rating.MemberDisplayName, &rating.MemberRegion, &rating.MemberIconURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberEloRatingNotFound
		}
		return nil, err
	}

	return rating, nil
}

func (r *memberEloRatingRepository) GetByMemberAndSystem(ctx context.Context, memberID, systemID uint64) (*domain.MemberEloRating, error) {
	query := `
		SELECT mer.id, mer.member_id, mer.elo_system_id, mer.rating, mer.games_played, mer.games_won,
			mer.current_win_streak, mer.highest_rating, mer.lowest_rating, mer.last_game_at,
			mer.created_at, mer.updated_at, cm.display_name, cm.region, cm.icon_url
		FROM member_elo_ratings mer
		JOIN community_members cm ON cm.id = mer.member_id
		WHERE mer.member_id = $1 AND mer.elo_system_id = $2
	`
	rating := &domain.MemberEloRating{}
	err := r.db.QueryRowContext(ctx, query, memberID, systemID).Scan(
		&rating.ID, &rating.MemberID, &rating.EloSystemID, &rating.Rating, &rating.GamesPlayed, &rating.GamesWon,
		&rating.CurrentWinStreak, &rating.HighestRating, &rating.LowestRating, &rating.LastGameAt,
		&rating.CreatedAt, &rating.UpdatedAt, &rating.MemberDisplayName, &rating.MemberRegion, &rating.MemberIconURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberEloRatingNotFound
		}
		return nil, err
	}

	return rating, nil
}

func (r *memberEloRatingRepository) GetByMembersAndSystem(ctx context.Context, memberIDs []uint64, systemID uint64) ([]*domain.MemberEloRating, error) {
	if len(memberIDs) == 0 {
		return nil, nil
	}

	// Build placeholders: $2, $3, $4, ... (systemID is $1)
	placeholders := make([]string, len(memberIDs))
	args := make([]any, len(memberIDs)+1)
	args[0] = systemID
	for i, id := range memberIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		SELECT mer.id, mer.member_id, mer.elo_system_id, mer.rating, mer.games_played, mer.games_won,
			mer.current_win_streak, mer.highest_rating, mer.lowest_rating, mer.last_game_at,
			mer.created_at, mer.updated_at, cm.display_name, cm.region, cm.icon_url
		FROM member_elo_ratings mer
		JOIN community_members cm ON cm.id = mer.member_id
		WHERE mer.elo_system_id = $1 AND mer.member_id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []*domain.MemberEloRating
	for rows.Next() {
		rating := &domain.MemberEloRating{}
		err := rows.Scan(
			&rating.ID, &rating.MemberID, &rating.EloSystemID, &rating.Rating, &rating.GamesPlayed, &rating.GamesWon,
			&rating.CurrentWinStreak, &rating.HighestRating, &rating.LowestRating, &rating.LastGameAt,
			&rating.CreatedAt, &rating.UpdatedAt, &rating.MemberDisplayName, &rating.MemberRegion, &rating.MemberIconURL,
		)
		if err != nil {
			return nil, err
		}
		ratings = append(ratings, rating)
	}

	return ratings, rows.Err()
}

func (r *memberEloRatingRepository) GetByMember(ctx context.Context, memberID uint64) ([]*domain.MemberEloRating, error) {
	query := `
		SELECT mer.id, mer.member_id, mer.elo_system_id, mer.rating, mer.games_played, mer.games_won,
			mer.current_win_streak, mer.highest_rating, mer.lowest_rating, mer.last_game_at,
			mer.created_at, mer.updated_at, cm.display_name, cm.region, cm.icon_url
		FROM member_elo_ratings mer
		JOIN community_members cm ON cm.id = mer.member_id
		WHERE mer.member_id = $1
		ORDER BY mer.rating DESC
	`
	rows, err := r.db.QueryContext(ctx, query, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []*domain.MemberEloRating
	for rows.Next() {
		rating := &domain.MemberEloRating{}
		err := rows.Scan(
			&rating.ID, &rating.MemberID, &rating.EloSystemID, &rating.Rating, &rating.GamesPlayed, &rating.GamesWon,
			&rating.CurrentWinStreak, &rating.HighestRating, &rating.LowestRating, &rating.LastGameAt,
			&rating.CreatedAt, &rating.UpdatedAt, &rating.MemberDisplayName, &rating.MemberRegion, &rating.MemberIconURL,
		)
		if err != nil {
			return nil, err
		}
		ratings = append(ratings, rating)
	}

	return ratings, rows.Err()
}

func (r *memberEloRatingRepository) GetLeaderboard(ctx context.Context, systemID uint64, limit int) ([]*domain.MemberEloRating, error) {
	query := `
		SELECT mer.id, mer.member_id, mer.elo_system_id, mer.rating, mer.games_played, mer.games_won,
			mer.current_win_streak, mer.highest_rating, mer.lowest_rating, mer.last_game_at,
			mer.created_at, mer.updated_at, cm.display_name, cm.region, cm.icon_url,
			(
				SELECT COALESCE(string_agg(
					CASE
						WHEN eh.is_winner = true THEN 'W'
						WHEN eh.is_winner = false THEN 'L'
						ELSE 'D'
					END, ','
				ORDER BY eh.created_at DESC), '')
				FROM (
					SELECT eh.is_winner, eh.created_at
					FROM elo_history eh
					WHERE eh.member_id = mer.member_id
						AND eh.elo_system_id = mer.elo_system_id
						AND eh.change_type = 'match'
					ORDER BY eh.created_at DESC
					LIMIT 5
				) eh
			) as recent_results
		FROM member_elo_ratings mer
		JOIN community_members cm ON cm.id = mer.member_id
		WHERE mer.elo_system_id = $1
		ORDER BY mer.rating DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, systemID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []*domain.MemberEloRating
	for rows.Next() {
		rating := &domain.MemberEloRating{}
		var recentResultsStr string
		err := rows.Scan(
			&rating.ID, &rating.MemberID, &rating.EloSystemID, &rating.Rating, &rating.GamesPlayed, &rating.GamesWon,
			&rating.CurrentWinStreak, &rating.HighestRating, &rating.LowestRating, &rating.LastGameAt,
			&rating.CreatedAt, &rating.UpdatedAt, &rating.MemberDisplayName, &rating.MemberRegion, &rating.MemberIconURL,
			&recentResultsStr,
		)
		if err != nil {
			return nil, err
		}
		// Parse the recent results string into a slice
		if recentResultsStr != "" {
			parts := strings.Split(recentResultsStr, ",")
			for _, p := range parts {
				rating.RecentResults = append(rating.RecentResults, domain.MatchResult(p))
			}
		}
		ratings = append(ratings, rating)
	}

	return ratings, rows.Err()
}

func (r *memberEloRatingRepository) Update(ctx context.Context, rating *domain.MemberEloRating) error {
	now := time.Now()
	rating.LastGameAt = &now

	query := `
		UPDATE member_elo_ratings SET
			rating = $1, games_played = $2, games_won = $3,
			current_win_streak = $4, highest_rating = $5, lowest_rating = $6,
			last_game_at = $7
		WHERE id = $8
	`
	result, err := r.db.ExecContext(ctx, query,
		rating.Rating, rating.GamesPlayed, rating.GamesWon,
		rating.CurrentWinStreak, rating.HighestRating, rating.LowestRating,
		rating.LastGameAt, rating.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMemberEloRatingNotFound
	}

	return nil
}

func (r *memberEloRatingRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM member_elo_ratings WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMemberEloRatingNotFound
	}

	return nil
}

// DeleteByMemberAndSystem removes a rating record for a specific member and system.
// This is used when reverting a tournament where a member had no prior games.
func (r *memberEloRatingRepository) DeleteByMemberAndSystem(ctx context.Context, memberID, systemID uint64) error {
	query := `DELETE FROM member_elo_ratings WHERE member_id = $1 AND elo_system_id = $2`
	_, err := r.db.ExecContext(ctx, query, memberID, systemID)
	return err
}

// ApplyRatingReversion reverses a single match's effect on a member's rating.
// ratingChange is the original change that was applied (will be subtracted).
// wasWinner indicates if this member won the original match (decrements games_won).
// floorRating is the minimum rating allowed.
func (r *memberEloRatingRepository) ApplyRatingReversion(ctx context.Context, memberID, systemID uint64, ratingChange int, wasWinner bool, floorRating int) error {
	// Reverse the rating change, ensuring we don't go below floor
	// Also decrement games_played, and games_won if they won
	// Reset current_win_streak to 0 (too complex to recalculate accurately)
	var query string
	if wasWinner {
		query = `
			UPDATE member_elo_ratings SET
				rating = GREATEST($1, rating - $2),
				games_played = GREATEST(0, games_played - 1),
				games_won = GREATEST(0, games_won - 1),
				current_win_streak = 0
			WHERE member_id = $3 AND elo_system_id = $4
		`
	} else {
		query = `
			UPDATE member_elo_ratings SET
				rating = GREATEST($1, rating - $2),
				games_played = GREATEST(0, games_played - 1),
				current_win_streak = 0
			WHERE member_id = $3 AND elo_system_id = $4
		`
	}

	result, err := r.db.ExecContext(ctx, query, floorRating, ratingChange, memberID, systemID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		// Member rating record doesn't exist - this is OK, they might not have played yet
		return nil
	}

	return nil
}
