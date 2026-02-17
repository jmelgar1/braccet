package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/bracket/internal/domain"
)

var ErrMatchNotFound = errors.New("match not found")

type MatchRepository interface {
	CreateBatch(ctx context.Context, matches []*domain.Match) error
	GetByID(ctx context.Context, id uint64) (*domain.Match, error)
	GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.Match, error)
	UpdateResult(ctx context.Context, matchID uint64, result domain.MatchResult) error
	UpdateStatus(ctx context.Context, matchID uint64, status domain.MatchStatus) error
	SetParticipant(ctx context.Context, matchID uint64, slot int, participantID uint64, name string) error
	UpdateNextMatchLinks(ctx context.Context, matches []*domain.Match) error
}

type matchRepository struct {
	db *sql.DB
}

// pointer to the struct (basically a constructor)
func NewMatchRepository(db *sql.DB) MatchRepository {
	return &matchRepository{db: db}
}

func (r *matchRepository) CreateBatch(ctx context.Context, matches []*domain.Match) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO matches (tournament_id, bracket_type, round, position, participant1_id, participant2_id, participant1_name, participant2_name, status, scheduled_at, next_match_id, loser_match_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range matches {
		err := stmt.QueryRowContext(ctx,
			m.TournamentID, m.BracketType, m.Round, m.Position,
			m.Participant1ID, m.Participant2ID, m.Participant1Name, m.Participant2Name,
			m.Status, m.ScheduledAt, m.NextMatchID, m.LoserMatchID,
		).Scan(&m.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *matchRepository) GetByID(ctx context.Context, id uint64) (*domain.Match, error) {
	query := `
		SELECT id, tournament_id, bracket_type, round, position,
		       participant1_id, participant2_id, participant1_name, participant2_name,
		       winner_id, participant1_score, participant2_score, status,
		       scheduled_at, completed_at, next_match_id, loser_match_id, created_at, updated_at
		FROM matches
		WHERE id = $1
	`
	m := &domain.Match{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.TournamentID, &m.BracketType, &m.Round, &m.Position,
		&m.Participant1ID, &m.Participant2ID, &m.Participant1Name, &m.Participant2Name,
		&m.WinnerID, &m.Participant1Score, &m.Participant2Score, &m.Status,
		&m.ScheduledAt, &m.CompletedAt, &m.NextMatchID, &m.LoserMatchID, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	return m, nil
}

func (r *matchRepository) GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.Match, error) {
	query := `
		SELECT id, tournament_id, bracket_type, round, position,
		       participant1_id, participant2_id, participant1_name, participant2_name,
		       winner_id, participant1_score, participant2_score, status,
		       scheduled_at, completed_at, next_match_id, loser_match_id, created_at, updated_at
		FROM matches
		WHERE tournament_id = $1
		ORDER BY bracket_type, round, position
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*domain.Match
	for rows.Next() {
		// create new domain.Match and set m as a pointer to it
		m := &domain.Match{}
		// here we use & AGAIN because we need to get the pointers for each field.
		err := rows.Scan(
			&m.ID, &m.TournamentID, &m.BracketType, &m.Round, &m.Position,
			&m.Participant1ID, &m.Participant2ID, &m.Participant1Name, &m.Participant2Name,
			&m.WinnerID, &m.Participant1Score, &m.Participant2Score, &m.Status,
			&m.ScheduledAt, &m.CompletedAt, &m.NextMatchID, &m.LoserMatchID, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}

func (r *matchRepository) UpdateResult(ctx context.Context, matchID uint64, result domain.MatchResult) error {
	query := `
		UPDATE matches
		SET winner_id = $1, participant1_score = $2, participant2_score = $3, status = $4, completed_at = NOW()
		WHERE id = $5
	`
	res, err := r.db.ExecContext(ctx, query,
		result.WinnerID, result.Participant1Score, result.Participant2Score, domain.MatchCompleted, matchID,
	)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMatchNotFound
	}

	return nil
}

func (r *matchRepository) UpdateStatus(ctx context.Context, matchID uint64, status domain.MatchStatus) error {
	query := `UPDATE matches SET status = $1 WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, status, matchID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMatchNotFound
	}

	return nil
}

func (r *matchRepository) SetParticipant(ctx context.Context, matchID uint64, slot int, participantID uint64, name string) error {
	var query string
	if slot == 1 {
		query = `UPDATE matches SET participant1_id = $1, participant1_name = $2 WHERE id = $3`
	} else {
		query = `UPDATE matches SET participant2_id = $1, participant2_name = $2 WHERE id = $3`
	}

	res, err := r.db.ExecContext(ctx, query, participantID, name, matchID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMatchNotFound
	}

	return nil
}

func (r *matchRepository) UpdateNextMatchLinks(ctx context.Context, matches []*domain.Match) error {
	query := `UPDATE matches SET next_match_id = $1, loser_match_id = $2 WHERE id = $3`
	for _, m := range matches {
		_, err := r.db.ExecContext(ctx, query, m.NextMatchID, m.LoserMatchID, m.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
