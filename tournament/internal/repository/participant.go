package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/braccet/tournament/internal/domain"
	"github.com/lib/pq"
)

var ErrParticipantNotFound = errors.New("participant not found")

type ParticipantRepository interface {
	Create(ctx context.Context, p *domain.Participant) error
	GetByID(ctx context.Context, id uint64) (*domain.Participant, error)
	GetByIDs(ctx context.Context, ids []uint64) ([]*domain.Participant, error)
	GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.Participant, error)
	GetByTournamentAndUser(ctx context.Context, tournamentID, userID uint64) (*domain.Participant, error)
	CountByTournament(ctx context.Context, tournamentID uint64) (int, error)
	GetCommunityMemberIDsByTournaments(ctx context.Context, tournamentIDs []uint64, limit int) (map[uint64][]uint64, error)
	GetCommunityMemberIDsFromTournaments(ctx context.Context, tournamentIDs []uint64) ([]uint64, error)
	UpdateSeeding(ctx context.Context, tournamentID uint64, seeds map[uint64]uint) error
	UpdateStatus(ctx context.Context, id uint64, status domain.ParticipantStatus) error
	UpdateCommunityMemberID(ctx context.Context, id uint64, communityMemberID uint64) error
	Delete(ctx context.Context, id uint64) error
	GetOrphanedCommunityMemberIDs(ctx context.Context, tournamentID uint64, communityID uint64) ([]uint64, error)
	ResetAllStatuses(ctx context.Context, tournamentID uint64, status domain.ParticipantStatus) error
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

func (r *participantRepository) GetByIDs(ctx context.Context, ids []uint64) ([]*domain.Participant, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Convert []uint64 to []int64 for pq.Array (pq doesn't support uint64 directly)
	int64Ids := make([]int64, len(ids))
	for i, id := range ids {
		int64Ids[i] = int64(id)
	}

	query := `
		SELECT id, tournament_id, user_id, community_member_id, display_name, seed, status, checked_in_at, created_at
		FROM participants
		WHERE id = ANY($1)
		ORDER BY seed ASC NULLS LAST, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, pq.Array(int64Ids))
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

func (r *participantRepository) UpdateCommunityMemberID(ctx context.Context, id uint64, communityMemberID uint64) error {
	query := `UPDATE participants SET community_member_id = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, communityMemberID, id)
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

// GetOrphanedCommunityMemberIDs finds community_member_ids from the specified tournament
// that do NOT appear in any other tournament within the same community.
func (r *participantRepository) GetOrphanedCommunityMemberIDs(ctx context.Context, tournamentID uint64, communityID uint64) ([]uint64, error) {
	query := `
		SELECT DISTINCT p1.community_member_id
		FROM participants p1
		JOIN tournaments t1 ON p1.tournament_id = t1.id
		WHERE p1.tournament_id = $1
		  AND p1.community_member_id IS NOT NULL
		  AND NOT EXISTS (
			SELECT 1 FROM participants p2
			JOIN tournaments t2 ON p2.tournament_id = t2.id
			WHERE p2.community_member_id = p1.community_member_id
			  AND t2.community_id = $2
			  AND p2.tournament_id != $1
		  )
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID, communityID)
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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memberIDs, nil
}

// ResetAllStatuses resets the status of all participants in a tournament.
// This is used when resetting a tournament back to registration phase.
func (r *participantRepository) ResetAllStatuses(ctx context.Context, tournamentID uint64, status domain.ParticipantStatus) error {
	query := `UPDATE participants SET status = $1 WHERE tournament_id = $2`
	_, err := r.db.ExecContext(ctx, query, status, tournamentID)
	return err
}

// GetCommunityMemberIDsFromTournaments gets all unique community member IDs from multiple tournaments.
// Returns a flat slice of unique community_member_ids (excluding withdrawn/disqualified).
func (r *participantRepository) GetCommunityMemberIDsFromTournaments(ctx context.Context, tournamentIDs []uint64) ([]uint64, error) {
	if len(tournamentIDs) == 0 {
		return []uint64{}, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(tournamentIDs))
	args := make([]interface{}, len(tournamentIDs))
	for i, id := range tournamentIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT community_member_id
		FROM participants
		WHERE tournament_id IN (%s)
		  AND community_member_id IS NOT NULL
		  AND status NOT IN ('withdrawn', 'disqualified')
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return memberIDs, nil
}

// GetCommunityMemberIDsByTournaments gets community member IDs for multiple tournaments.
// Returns a map of tournament_id -> list of community_member_ids (up to `limit` per tournament, ordered by seed).
func (r *participantRepository) GetCommunityMemberIDsByTournaments(ctx context.Context, tournamentIDs []uint64, limit int) (map[uint64][]uint64, error) {
	if len(tournamentIDs) == 0 {
		return make(map[uint64][]uint64), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(tournamentIDs))
	args := make([]interface{}, len(tournamentIDs)+1)
	for i, id := range tournamentIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	args[len(tournamentIDs)] = limit

	query := fmt.Sprintf(`
		SELECT tournament_id, community_member_id
		FROM (
			SELECT tournament_id, community_member_id,
			       ROW_NUMBER() OVER (PARTITION BY tournament_id ORDER BY seed ASC NULLS LAST, created_at ASC) as rn
			FROM participants
			WHERE tournament_id IN (%s)
			  AND community_member_id IS NOT NULL
			  AND status NOT IN ('withdrawn', 'disqualified')
		) ranked
		WHERE rn <= $%d
		ORDER BY tournament_id, rn
	`, strings.Join(placeholders, ","), len(tournamentIDs)+1)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uint64][]uint64)
	for rows.Next() {
		var tournamentID, memberID uint64
		if err := rows.Scan(&tournamentID, &memberID); err != nil {
			return nil, err
		}
		result[tournamentID] = append(result[tournamentID], memberID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
