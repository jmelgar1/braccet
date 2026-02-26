package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/tournament/internal/domain"
)

var ErrEventNotFound = errors.New("event not found")
var ErrEventTournamentNotFound = errors.New("event tournament not found")

type EventRepository interface {
	// Event CRUD
	Create(ctx context.Context, e *domain.Event) error
	GetByID(ctx context.Context, id uint64) (*domain.Event, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Event, error)
	Update(ctx context.Context, e *domain.Event) error
	Delete(ctx context.Context, slug string) error
	ListByCommunity(ctx context.Context, communityID uint64) ([]*domain.Event, error)
	ListByOrganizer(ctx context.Context, organizerID uint64) ([]*domain.Event, error)

	// Event Tournaments
	AddTournament(ctx context.Context, et *domain.EventTournament) error
	GetTournamentsByEvent(ctx context.Context, eventID uint64) ([]*domain.EventTournament, error)
	GetEventTournamentByTournamentID(ctx context.Context, tournamentID uint64) (*domain.EventTournament, error)
	UpdateTournament(ctx context.Context, et *domain.EventTournament) error
	UpdateTournamentByTournamentID(ctx context.Context, et *domain.EventTournament) error
	RemoveTournament(ctx context.Context, eventID, tournamentID uint64) error

	// Advancement
	CreateAdvancement(ctx context.Context, adv *domain.EventAdvancement) error
	GetAdvancementsBySource(ctx context.Context, sourceTournamentID uint64) ([]*domain.EventAdvancement, error)
	GetAdvancementsByTarget(ctx context.Context, targetTournamentID uint64) ([]*domain.EventAdvancement, error)
	DeleteAdvancementsBySource(ctx context.Context, sourceTournamentID uint64) error
	DeleteAdvancementByTargetParticipant(ctx context.Context, targetTournamentID, targetParticipantID uint64) error

	// DAG traversal
	GetDownstreamTournamentIDs(ctx context.Context, tournamentID uint64) ([]uint64, error)
}

type eventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) EventRepository {
	return &eventRepository{db: db}
}

// Event CRUD

func (r *eventRepository) Create(ctx context.Context, e *domain.Event) error {
	query := `
		INSERT INTO events (slug, community_id, organizer_id, name, description, game, status, starts_at, logo_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		e.Slug, e.CommunityID, e.OrganizerID, e.Name, e.Description, e.Game, e.Status, e.StartsAt, e.LogoURL,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	return err
}

func (r *eventRepository) GetByID(ctx context.Context, id uint64) (*domain.Event, error) {
	query := `
		SELECT id, slug, community_id, organizer_id, name, description, game, status::text, starts_at, logo_url, created_at, updated_at
		FROM events
		WHERE id = $1
	`
	e := &domain.Event{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&e.ID, &e.Slug, &e.CommunityID, &e.OrganizerID, &e.Name, &e.Description, &e.Game, &e.Status, &e.StartsAt, &e.LogoURL, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	return e, nil
}

func (r *eventRepository) GetBySlug(ctx context.Context, slug string) (*domain.Event, error) {
	query := `
		SELECT id, slug, community_id, organizer_id, name, description, game, status::text, starts_at, logo_url, created_at, updated_at
		FROM events
		WHERE LOWER(slug) = LOWER($1)
	`
	e := &domain.Event{}
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&e.ID, &e.Slug, &e.CommunityID, &e.OrganizerID, &e.Name, &e.Description, &e.Game, &e.Status, &e.StartsAt, &e.LogoURL, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	return e, nil
}

func (r *eventRepository) Update(ctx context.Context, e *domain.Event) error {
	query := `
		UPDATE events
		SET name = $1, description = $2, game = $3, status = $4, starts_at = $5, logo_url = $6
		WHERE id = $7
	`
	result, err := r.db.ExecContext(ctx, query,
		e.Name, e.Description, e.Game, e.Status, e.StartsAt, e.LogoURL, e.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEventNotFound
	}
	return nil
}

func (r *eventRepository) Delete(ctx context.Context, slug string) error {
	query := `DELETE FROM events WHERE LOWER(slug) = LOWER($1)`
	result, err := r.db.ExecContext(ctx, query, slug)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEventNotFound
	}
	return nil
}

func (r *eventRepository) ListByCommunity(ctx context.Context, communityID uint64) ([]*domain.Event, error) {
	query := `
		SELECT id, slug, community_id, organizer_id, name, description, game, status::text, starts_at, logo_url, created_at, updated_at
		FROM events
		WHERE community_id = $1
		ORDER BY created_at DESC
	`
	return r.queryEvents(ctx, query, communityID)
}

func (r *eventRepository) ListByOrganizer(ctx context.Context, organizerID uint64) ([]*domain.Event, error) {
	query := `
		SELECT id, slug, community_id, organizer_id, name, description, game, status::text, starts_at, logo_url, created_at, updated_at
		FROM events
		WHERE organizer_id = $1
		ORDER BY created_at DESC
	`
	return r.queryEvents(ctx, query, organizerID)
}

func (r *eventRepository) queryEvents(ctx context.Context, query string, args ...any) ([]*domain.Event, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		e := &domain.Event{}
		err := rows.Scan(
			&e.ID, &e.Slug, &e.CommunityID, &e.OrganizerID, &e.Name, &e.Description, &e.Game, &e.Status, &e.StartsAt, &e.LogoURL, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

// Event Tournaments

func (r *eventRepository) AddTournament(ctx context.Context, et *domain.EventTournament) error {
	query := `
		INSERT INTO event_tournaments (event_id, tournament_id, role, display_order, advancing_count, target_tournament_id, position_x, position_y)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query,
		et.EventID, et.TournamentID, et.Role, et.DisplayOrder, et.AdvancingCount, et.TargetTournamentID, et.PositionX, et.PositionY,
	).Scan(&et.ID, &et.CreatedAt)
	return err
}

func (r *eventRepository) GetTournamentsByEvent(ctx context.Context, eventID uint64) ([]*domain.EventTournament, error) {
	query := `
		SELECT id, event_id, tournament_id, role::text, display_order, advancing_count, target_tournament_id, position_x, position_y, created_at
		FROM event_tournaments
		WHERE event_id = $1
		ORDER BY display_order ASC
	`
	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventTournaments []*domain.EventTournament
	for rows.Next() {
		et := &domain.EventTournament{}
		err := rows.Scan(
			&et.ID, &et.EventID, &et.TournamentID, &et.Role, &et.DisplayOrder, &et.AdvancingCount, &et.TargetTournamentID, &et.PositionX, &et.PositionY, &et.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		eventTournaments = append(eventTournaments, et)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return eventTournaments, nil
}

func (r *eventRepository) GetEventTournamentByTournamentID(ctx context.Context, tournamentID uint64) (*domain.EventTournament, error) {
	query := `
		SELECT id, event_id, tournament_id, role::text, display_order, advancing_count, target_tournament_id, position_x, position_y, created_at
		FROM event_tournaments
		WHERE tournament_id = $1
	`
	et := &domain.EventTournament{}
	err := r.db.QueryRowContext(ctx, query, tournamentID).Scan(
		&et.ID, &et.EventID, &et.TournamentID, &et.Role, &et.DisplayOrder, &et.AdvancingCount, &et.TargetTournamentID, &et.PositionX, &et.PositionY, &et.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEventTournamentNotFound
		}
		return nil, err
	}
	return et, nil
}

func (r *eventRepository) UpdateTournament(ctx context.Context, et *domain.EventTournament) error {
	query := `
		UPDATE event_tournaments
		SET role = $1, display_order = $2, advancing_count = $3, target_tournament_id = $4, position_x = $5, position_y = $6
		WHERE id = $7
	`
	result, err := r.db.ExecContext(ctx, query,
		et.Role, et.DisplayOrder, et.AdvancingCount, et.TargetTournamentID, et.PositionX, et.PositionY, et.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEventTournamentNotFound
	}
	return nil
}

func (r *eventRepository) RemoveTournament(ctx context.Context, eventID, tournamentID uint64) error {
	query := `DELETE FROM event_tournaments WHERE event_id = $1 AND tournament_id = $2`
	result, err := r.db.ExecContext(ctx, query, eventID, tournamentID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEventTournamentNotFound
	}
	return nil
}

func (r *eventRepository) UpdateTournamentByTournamentID(ctx context.Context, et *domain.EventTournament) error {
	query := `
		UPDATE event_tournaments
		SET role = $1, display_order = $2, advancing_count = $3, target_tournament_id = $4, position_x = $5, position_y = $6
		WHERE tournament_id = $7
	`
	result, err := r.db.ExecContext(ctx, query,
		et.Role, et.DisplayOrder, et.AdvancingCount, et.TargetTournamentID, et.PositionX, et.PositionY, et.TournamentID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEventTournamentNotFound
	}
	return nil
}

// Advancement

func (r *eventRepository) CreateAdvancement(ctx context.Context, adv *domain.EventAdvancement) error {
	query := `
		INSERT INTO event_advancement (event_id, source_tournament_id, source_participant_id, target_tournament_id, target_participant_id, final_placement)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, advanced_at
	`
	err := r.db.QueryRowContext(ctx, query,
		adv.EventID, adv.SourceTournamentID, adv.SourceParticipantID, adv.TargetTournamentID, adv.TargetParticipantID, adv.FinalPlacement,
	).Scan(&adv.ID, &adv.AdvancedAt)
	return err
}

func (r *eventRepository) GetAdvancementsBySource(ctx context.Context, sourceTournamentID uint64) ([]*domain.EventAdvancement, error) {
	query := `
		SELECT id, event_id, source_tournament_id, source_participant_id, target_tournament_id, target_participant_id, final_placement, advanced_at
		FROM event_advancement
		WHERE source_tournament_id = $1
		ORDER BY final_placement ASC
	`
	return r.queryAdvancements(ctx, query, sourceTournamentID)
}

func (r *eventRepository) GetAdvancementsByTarget(ctx context.Context, targetTournamentID uint64) ([]*domain.EventAdvancement, error) {
	query := `
		SELECT id, event_id, source_tournament_id, source_participant_id, target_tournament_id, target_participant_id, final_placement, advanced_at
		FROM event_advancement
		WHERE target_tournament_id = $1
		ORDER BY final_placement ASC
	`
	return r.queryAdvancements(ctx, query, targetTournamentID)
}

func (r *eventRepository) queryAdvancements(ctx context.Context, query string, args ...any) ([]*domain.EventAdvancement, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var advancements []*domain.EventAdvancement
	for rows.Next() {
		adv := &domain.EventAdvancement{}
		err := rows.Scan(
			&adv.ID, &adv.EventID, &adv.SourceTournamentID, &adv.SourceParticipantID, &adv.TargetTournamentID, &adv.TargetParticipantID, &adv.FinalPlacement, &adv.AdvancedAt,
		)
		if err != nil {
			return nil, err
		}
		advancements = append(advancements, adv)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return advancements, nil
}

func (r *eventRepository) DeleteAdvancementsBySource(ctx context.Context, sourceTournamentID uint64) error {
	query := `DELETE FROM event_advancement WHERE source_tournament_id = $1`
	_, err := r.db.ExecContext(ctx, query, sourceTournamentID)
	return err
}

func (r *eventRepository) DeleteAdvancementByTargetParticipant(ctx context.Context, targetTournamentID, targetParticipantID uint64) error {
	query := `DELETE FROM event_advancement WHERE target_tournament_id = $1 AND target_participant_id = $2`
	_, err := r.db.ExecContext(ctx, query, targetTournamentID, targetParticipantID)
	return err
}

// GetDownstreamTournamentIDs returns all tournament IDs that are "downstream" (later) from the given tournament
// in the event DAG. This traverses the target_tournament_id chain to find all reachable tournaments.
// Returns empty slice if tournament is not part of an event or has no downstream tournaments.
func (r *eventRepository) GetDownstreamTournamentIDs(ctx context.Context, tournamentID uint64) ([]uint64, error) {
	// First, get the event tournament record for this tournament
	et, err := r.GetEventTournamentByTournamentID(ctx, tournamentID)
	if err != nil {
		if errors.Is(err, ErrEventTournamentNotFound) {
			// Tournament is not part of an event
			return []uint64{}, nil
		}
		return nil, err
	}

	// Get all event tournaments for this event
	allEventTournaments, err := r.GetTournamentsByEvent(ctx, et.EventID)
	if err != nil {
		return nil, err
	}

	// Build adjacency map: tournament_id -> target_tournament_id
	adjacency := make(map[uint64]uint64)
	for _, t := range allEventTournaments {
		if t.TargetTournamentID != nil {
			adjacency[t.TournamentID] = *t.TargetTournamentID
		}
	}

	// BFS to find all reachable tournament IDs from the current tournament
	visited := make(map[uint64]bool)
	var downstream []uint64

	// Start from the current tournament's target (if any)
	current := tournamentID
	for {
		target, hasTarget := adjacency[current]
		if !hasTarget {
			break
		}
		if visited[target] {
			// Cycle detection (shouldn't happen with proper DAG, but be safe)
			break
		}
		visited[target] = true
		downstream = append(downstream, target)
		current = target
	}

	return downstream, nil
}
