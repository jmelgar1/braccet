package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/tournament/internal/domain"
)

var ErrParticipantNotFound = errors.New("participant not found")

type ParticipantRepository interface {
	Create(ctx context.Context, p *domain.Participant) error
	GetByID(ctx context.Context, id uint64) (*domain.Participant, error)
	GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.Participant, error)
	UpdateSeeding(ctx context.Context, tournamentID uint64, seeds map[uint64]uint) error
	UpdateStatus(ctx context.Context, id uint64, status domain.ParticipantStatus) error
	Delete(ctx context.Context, id uint64) error
}

type participantRepository struct {
	db *sql.DB
}

func NewParticipantRepository(db *sql.DB) ParticipantRepository {
	return &participantRepository{db: db}
}

func (r *participantRepository) Create(ctx context.Context, p *domain.Participant) error {
	query := `
		INSERT INTO participants (tournament_id, user_id, display_name, seed, status)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		p.TournamentID, p.UserID, p.DisplayName, p.Seed, p.Status,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = uint64(id)

	return nil
}

func (r *participantRepository) GetByID(ctx context.Context, id uint64) (*domain.Participant, error) {
	query := `
		SELECT id, tournament_id, user_id, display_name, seed, status, checked_in_at, created_at
		FROM participants
		WHERE id = ?
	`
	p := &domain.Participant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.TournamentID, &p.UserID, &p.DisplayName, &p.Seed, &p.Status, &p.CheckedInAt, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrParticipantNotFound
		}
		return nil, err
	}

	return p, nil
}

func (r *participantRepository) GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.Participant, error) {
	query := `
		SELECT id, tournament_id, user_id, display_name, seed, status, checked_in_at, created_at
		FROM participants
		WHERE tournament_id = ?
		ORDER BY seed ASC, created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*domain.Participant
	for rows.Next() {
		p := &domain.Participant{}
		err := rows.Scan(
			&p.ID, &p.TournamentID, &p.UserID, &p.DisplayName, &p.Seed, &p.Status, &p.CheckedInAt, &p.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return participants, nil
}

func (r *participantRepository) UpdateSeeding(ctx context.Context, tournamentID uint64, seeds map[uint64]uint) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE participants SET seed = ? WHERE id = ? AND tournament_id = ?`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for participantID, seed := range seeds {
		_, err := stmt.ExecContext(ctx, seed, participantID, tournamentID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *participantRepository) UpdateStatus(ctx context.Context, id uint64, status domain.ParticipantStatus) error {
	query := `UPDATE participants SET status = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrParticipantNotFound
	}

	return nil
}

func (r *participantRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM participants WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrParticipantNotFound
	}

	return nil
}
