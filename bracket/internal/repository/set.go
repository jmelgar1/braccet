package repository

import (
	"context"
	"database/sql"

	"github.com/braccet/bracket/internal/domain"
	"github.com/lib/pq"
)

type SetRepository interface {
	GetByMatchID(ctx context.Context, matchID uint64) ([]domain.Set, error)
	GetByMatchIDs(ctx context.Context, matchIDs []uint64) (map[uint64][]domain.Set, error)
	CreateBatch(ctx context.Context, matchID uint64, sets []domain.SetScore) error
	DeleteByMatchID(ctx context.Context, matchID uint64) error
}

type setRepository struct {
	db *sql.DB
}

func NewSetRepository(db *sql.DB) SetRepository {
	return &setRepository{db: db}
}

func (r *setRepository) GetByMatchID(ctx context.Context, matchID uint64) ([]domain.Set, error) {
	query := `
		SELECT id, match_id, set_number, participant1_score, participant2_score, created_at, updated_at
		FROM match_sets
		WHERE match_id = $1
		ORDER BY set_number
	`
	rows, err := r.db.QueryContext(ctx, query, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sets []domain.Set
	for rows.Next() {
		var s domain.Set
		err := rows.Scan(
			&s.ID, &s.MatchID, &s.SetNumber,
			&s.Participant1Score, &s.Participant2Score,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sets = append(sets, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sets, nil
}

func (r *setRepository) GetByMatchIDs(ctx context.Context, matchIDs []uint64) (map[uint64][]domain.Set, error) {
	if len(matchIDs) == 0 {
		return make(map[uint64][]domain.Set), nil
	}

	// Build query with IN clause using pq.Array for PostgreSQL
	query := `
		SELECT id, match_id, set_number, participant1_score, participant2_score, created_at, updated_at
		FROM match_sets
		WHERE match_id = ANY($1)
		ORDER BY match_id, set_number
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(matchIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uint64][]domain.Set)
	for rows.Next() {
		var s domain.Set
		err := rows.Scan(
			&s.ID, &s.MatchID, &s.SetNumber,
			&s.Participant1Score, &s.Participant2Score,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result[s.MatchID] = append(result[s.MatchID], s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *setRepository) CreateBatch(ctx context.Context, matchID uint64, sets []domain.SetScore) error {
	if len(sets) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing sets for this match (allows re-entry)
	_, err = tx.ExecContext(ctx, "DELETE FROM match_sets WHERE match_id = $1", matchID)
	if err != nil {
		return err
	}

	// Insert new sets
	query := `
		INSERT INTO match_sets (match_id, set_number, participant1_score, participant2_score)
		VALUES ($1, $2, $3, $4)
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range sets {
		_, err := stmt.ExecContext(ctx, matchID, s.SetNumber, s.Participant1Score, s.Participant2Score)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *setRepository) DeleteByMatchID(ctx context.Context, matchID uint64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM match_sets WHERE match_id = $1", matchID)
	return err
}
