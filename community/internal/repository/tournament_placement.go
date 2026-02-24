package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/braccet/community/internal/domain"
)

var ErrPlacementNotFound = errors.New("tournament placement not found")

type TournamentPlacementRepository interface {
	Create(ctx context.Context, p *domain.TournamentPlacement) error
	GetByID(ctx context.Context, id uint64) (*domain.TournamentPlacement, error)
	GetByMember(ctx context.Context, memberID, systemID uint64) ([]*domain.TournamentPlacement, error)
	GetByMemberSince(ctx context.Context, memberID, systemID uint64, since time.Time) ([]*domain.TournamentPlacement, error)
	GetLANByMember(ctx context.Context, memberID, systemID uint64, limit int) ([]*domain.TournamentPlacement, error)
	GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.TournamentPlacement, error)
	Upsert(ctx context.Context, p *domain.TournamentPlacement) error
	DeleteByTournament(ctx context.Context, tournamentID uint64) error
}

type tournamentPlacementRepository struct {
	db *sql.DB
}

func NewTournamentPlacementRepository(db *sql.DB) TournamentPlacementRepository {
	return &tournamentPlacementRepository{db: db}
}

func (r *tournamentPlacementRepository) Create(ctx context.Context, p *domain.TournamentPlacement) error {
	query := `
		INSERT INTO tournament_placements (
			member_id, pr_system_id, tournament_id,
			placement, participant_count,
			tournament_class, prize_pool_usd, is_bo3_playoffs, avg_opponent_rating,
			is_lan, raw_points, completed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		p.MemberID, p.PRSystemID, p.TournamentID,
		p.Placement, p.ParticipantCount,
		p.TournamentClass, p.PrizePoolUSD, p.IsBo3Playoffs, p.AvgOpponentRating,
		p.IsLAN, p.RawPoints, p.CompletedAt,
	).Scan(&p.ID, &p.CreatedAt)
}

func (r *tournamentPlacementRepository) GetByID(ctx context.Context, id uint64) (*domain.TournamentPlacement, error) {
	query := `
		SELECT id, member_id, pr_system_id, tournament_id,
			placement, participant_count,
			tournament_class, prize_pool_usd, is_bo3_playoffs, avg_opponent_rating,
			is_lan, raw_points, completed_at, created_at
		FROM tournament_placements
		WHERE id = $1
	`
	p := &domain.TournamentPlacement{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.MemberID, &p.PRSystemID, &p.TournamentID,
		&p.Placement, &p.ParticipantCount,
		&p.TournamentClass, &p.PrizePoolUSD, &p.IsBo3Playoffs, &p.AvgOpponentRating,
		&p.IsLAN, &p.RawPoints, &p.CompletedAt, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlacementNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *tournamentPlacementRepository) GetByMember(ctx context.Context, memberID, systemID uint64) ([]*domain.TournamentPlacement, error) {
	query := `
		SELECT id, member_id, pr_system_id, tournament_id,
			placement, participant_count,
			tournament_class, prize_pool_usd, is_bo3_playoffs, avg_opponent_rating,
			is_lan, raw_points, completed_at, created_at
		FROM tournament_placements
		WHERE member_id = $1 AND pr_system_id = $2
		ORDER BY completed_at DESC
	`
	return r.queryPlacements(ctx, query, memberID, systemID)
}

func (r *tournamentPlacementRepository) GetByMemberSince(ctx context.Context, memberID, systemID uint64, since time.Time) ([]*domain.TournamentPlacement, error) {
	query := `
		SELECT id, member_id, pr_system_id, tournament_id,
			placement, participant_count,
			tournament_class, prize_pool_usd, is_bo3_playoffs, avg_opponent_rating,
			is_lan, raw_points, completed_at, created_at
		FROM tournament_placements
		WHERE member_id = $1 AND pr_system_id = $2 AND completed_at >= $3
		ORDER BY completed_at DESC
	`
	return r.queryPlacements(ctx, query, memberID, systemID, since)
}

func (r *tournamentPlacementRepository) GetLANByMember(ctx context.Context, memberID, systemID uint64, limit int) ([]*domain.TournamentPlacement, error) {
	query := `
		SELECT id, member_id, pr_system_id, tournament_id,
			placement, participant_count,
			tournament_class, prize_pool_usd, is_bo3_playoffs, avg_opponent_rating,
			is_lan, raw_points, completed_at, created_at
		FROM tournament_placements
		WHERE member_id = $1 AND pr_system_id = $2 AND is_lan = true
		ORDER BY completed_at DESC
		LIMIT $3
	`
	return r.queryPlacements(ctx, query, memberID, systemID, limit)
}

func (r *tournamentPlacementRepository) GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.TournamentPlacement, error) {
	query := `
		SELECT id, member_id, pr_system_id, tournament_id,
			placement, participant_count,
			tournament_class, prize_pool_usd, is_bo3_playoffs, avg_opponent_rating,
			is_lan, raw_points, completed_at, created_at
		FROM tournament_placements
		WHERE tournament_id = $1
		ORDER BY placement ASC
	`
	return r.queryPlacements(ctx, query, tournamentID)
}

func (r *tournamentPlacementRepository) queryPlacements(ctx context.Context, query string, args ...any) ([]*domain.TournamentPlacement, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var placements []*domain.TournamentPlacement
	for rows.Next() {
		p := &domain.TournamentPlacement{}
		err := rows.Scan(
			&p.ID, &p.MemberID, &p.PRSystemID, &p.TournamentID,
			&p.Placement, &p.ParticipantCount,
			&p.TournamentClass, &p.PrizePoolUSD, &p.IsBo3Playoffs, &p.AvgOpponentRating,
			&p.IsLAN, &p.RawPoints, &p.CompletedAt, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		placements = append(placements, p)
	}
	return placements, rows.Err()
}

func (r *tournamentPlacementRepository) Upsert(ctx context.Context, p *domain.TournamentPlacement) error {
	query := `
		INSERT INTO tournament_placements (
			member_id, pr_system_id, tournament_id,
			placement, participant_count,
			tournament_class, prize_pool_usd, is_bo3_playoffs, avg_opponent_rating,
			is_lan, raw_points, completed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (member_id, tournament_id) DO UPDATE SET
			placement = EXCLUDED.placement,
			participant_count = EXCLUDED.participant_count,
			tournament_class = EXCLUDED.tournament_class,
			prize_pool_usd = EXCLUDED.prize_pool_usd,
			is_bo3_playoffs = EXCLUDED.is_bo3_playoffs,
			avg_opponent_rating = EXCLUDED.avg_opponent_rating,
			is_lan = EXCLUDED.is_lan,
			raw_points = EXCLUDED.raw_points,
			completed_at = EXCLUDED.completed_at
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		p.MemberID, p.PRSystemID, p.TournamentID,
		p.Placement, p.ParticipantCount,
		p.TournamentClass, p.PrizePoolUSD, p.IsBo3Playoffs, p.AvgOpponentRating,
		p.IsLAN, p.RawPoints, p.CompletedAt,
	).Scan(&p.ID, &p.CreatedAt)
}

func (r *tournamentPlacementRepository) DeleteByTournament(ctx context.Context, tournamentID uint64) error {
	query := `DELETE FROM tournament_placements WHERE tournament_id = $1`
	_, err := r.db.ExecContext(ctx, query, tournamentID)
	return err
}
