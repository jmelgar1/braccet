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
	GetBySlug(ctx context.Context, slug string) (*domain.Tournament, error)
	GetByID(ctx context.Context, id uint64) (*domain.Tournament, error)
	Update(ctx context.Context, t *domain.Tournament) error
	Delete(ctx context.Context, slug string) error
	ListByOrganizer(ctx context.Context, organizerID uint64) ([]*domain.Tournament, error)
	ListByStatus(ctx context.Context, status domain.TournamentStatus) ([]*domain.Tournament, error)
	ListByCommunityID(ctx context.Context, communityID uint64) ([]*domain.Tournament, error)
	ListAvailableForEvent(ctx context.Context, communityID uint64) ([]*domain.Tournament, error)
}

type tournamentRepository struct {
	db *sql.DB
}

func NewTournamentRepository(db *sql.DB) TournamentRepository {
	return &tournamentRepository{db: db}
}

func (r *tournamentRepository) Create(ctx context.Context, t *domain.Tournament) error {
	query := `
		INSERT INTO tournaments (slug, organizer_id, community_id, elo_system_id, pr_system_id, name, description, game, format, status, max_participants, swiss_rounds, registration_open, settings, starts_at, tournament_class, prize_pool_usd, logo_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		t.Slug, t.OrganizerID, t.CommunityID, t.EloSystemID, t.PRSystemID, t.Name, t.Description, t.Game, t.Format, t.Status,
		t.MaxParticipants, t.SwissRounds, t.RegistrationOpen, t.Settings, t.StartsAt, t.TournamentClass, t.PrizePoolUSD, t.LogoURL,
	).Scan(&t.ID)
	if err != nil {
		return err
	}

	return nil
}

func (r *tournamentRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tournament, error) {
	query := `
		SELECT id, slug, organizer_id, community_id, elo_system_id, pr_system_id, name, description, game, format::text, status::text, max_participants, swiss_rounds, registration_open, COALESCE(settings, '{}'), starts_at, tournament_class, prize_pool_usd, logo_url, created_at, updated_at
		FROM tournaments
		WHERE LOWER(slug) = LOWER($1)
	`
	t := &domain.Tournament{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&t.ID, &t.Slug, &t.OrganizerID, &t.CommunityID, &t.EloSystemID, &t.PRSystemID, &t.Name, &t.Description, &t.Game, &t.Format, &t.Status,
		&t.MaxParticipants, &t.SwissRounds, &t.RegistrationOpen, &t.Settings, &t.StartsAt, &t.TournamentClass, &t.PrizePoolUSD, &t.LogoURL, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTournamentNotFound
		}
		return nil, err
	}

	return t, nil
}

func (r *tournamentRepository) GetByID(ctx context.Context, id uint64) (*domain.Tournament, error) {
	query := `
		SELECT id, slug, organizer_id, community_id, elo_system_id, pr_system_id, name, description, game, format::text, status::text, max_participants, swiss_rounds, registration_open, COALESCE(settings, '{}'), starts_at, tournament_class, prize_pool_usd, logo_url, created_at, updated_at
		FROM tournaments
		WHERE id = $1
	`
	t := &domain.Tournament{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.Slug, &t.OrganizerID, &t.CommunityID, &t.EloSystemID, &t.PRSystemID, &t.Name, &t.Description, &t.Game, &t.Format, &t.Status,
		&t.MaxParticipants, &t.SwissRounds, &t.RegistrationOpen, &t.Settings, &t.StartsAt, &t.TournamentClass, &t.PrizePoolUSD, &t.LogoURL, &t.CreatedAt, &t.UpdatedAt,
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
		SET name = $1, description = $2, game = $3, format = $4, status = $5, max_participants = $6, swiss_rounds = $7, registration_open = $8, settings = $9, starts_at = $10, community_id = $11, elo_system_id = $12, pr_system_id = $13, tournament_class = $14, prize_pool_usd = $15, logo_url = $16
		WHERE LOWER(slug) = LOWER($17)
	`
	result, err := r.db.ExecContext(ctx, query,
		t.Name, t.Description, t.Game, t.Format, t.Status,
		t.MaxParticipants, t.SwissRounds, t.RegistrationOpen, t.Settings, t.StartsAt, t.CommunityID, t.EloSystemID, t.PRSystemID, t.TournamentClass, t.PrizePoolUSD, t.LogoURL, t.Slug,
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

func (r *tournamentRepository) Delete(ctx context.Context, slug string) error {
	query := `DELETE FROM tournaments WHERE LOWER(slug) = LOWER($1)`
	result, err := r.db.ExecContext(ctx, query, slug)
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
		SELECT t.id, t.slug, t.organizer_id, t.community_id, t.elo_system_id, t.pr_system_id, t.name, t.description, t.game, t.format::text, t.status::text, t.max_participants, t.swiss_rounds, t.registration_open, COALESCE(t.settings, '{}'), t.starts_at, t.tournament_class, t.prize_pool_usd, t.logo_url, et.event_id, et.role::text, t.created_at, t.updated_at
		FROM tournaments t
		LEFT JOIN event_tournaments et ON t.id = et.tournament_id
		WHERE t.organizer_id = $1
		ORDER BY t.created_at DESC
	`
	return r.queryTournamentsWithEvent(ctx, query, organizerID)
}

func (r *tournamentRepository) ListByStatus(ctx context.Context, status domain.TournamentStatus) ([]*domain.Tournament, error) {
	query := `
		SELECT id, slug, organizer_id, community_id, elo_system_id, pr_system_id, name, description, game, format::text, status::text, max_participants, swiss_rounds, registration_open, COALESCE(settings, '{}'), starts_at, tournament_class, prize_pool_usd, logo_url, created_at, updated_at
		FROM tournaments
		WHERE status = $1
		ORDER BY created_at DESC
	`
	return r.queryTournaments(ctx, query, status)
}

func (r *tournamentRepository) ListByCommunityID(ctx context.Context, communityID uint64) ([]*domain.Tournament, error) {
	query := `
		SELECT t.id, t.slug, t.organizer_id, t.community_id, t.elo_system_id, t.pr_system_id, t.name, t.description, t.game, t.format::text, t.status::text, t.max_participants, t.swiss_rounds, t.registration_open, COALESCE(t.settings, '{}'), t.starts_at, t.tournament_class, t.prize_pool_usd, t.logo_url, et.event_id, et.role::text, t.created_at, t.updated_at
		FROM tournaments t
		LEFT JOIN event_tournaments et ON t.id = et.tournament_id
		WHERE t.community_id = $1
		ORDER BY t.created_at DESC
	`
	return r.queryTournamentsWithEvent(ctx, query, communityID)
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
			&t.ID, &t.Slug, &t.OrganizerID, &t.CommunityID, &t.EloSystemID, &t.PRSystemID, &t.Name, &t.Description, &t.Game, &t.Format, &t.Status,
			&t.MaxParticipants, &t.SwissRounds, &t.RegistrationOpen, &t.Settings, &t.StartsAt, &t.TournamentClass, &t.PrizePoolUSD, &t.LogoURL, &t.CreatedAt, &t.UpdatedAt,
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

func (r *tournamentRepository) queryTournamentsWithEvent(ctx context.Context, query string, args ...any) ([]*domain.Tournament, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tournaments []*domain.Tournament
	for rows.Next() {
		t := &domain.Tournament{}
		err := rows.Scan(
			&t.ID, &t.Slug, &t.OrganizerID, &t.CommunityID, &t.EloSystemID, &t.PRSystemID, &t.Name, &t.Description, &t.Game, &t.Format, &t.Status,
			&t.MaxParticipants, &t.SwissRounds, &t.RegistrationOpen, &t.Settings, &t.StartsAt, &t.TournamentClass, &t.PrizePoolUSD, &t.LogoURL, &t.EventID, &t.EventRole, &t.CreatedAt, &t.UpdatedAt,
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

func (r *tournamentRepository) ListAvailableForEvent(ctx context.Context, communityID uint64) ([]*domain.Tournament, error) {
	query := `
		SELECT id, slug, organizer_id, community_id, elo_system_id, pr_system_id, name, description, game, format::text, status::text, max_participants, swiss_rounds, registration_open, COALESCE(settings, '{}'), starts_at, tournament_class, prize_pool_usd, logo_url, created_at, updated_at
		FROM tournaments
		WHERE community_id = $1
		  AND id NOT IN (SELECT tournament_id FROM event_tournaments)
		ORDER BY created_at DESC
	`
	return r.queryTournaments(ctx, query, communityID)
}
