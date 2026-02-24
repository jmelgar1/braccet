package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/braccet/bracket/internal/domain"
)

var (
	ErrTiebreakerNotFound = errors.New("tiebreaker not found")
)

type TiebreakerRepository interface {
	// Tiebreaker bracket operations
	Create(ctx context.Context, tb *domain.TiebreakerBracket) error
	GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.TiebreakerBracket, error)
	GetByID(ctx context.Context, id uint64) (*domain.TiebreakerBracket, error)
	GetByTournamentAndPosition(ctx context.Context, tournamentID uint64, tiePosition int) (*domain.TiebreakerBracket, error)
	MarkComplete(ctx context.Context, id uint64) error
	DeleteByTournament(ctx context.Context, tournamentID uint64) error

	// Tiebreaker result operations
	CreateResult(ctx context.Context, result *domain.TiebreakerResult) error
	GetResults(ctx context.Context, tiebreakerID uint64) ([]*domain.TiebreakerResult, error)
	GetResultsByTournament(ctx context.Context, tournamentID uint64) (map[uint64]int, error)
}

type tiebreakerRepository struct {
	db *sql.DB
}

func NewTiebreakerRepository(db *sql.DB) TiebreakerRepository {
	return &tiebreakerRepository{db: db}
}

// Tiebreaker bracket operations

func (r *tiebreakerRepository) Create(ctx context.Context, tb *domain.TiebreakerBracket) error {
	query := `
		INSERT INTO swiss_tiebreakers (tournament_id, tie_position, participant_ids, is_complete)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		tb.TournamentID, tb.TiePosition, pq.Array(tb.ParticipantIDs), tb.IsComplete,
	).Scan(&tb.ID, &tb.CreatedAt, &tb.UpdatedAt)
	return err
}

func (r *tiebreakerRepository) GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.TiebreakerBracket, error) {
	query := `
		SELECT id, tournament_id, tie_position, participant_ids, is_complete, created_at, updated_at
		FROM swiss_tiebreakers
		WHERE tournament_id = $1
		ORDER BY tie_position ASC
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tiebreakers []*domain.TiebreakerBracket
	for rows.Next() {
		tb := &domain.TiebreakerBracket{}
		var participantIDs pq.Int64Array
		err := rows.Scan(
			&tb.ID, &tb.TournamentID, &tb.TiePosition,
			&participantIDs, &tb.IsComplete,
			&tb.CreatedAt, &tb.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tb.ParticipantIDs = int64ArrayToUint64(participantIDs)
		tiebreakers = append(tiebreakers, tb)
	}

	return tiebreakers, rows.Err()
}

func (r *tiebreakerRepository) GetByID(ctx context.Context, id uint64) (*domain.TiebreakerBracket, error) {
	query := `
		SELECT id, tournament_id, tie_position, participant_ids, is_complete, created_at, updated_at
		FROM swiss_tiebreakers
		WHERE id = $1
	`
	tb := &domain.TiebreakerBracket{}
	var participantIDs pq.Int64Array
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tb.ID, &tb.TournamentID, &tb.TiePosition,
		&participantIDs, &tb.IsComplete,
		&tb.CreatedAt, &tb.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTiebreakerNotFound
		}
		return nil, err
	}
	tb.ParticipantIDs = int64ArrayToUint64(participantIDs)
	return tb, nil
}

func (r *tiebreakerRepository) GetByTournamentAndPosition(ctx context.Context, tournamentID uint64, tiePosition int) (*domain.TiebreakerBracket, error) {
	query := `
		SELECT id, tournament_id, tie_position, participant_ids, is_complete, created_at, updated_at
		FROM swiss_tiebreakers
		WHERE tournament_id = $1 AND tie_position = $2
	`
	tb := &domain.TiebreakerBracket{}
	var participantIDs pq.Int64Array
	err := r.db.QueryRowContext(ctx, query, tournamentID, tiePosition).Scan(
		&tb.ID, &tb.TournamentID, &tb.TiePosition,
		&participantIDs, &tb.IsComplete,
		&tb.CreatedAt, &tb.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTiebreakerNotFound
		}
		return nil, err
	}
	tb.ParticipantIDs = int64ArrayToUint64(participantIDs)
	return tb, nil
}

func (r *tiebreakerRepository) MarkComplete(ctx context.Context, id uint64) error {
	query := `
		UPDATE swiss_tiebreakers
		SET is_complete = true
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *tiebreakerRepository) DeleteByTournament(ctx context.Context, tournamentID uint64) error {
	// Results are deleted by CASCADE
	query := `DELETE FROM swiss_tiebreakers WHERE tournament_id = $1`
	_, err := r.db.ExecContext(ctx, query, tournamentID)
	return err
}

// Tiebreaker result operations

func (r *tiebreakerRepository) CreateResult(ctx context.Context, result *domain.TiebreakerResult) error {
	query := `
		INSERT INTO swiss_tiebreaker_results (tiebreaker_id, participant_id, final_position)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query,
		result.TiebreakerID, result.ParticipantID, result.FinalPosition,
	).Scan(&result.ID, &result.CreatedAt)
	return err
}

func (r *tiebreakerRepository) GetResults(ctx context.Context, tiebreakerID uint64) ([]*domain.TiebreakerResult, error) {
	query := `
		SELECT id, tiebreaker_id, participant_id, final_position, created_at
		FROM swiss_tiebreaker_results
		WHERE tiebreaker_id = $1
		ORDER BY final_position ASC
	`
	rows, err := r.db.QueryContext(ctx, query, tiebreakerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.TiebreakerResult
	for rows.Next() {
		r := &domain.TiebreakerResult{}
		err := rows.Scan(&r.ID, &r.TiebreakerID, &r.ParticipantID, &r.FinalPosition, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

func (r *tiebreakerRepository) GetResultsByTournament(ctx context.Context, tournamentID uint64) (map[uint64]int, error) {
	query := `
		SELECT tr.participant_id, tr.final_position
		FROM swiss_tiebreaker_results tr
		JOIN swiss_tiebreakers t ON tr.tiebreaker_id = t.id
		WHERE t.tournament_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[uint64]int)
	for rows.Next() {
		var participantID uint64
		var position int
		if err := rows.Scan(&participantID, &position); err != nil {
			return nil, err
		}
		results[participantID] = position
	}

	return results, rows.Err()
}

// Helper to convert pq.Int64Array to []uint64
func int64ArrayToUint64(arr pq.Int64Array) []uint64 {
	result := make([]uint64, len(arr))
	for i, v := range arr {
		result[i] = uint64(v)
	}
	return result
}
