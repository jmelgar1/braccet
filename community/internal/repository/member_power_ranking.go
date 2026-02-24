package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/community/internal/domain"
)

var ErrMemberPRNotFound = errors.New("member power ranking not found")

type MemberPowerRankingRepository interface {
	Create(ctx context.Context, r *domain.MemberPowerRanking) error
	GetByMemberAndSystem(ctx context.Context, memberID, systemID uint64) (*domain.MemberPowerRanking, error)
	GetLeaderboard(ctx context.Context, systemID uint64, limit int) ([]*domain.MemberPowerRanking, error)
	Update(ctx context.Context, r *domain.MemberPowerRanking) error
	Upsert(ctx context.Context, r *domain.MemberPowerRanking) error
	RecalculateRanks(ctx context.Context, systemID uint64) error
	DeleteBySystem(ctx context.Context, systemID uint64) error
	GetMembersWithActivity(ctx context.Context, systemID uint64) ([]uint64, error)
}

type memberPowerRankingRepository struct {
	db *sql.DB
}

func NewMemberPowerRankingRepository(db *sql.DB) MemberPowerRankingRepository {
	return &memberPowerRankingRepository{db: db}
}

func (r *memberPowerRankingRepository) Create(ctx context.Context, m *domain.MemberPowerRanking) error {
	query := `
		INSERT INTO member_power_rankings (
			member_id, pr_system_id,
			total_points, rank,
			achievements_score, form_score, lan_score,
			form_wins, form_set_wins, form_set_losses, form_avg_opponent_rating, form_best_win_rating,
			lan_placements_count, last_calculated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		m.MemberID, m.PRSystemID,
		m.TotalPoints, m.Rank,
		m.AchievementsScore, m.FormScore, m.LANScore,
		m.FormWins, m.FormSetWins, m.FormSetLosses, m.FormAvgOpponentRating, m.FormBestWinRating,
		m.LANPlacementsCount, m.LastCalculatedAt,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *memberPowerRankingRepository) GetByMemberAndSystem(ctx context.Context, memberID, systemID uint64) (*domain.MemberPowerRanking, error) {
	query := `
		SELECT mpr.id, mpr.member_id, mpr.pr_system_id,
			mpr.total_points, mpr.rank,
			mpr.achievements_score, mpr.form_score, mpr.lan_score,
			mpr.form_wins, mpr.form_set_wins, mpr.form_set_losses, mpr.form_avg_opponent_rating, mpr.form_best_win_rating,
			mpr.lan_placements_count, mpr.last_calculated_at, mpr.created_at, mpr.updated_at,
			cm.display_name, cm.region, cm.icon_url
		FROM member_power_rankings mpr
		JOIN community_members cm ON cm.id = mpr.member_id
		WHERE mpr.member_id = $1 AND mpr.pr_system_id = $2
	`
	m := &domain.MemberPowerRanking{}
	err := r.db.QueryRowContext(ctx, query, memberID, systemID).Scan(
		&m.ID, &m.MemberID, &m.PRSystemID,
		&m.TotalPoints, &m.Rank,
		&m.AchievementsScore, &m.FormScore, &m.LANScore,
		&m.FormWins, &m.FormSetWins, &m.FormSetLosses, &m.FormAvgOpponentRating, &m.FormBestWinRating,
		&m.LANPlacementsCount, &m.LastCalculatedAt, &m.CreatedAt, &m.UpdatedAt,
		&m.MemberDisplayName, &m.MemberRegion, &m.MemberIconURL,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMemberPRNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *memberPowerRankingRepository) GetLeaderboard(ctx context.Context, systemID uint64, limit int) ([]*domain.MemberPowerRanking, error) {
	query := `
		SELECT mpr.id, mpr.member_id, mpr.pr_system_id,
			mpr.total_points, mpr.rank,
			mpr.achievements_score, mpr.form_score, mpr.lan_score,
			mpr.form_wins, mpr.form_set_wins, mpr.form_set_losses, mpr.form_avg_opponent_rating, mpr.form_best_win_rating,
			mpr.lan_placements_count, mpr.last_calculated_at, mpr.created_at, mpr.updated_at,
			cm.display_name, cm.region, cm.icon_url
		FROM member_power_rankings mpr
		JOIN community_members cm ON cm.id = mpr.member_id
		WHERE mpr.pr_system_id = $1 AND mpr.total_points > 0
		ORDER BY mpr.total_points DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, systemID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rankings []*domain.MemberPowerRanking
	for rows.Next() {
		m := &domain.MemberPowerRanking{}
		err := rows.Scan(
			&m.ID, &m.MemberID, &m.PRSystemID,
			&m.TotalPoints, &m.Rank,
			&m.AchievementsScore, &m.FormScore, &m.LANScore,
			&m.FormWins, &m.FormSetWins, &m.FormSetLosses, &m.FormAvgOpponentRating, &m.FormBestWinRating,
			&m.LANPlacementsCount, &m.LastCalculatedAt, &m.CreatedAt, &m.UpdatedAt,
			&m.MemberDisplayName, &m.MemberRegion, &m.MemberIconURL,
		)
		if err != nil {
			return nil, err
		}
		rankings = append(rankings, m)
	}
	return rankings, rows.Err()
}

func (r *memberPowerRankingRepository) Update(ctx context.Context, m *domain.MemberPowerRanking) error {
	query := `
		UPDATE member_power_rankings SET
			total_points = $1, rank = $2,
			achievements_score = $3, form_score = $4, lan_score = $5,
			form_wins = $6, form_set_wins = $7, form_set_losses = $8, form_avg_opponent_rating = $9, form_best_win_rating = $10,
			lan_placements_count = $11, last_calculated_at = $12
		WHERE id = $13
	`
	result, err := r.db.ExecContext(ctx, query,
		m.TotalPoints, m.Rank,
		m.AchievementsScore, m.FormScore, m.LANScore,
		m.FormWins, m.FormSetWins, m.FormSetLosses, m.FormAvgOpponentRating, m.FormBestWinRating,
		m.LANPlacementsCount, m.LastCalculatedAt, m.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMemberPRNotFound
	}

	return nil
}

func (r *memberPowerRankingRepository) Upsert(ctx context.Context, m *domain.MemberPowerRanking) error {
	query := `
		INSERT INTO member_power_rankings (
			member_id, pr_system_id,
			total_points, rank,
			achievements_score, form_score, lan_score,
			form_wins, form_set_wins, form_set_losses, form_avg_opponent_rating, form_best_win_rating,
			lan_placements_count, last_calculated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (member_id, pr_system_id) DO UPDATE SET
			total_points = EXCLUDED.total_points,
			rank = EXCLUDED.rank,
			achievements_score = EXCLUDED.achievements_score,
			form_score = EXCLUDED.form_score,
			lan_score = EXCLUDED.lan_score,
			form_wins = EXCLUDED.form_wins,
			form_set_wins = EXCLUDED.form_set_wins,
			form_set_losses = EXCLUDED.form_set_losses,
			form_avg_opponent_rating = EXCLUDED.form_avg_opponent_rating,
			form_best_win_rating = EXCLUDED.form_best_win_rating,
			lan_placements_count = EXCLUDED.lan_placements_count,
			last_calculated_at = EXCLUDED.last_calculated_at
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		m.MemberID, m.PRSystemID,
		m.TotalPoints, m.Rank,
		m.AchievementsScore, m.FormScore, m.LANScore,
		m.FormWins, m.FormSetWins, m.FormSetLosses, m.FormAvgOpponentRating, m.FormBestWinRating,
		m.LANPlacementsCount, m.LastCalculatedAt,
	).Scan(&m.ID, &m.CreatedAt, &m.UpdatedAt)
}

func (r *memberPowerRankingRepository) RecalculateRanks(ctx context.Context, systemID uint64) error {
	// Update ranks based on total_points ordering
	query := `
		WITH ranked AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY total_points DESC) as new_rank
			FROM member_power_rankings
			WHERE pr_system_id = $1 AND total_points > 0
		)
		UPDATE member_power_rankings mpr
		SET rank = ranked.new_rank
		FROM ranked
		WHERE mpr.id = ranked.id
	`
	_, err := r.db.ExecContext(ctx, query, systemID)
	return err
}

func (r *memberPowerRankingRepository) DeleteBySystem(ctx context.Context, systemID uint64) error {
	query := `DELETE FROM member_power_rankings WHERE pr_system_id = $1`
	_, err := r.db.ExecContext(ctx, query, systemID)
	return err
}

func (r *memberPowerRankingRepository) GetMembersWithActivity(ctx context.Context, systemID uint64) ([]uint64, error) {
	// Get all members that have either placements or match cache entries
	query := `
		SELECT DISTINCT member_id FROM (
			SELECT member_id FROM tournament_placements WHERE pr_system_id = $1
			UNION
			SELECT member_id FROM power_ranking_match_cache WHERE pr_system_id = $1
		) AS active_members
	`
	rows, err := r.db.QueryContext(ctx, query, systemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memberIDs []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		memberIDs = append(memberIDs, id)
	}
	return memberIDs, rows.Err()
}
