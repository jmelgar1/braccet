package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/braccet/community/internal/domain"
)

var ErrMemberEloRatingNotFound = errors.New("member elo rating not found")

type MemberEloRatingRepository interface {
	Create(ctx context.Context, r *domain.MemberEloRating) error
	GetByID(ctx context.Context, id uint64) (*domain.MemberEloRating, error)
	GetByMemberAndSystem(ctx context.Context, memberID, systemID uint64) (*domain.MemberEloRating, error)
	GetByMember(ctx context.Context, memberID uint64) ([]*domain.MemberEloRating, error)
	GetLeaderboard(ctx context.Context, systemID uint64, limit int) ([]*domain.MemberEloRating, error)
	Update(ctx context.Context, r *domain.MemberEloRating) error
	Delete(ctx context.Context, id uint64) error
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
			mer.created_at, mer.updated_at, cm.display_name
		FROM member_elo_ratings mer
		JOIN community_members cm ON cm.id = mer.member_id
		WHERE mer.id = $1
	`
	rating := &domain.MemberEloRating{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&rating.ID, &rating.MemberID, &rating.EloSystemID, &rating.Rating, &rating.GamesPlayed, &rating.GamesWon,
		&rating.CurrentWinStreak, &rating.HighestRating, &rating.LowestRating, &rating.LastGameAt,
		&rating.CreatedAt, &rating.UpdatedAt, &rating.MemberDisplayName,
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
			mer.created_at, mer.updated_at, cm.display_name
		FROM member_elo_ratings mer
		JOIN community_members cm ON cm.id = mer.member_id
		WHERE mer.member_id = $1 AND mer.elo_system_id = $2
	`
	rating := &domain.MemberEloRating{}
	err := r.db.QueryRowContext(ctx, query, memberID, systemID).Scan(
		&rating.ID, &rating.MemberID, &rating.EloSystemID, &rating.Rating, &rating.GamesPlayed, &rating.GamesWon,
		&rating.CurrentWinStreak, &rating.HighestRating, &rating.LowestRating, &rating.LastGameAt,
		&rating.CreatedAt, &rating.UpdatedAt, &rating.MemberDisplayName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberEloRatingNotFound
		}
		return nil, err
	}

	return rating, nil
}

func (r *memberEloRatingRepository) GetByMember(ctx context.Context, memberID uint64) ([]*domain.MemberEloRating, error) {
	query := `
		SELECT mer.id, mer.member_id, mer.elo_system_id, mer.rating, mer.games_played, mer.games_won,
			mer.current_win_streak, mer.highest_rating, mer.lowest_rating, mer.last_game_at,
			mer.created_at, mer.updated_at, cm.display_name
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
			&rating.CreatedAt, &rating.UpdatedAt, &rating.MemberDisplayName,
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
			mer.created_at, mer.updated_at, cm.display_name
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
		err := rows.Scan(
			&rating.ID, &rating.MemberID, &rating.EloSystemID, &rating.Rating, &rating.GamesPlayed, &rating.GamesWon,
			&rating.CurrentWinStreak, &rating.HighestRating, &rating.LowestRating, &rating.LastGameAt,
			&rating.CreatedAt, &rating.UpdatedAt, &rating.MemberDisplayName,
		)
		if err != nil {
			return nil, err
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
