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
	GetByTournamentAndUser(ctx context.Context, tournamentID, userID uint64) (*domain.Participant, error)
	CountByTournament(ctx context.Context, tournamentID uint64) (int, error)
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
		INSERT INTO participants (tournament_id, user_id, community_member_id, display_name, seed, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		p.TournamentID, p.UserID, p.CommunityMemberID, p.DisplayName, p.Seed, p.Status,
	).Scan(&p.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *participantRepository) GetByID(ctx context.Context, id uint64) (*domain.Participant, error) {
	query := `
		SELECT id, tournament_id, user_id, community_member_id, display_name, seed, status, checked_in_at, created_at
		FROM participants
		WHERE id = $1
	`
	p := &domain.Participant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.TournamentID, &p.UserID, &p.CommunityMemberID, &p.DisplayName, &p.Seed, &p.Status, &p.CheckedInAt, &p.CreatedAt,
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
		SELECT id, tournament_id, user_id, community_member_id, display_name, seed, status, checked_in_at, created_at
		FROM participants
		WHERE tournament_id = $1
		ORDER BY seed ASC NULLS LAST, created_at ASC
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
			&p.ID, &p.TournamentID, &p.UserID, &p.CommunityMemberID, &p.DisplayName, &p.Seed, &p.Status, &p.CheckedInAt, &p.CreatedAt,
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

func (r *participantRepository) GetByTournamentAndUser(ctx context.Context, tournamentID, userID uint64) (*domain.Participant, error) {
	query := `
		SELECT id, tournament_id, user_id, community_member_id, display_name, seed, status, checked_in_at, created_at
		FROM participants
		WHERE tournament_id = $1 AND user_id = $2
	`
	p := &domain.Participant{}
	err := r.db.QueryRowContext(ctx, query, tournamentID, userID).Scan(
		&p.ID, &p.TournamentID, &p.UserID, &p.CommunityMemberID, &p.DisplayName, &p.Seed, &p.Status, &p.CheckedInAt, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrParticipantNotFound
		}
		return nil, err
	}

	return p, nil
}

func (r *participantRepository) CountByTournament(ctx context.Context, tournamentID uint64) (int, error) {
	query := `SELECT COUNT(*) FROM participants WHERE tournament_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, tournamentID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *participantRepository) UpdateSeeding(ctx context.Context, tournamentID uint64, seeds map[uint64]uint) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE participants SET seed = $1 WHERE id = $2 AND tournament_id = $3`
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
	query := `UPDATE participants SET status = $1 WHERE id = $2`
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
	query := `DELETE FROM participants WHERE id = $1`
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
