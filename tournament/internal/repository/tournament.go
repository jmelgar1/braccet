package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/tournament/internal/domain"
)

var ErrTournamentNotFound = errors.New("tournament not found")

type TournamentRepository interface {
	Create(ctx context.Context, t *domain.Tournament) error
	GetByID(ctx context.Context, id uint64) (*domain.Tournament, error)
	Update(ctx context.Context, t *domain.Tournament) error
	Delete(ctx context.Context, id uint64) error
	ListByOrganizer(ctx context.Context, organizerID uint64) ([]*domain.Tournament, error)
	ListByStatus(ctx context.Context, status domain.TournamentStatus) ([]*domain.Tournament, error)
}

type tournamentRepository struct {
	db *sql.DB
}

func NewTournamentRepository(db *sql.DB) TournamentRepository {
	return &tournamentRepository{db: db}
}

func (r *tournamentRepository) Create(ctx context.Context, t *domain.Tournament) error {
	query := `
		INSERT INTO tournaments (organizer_id, name, description, game, format, status, max_participants, registration_open, settings, starts_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		t.OrganizerID, t.Name, t.Description, t.Game, t.Format, t.Status,
		t.MaxParticipants, t.RegistrationOpen, t.Settings, t.StartsAt,
	).Scan(&t.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *tournamentRepository) GetByID(ctx context.Context, id uint64) (*domain.Tournament, error) {
	query := `
		SELECT id, organizer_id, name, description, game, format, status, max_participants, registration_open, settings, starts_at, created_at, updated_at
		FROM tournaments
		WHERE id = $1
	`
	t := &domain.Tournament{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.OrganizerID, &t.Name, &t.Description, &t.Game, &t.Format, &t.Status,
		&t.MaxParticipants, &t.RegistrationOpen, &t.Settings, &t.StartsAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTournamentNotFound
		}
		return nil, err
	}

	return t, nil
}

func (r *tournamentRepository) Update(ctx context.Context, t *domain.Tournament) error {
	query := `
		UPDATE tournaments
		SET name = $1, description = $2, game = $3, format = $4, status = $5, max_participants = $6, registration_open = $7, settings = $8, starts_at = $9
		WHERE id = $10
	`
	result, err := r.db.ExecContext(ctx, query,
		t.Name, t.Description, t.Game, t.Format, t.Status,
		t.MaxParticipants, t.RegistrationOpen, t.Settings, t.StartsAt, t.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTournamentNotFound
	}

	return nil
}

func (r *tournamentRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM tournaments WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTournamentNotFound
	}

	return nil
}

func (r *tournamentRepository) ListByOrganizer(ctx context.Context, organizerID uint64) ([]*domain.Tournament, error) {
	query := `
		SELECT id, organizer_id, name, description, game, format, status, max_participants, registration_open, settings, starts_at, created_at, updated_at
		FROM tournaments
		WHERE organizer_id = $1
		ORDER BY created_at DESC
	`
	return r.queryTournaments(ctx, query, organizerID)
}

func (r *tournamentRepository) ListByStatus(ctx context.Context, status domain.TournamentStatus) ([]*domain.Tournament, error) {
	query := `
		SELECT id, organizer_id, name, description, game, format, status, max_participants, registration_open, settings, starts_at, created_at, updated_at
		FROM tournaments
		WHERE status = $1
		ORDER BY created_at DESC
	`
	return r.queryTournaments(ctx, query, status)
}

func (r *tournamentRepository) queryTournaments(ctx context.Context, query string, args ...any) ([]*domain.Tournament, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tournaments []*domain.Tournament
	for rows.Next() {
		t := &domain.Tournament{}
		err := rows.Scan(
			&t.ID, &t.OrganizerID, &t.Name, &t.Description, &t.Game, &t.Format, &t.Status,
			&t.MaxParticipants, &t.RegistrationOpen, &t.Settings, &t.StartsAt, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tournaments = append(tournaments, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tournaments, nil
}
