package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/community/internal/domain"
)

var ErrEloHistoryNotFound = errors.New("elo history not found")

type EloHistoryRepository interface {
	Create(ctx context.Context, h *domain.EloHistory) error
	GetByMemberAndSystem(ctx context.Context, memberID, systemID uint64, limit int) ([]*domain.EloHistory, error)
	GetByMatch(ctx context.Context, matchID uint64) ([]*domain.EloHistory, error)
	GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.EloHistory, error)
	DeleteByMatch(ctx context.Context, matchID uint64) error
}

type eloHistoryRepository struct {
	db *sql.DB
}

func NewEloHistoryRepository(db *sql.DB) EloHistoryRepository {
	return &eloHistoryRepository{db: db}
}

func (r *eloHistoryRepository) Create(ctx context.Context, h *domain.EloHistory) error {
	query := `
		INSERT INTO elo_history (
			member_id, elo_system_id, change_type, rating_before, rating_change, rating_after,
			match_id, tournament_id, opponent_member_id, opponent_rating_before, is_winner,
			k_factor_used, expected_score, win_streak_bonus, notes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query,
		h.MemberID, h.EloSystemID, h.ChangeType, h.RatingBefore, h.RatingChange, h.RatingAfter,
		h.MatchID, h.TournamentID, h.OpponentMemberID, h.OpponentRatingBefore, h.IsWinner,
		h.KFactorUsed, h.ExpectedScore, h.WinStreakBonus, h.Notes,
	).Scan(&h.ID, &h.CreatedAt)

	return err
}

func (r *eloHistoryRepository) GetByMemberAndSystem(ctx context.Context, memberID, systemID uint64, limit int) ([]*domain.EloHistory, error) {
	query := `
		SELECT eh.id, eh.member_id, eh.elo_system_id, eh.change_type::text, eh.rating_before, eh.rating_change, eh.rating_after,
			eh.match_id, eh.tournament_id, eh.opponent_member_id, eh.opponent_rating_before, eh.is_winner,
			eh.k_factor_used, eh.expected_score, eh.win_streak_bonus, eh.notes, eh.created_at,
			cm.display_name
		FROM elo_history eh
		LEFT JOIN community_members cm ON cm.id = eh.opponent_member_id
		WHERE eh.member_id = $1 AND eh.elo_system_id = $2
		ORDER BY eh.created_at DESC
		LIMIT $3
	`
	rows, err := r.db.QueryContext(ctx, query, memberID, systemID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*domain.EloHistory
	for rows.Next() {
		h := &domain.EloHistory{}
		err := rows.Scan(
			&h.ID, &h.MemberID, &h.EloSystemID, &h.ChangeType, &h.RatingBefore, &h.RatingChange, &h.RatingAfter,
			&h.MatchID, &h.TournamentID, &h.OpponentMemberID, &h.OpponentRatingBefore, &h.IsWinner,
			&h.KFactorUsed, &h.ExpectedScore, &h.WinStreakBonus, &h.Notes, &h.CreatedAt,
			&h.OpponentDisplayName,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

func (r *eloHistoryRepository) GetByMatch(ctx context.Context, matchID uint64) ([]*domain.EloHistory, error) {
	query := `
		SELECT eh.id, eh.member_id, eh.elo_system_id, eh.change_type::text, eh.rating_before, eh.rating_change, eh.rating_after,
			eh.match_id, eh.tournament_id, eh.opponent_member_id, eh.opponent_rating_before, eh.is_winner,
			eh.k_factor_used, eh.expected_score, eh.win_streak_bonus, eh.notes, eh.created_at,
			cm.display_name
		FROM elo_history eh
		LEFT JOIN community_members cm ON cm.id = eh.opponent_member_id
		WHERE eh.match_id = $1
		ORDER BY eh.is_winner DESC
	`
	rows, err := r.db.QueryContext(ctx, query, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*domain.EloHistory
	for rows.Next() {
		h := &domain.EloHistory{}
		err := rows.Scan(
			&h.ID, &h.MemberID, &h.EloSystemID, &h.ChangeType, &h.RatingBefore, &h.RatingChange, &h.RatingAfter,
			&h.MatchID, &h.TournamentID, &h.OpponentMemberID, &h.OpponentRatingBefore, &h.IsWinner,
			&h.KFactorUsed, &h.ExpectedScore, &h.WinStreakBonus, &h.Notes, &h.CreatedAt,
			&h.OpponentDisplayName,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

func (r *eloHistoryRepository) GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.EloHistory, error) {
	query := `
		SELECT eh.id, eh.member_id, eh.elo_system_id, eh.change_type::text, eh.rating_before, eh.rating_change, eh.rating_after,
			eh.match_id, eh.tournament_id, eh.opponent_member_id, eh.opponent_rating_before, eh.is_winner,
			eh.k_factor_used, eh.expected_score, eh.win_streak_bonus, eh.notes, eh.created_at,
			cm.display_name
		FROM elo_history eh
		LEFT JOIN community_members cm ON cm.id = eh.opponent_member_id
		WHERE eh.tournament_id = $1
		ORDER BY eh.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*domain.EloHistory
	for rows.Next() {
		h := &domain.EloHistory{}
		err := rows.Scan(
			&h.ID, &h.MemberID, &h.EloSystemID, &h.ChangeType, &h.RatingBefore, &h.RatingChange, &h.RatingAfter,
			&h.MatchID, &h.TournamentID, &h.OpponentMemberID, &h.OpponentRatingBefore, &h.IsWinner,
			&h.KFactorUsed, &h.ExpectedScore, &h.WinStreakBonus, &h.Notes, &h.CreatedAt,
			&h.OpponentDisplayName,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

func (r *eloHistoryRepository) DeleteByMatch(ctx context.Context, matchID uint64) error {
	query := `DELETE FROM elo_history WHERE match_id = $1`
	_, err := r.db.ExecContext(ctx, query, matchID)
	return err
}
