package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/braccet/community/internal/domain"
)

var ErrMatchCacheNotFound = errors.New("match cache entry not found")

type PowerRankingMatchCacheRepository interface {
	Create(ctx context.Context, m *domain.PowerRankingMatchCache) error
	GetByMemberSince(ctx context.Context, memberID, systemID uint64, since time.Time) ([]*domain.PowerRankingMatchCache, error)
	GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.PowerRankingMatchCache, error)
	Upsert(ctx context.Context, m *domain.PowerRankingMatchCache) error
	DeleteByTournament(ctx context.Context, tournamentID uint64) error
	DeleteByMatch(ctx context.Context, matchID uint64) error
}

type powerRankingMatchCacheRepository struct {
	db *sql.DB
}

func NewPowerRankingMatchCacheRepository(db *sql.DB) PowerRankingMatchCacheRepository {
	return &powerRankingMatchCacheRepository{db: db}
}

func (r *powerRankingMatchCacheRepository) Create(ctx context.Context, m *domain.PowerRankingMatchCache) error {
	query := `
		INSERT INTO power_ranking_match_cache (
			member_id, pr_system_id, match_id, tournament_id,
			is_winner, sets_won, sets_lost, opponent_member_id, opponent_rating,
			is_lan, completed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		m.MemberID, m.PRSystemID, m.MatchID, m.TournamentID,
		m.IsWinner, m.SetsWon, m.SetsLost, m.OpponentMemberID, m.OpponentRating,
		m.IsLAN, m.CompletedAt,
	).Scan(&m.ID, &m.CreatedAt)
}

func (r *powerRankingMatchCacheRepository) GetByMemberSince(ctx context.Context, memberID, systemID uint64, since time.Time) ([]*domain.PowerRankingMatchCache, error) {
	query := `
		SELECT id, member_id, pr_system_id, match_id, tournament_id,
			is_winner, sets_won, sets_lost, opponent_member_id, opponent_rating,
			is_lan, completed_at, created_at
		FROM power_ranking_match_cache
		WHERE member_id = $1 AND pr_system_id = $2 AND completed_at >= $3
		ORDER BY completed_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, memberID, systemID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*domain.PowerRankingMatchCache
	for rows.Next() {
		m := &domain.PowerRankingMatchCache{}
		err := rows.Scan(
			&m.ID, &m.MemberID, &m.PRSystemID, &m.MatchID, &m.TournamentID,
			&m.IsWinner, &m.SetsWon, &m.SetsLost, &m.OpponentMemberID, &m.OpponentRating,
			&m.IsLAN, &m.CompletedAt, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

func (r *powerRankingMatchCacheRepository) GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.PowerRankingMatchCache, error) {
	query := `
		SELECT id, member_id, pr_system_id, match_id, tournament_id,
			is_winner, sets_won, sets_lost, opponent_member_id, opponent_rating,
			is_lan, completed_at, created_at
		FROM power_ranking_match_cache
		WHERE tournament_id = $1
		ORDER BY completed_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*domain.PowerRankingMatchCache
	for rows.Next() {
		m := &domain.PowerRankingMatchCache{}
		err := rows.Scan(
			&m.ID, &m.MemberID, &m.PRSystemID, &m.MatchID, &m.TournamentID,
			&m.IsWinner, &m.SetsWon, &m.SetsLost, &m.OpponentMemberID, &m.OpponentRating,
			&m.IsLAN, &m.CompletedAt, &m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

func (r *powerRankingMatchCacheRepository) Upsert(ctx context.Context, m *domain.PowerRankingMatchCache) error {
	query := `
		INSERT INTO power_ranking_match_cache (
			member_id, pr_system_id, match_id, tournament_id,
			is_winner, sets_won, sets_lost, opponent_member_id, opponent_rating,
			is_lan, completed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (member_id, match_id) DO UPDATE SET
			is_winner = EXCLUDED.is_winner,
			sets_won = EXCLUDED.sets_won,
			sets_lost = EXCLUDED.sets_lost,
			opponent_member_id = EXCLUDED.opponent_member_id,
			opponent_rating = EXCLUDED.opponent_rating,
			is_lan = EXCLUDED.is_lan,
			completed_at = EXCLUDED.completed_at
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		m.MemberID, m.PRSystemID, m.MatchID, m.TournamentID,
		m.IsWinner, m.SetsWon, m.SetsLost, m.OpponentMemberID, m.OpponentRating,
		m.IsLAN, m.CompletedAt,
	).Scan(&m.ID, &m.CreatedAt)
}

func (r *powerRankingMatchCacheRepository) DeleteByTournament(ctx context.Context, tournamentID uint64) error {
	query := `DELETE FROM power_ranking_match_cache WHERE tournament_id = $1`
	_, err := r.db.ExecContext(ctx, query, tournamentID)
	return err
}

func (r *powerRankingMatchCacheRepository) DeleteByMatch(ctx context.Context, matchID uint64) error {
	query := `DELETE FROM power_ranking_match_cache WHERE match_id = $1`
	_, err := r.db.ExecContext(ctx, query, matchID)
	return err
}
