package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/bracket/internal/domain"
)

var ErrGroupStandingNotFound = errors.New("group standing not found")

// GroupRepository handles database operations for group brackets
type GroupRepository interface {
	// Group standings
	CreateGroupStandings(ctx context.Context, standings []*domain.GroupStanding) error
	GetGroupStandings(ctx context.Context, groupID uint64) ([]*domain.GroupStanding, error)
	GetGroupStandingByParticipant(ctx context.Context, groupID, participantID uint64) (*domain.GroupStanding, error)
	UpdateGroupStanding(ctx context.Context, standing *domain.GroupStanding) error
	DeleteGroupStandingsByStage(ctx context.Context, stageID uint64) error
	DeleteGroupStandingsByGroup(ctx context.Context, groupID uint64) error
	DeleteGroupStandingsByTournament(ctx context.Context, tournamentID uint64) error

	// Stage standings
	CreateStageStandings(ctx context.Context, standings []*domain.StageStanding) error
	GetStageStandings(ctx context.Context, stageID uint64) ([]*domain.StageStanding, error)
	GetAdvancingParticipants(ctx context.Context, stageID uint64) ([]*domain.StageStanding, error)
	GetAdvancingParticipantsByTournament(ctx context.Context, tournamentID uint64) ([]*domain.StageStanding, error)
	UpdateStageStanding(ctx context.Context, standing *domain.StageStanding) error
	DeleteStageStandingsByStage(ctx context.Context, stageID uint64) error
	DeleteStageStandingsByTournament(ctx context.Context, tournamentID uint64) error

	// Group-scoped match queries
	GetMatchesByGroup(ctx context.Context, tournamentID, groupID uint64) ([]*domain.Match, error)
	GetMatchesByStage(ctx context.Context, tournamentID, stageID uint64) ([]*domain.Match, error)
}

type groupRepository struct {
	db *sql.DB
}

func NewGroupRepository(db *sql.DB) GroupRepository {
	return &groupRepository{db: db}
}

// Group standings

func (r *groupRepository) CreateGroupStandings(ctx context.Context, standings []*domain.GroupStanding) error {
	if len(standings) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO group_standings (
			tournament_id, stage_id, group_id, participant_id, participant_name, participant_icon_url, seed,
			match_wins, match_losses, set_wins, set_losses, points_scored, points_against, opponent_match_wins, group_rank
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range standings {
		if err := stmt.QueryRowContext(ctx,
			s.TournamentID, s.StageID, s.GroupID, s.ParticipantID, s.ParticipantName, s.ParticipantIconURL, s.Seed,
			s.MatchWins, s.MatchLosses, s.SetWins, s.SetLosses, s.PointsScored, s.PointsAgainst, s.OpponentMatchWins, s.GroupRank,
		).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *groupRepository) GetGroupStandings(ctx context.Context, groupID uint64) ([]*domain.GroupStanding, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_id, participant_id, participant_name, participant_icon_url, seed,
		       match_wins, match_losses, set_wins, set_losses, points_scored, points_against, opponent_match_wins, group_rank,
		       created_at, updated_at
		FROM group_standings
		WHERE group_id = $1
		ORDER BY COALESCE(group_rank, 999) ASC, match_wins DESC, (set_wins - set_losses) DESC
	`
	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var standings []*domain.GroupStanding
	for rows.Next() {
		s := &domain.GroupStanding{}
		if err := rows.Scan(
			&s.ID, &s.TournamentID, &s.StageID, &s.GroupID, &s.ParticipantID, &s.ParticipantName, &s.ParticipantIconURL, &s.Seed,
			&s.MatchWins, &s.MatchLosses, &s.SetWins, &s.SetLosses, &s.PointsScored, &s.PointsAgainst, &s.OpponentMatchWins, &s.GroupRank,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		standings = append(standings, s)
	}

	return standings, rows.Err()
}

func (r *groupRepository) GetGroupStandingByParticipant(ctx context.Context, groupID, participantID uint64) (*domain.GroupStanding, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_id, participant_id, participant_name, participant_icon_url, seed,
		       match_wins, match_losses, set_wins, set_losses, points_scored, points_against, opponent_match_wins, group_rank,
		       created_at, updated_at
		FROM group_standings
		WHERE group_id = $1 AND participant_id = $2
	`
	s := &domain.GroupStanding{}
	err := r.db.QueryRowContext(ctx, query, groupID, participantID).Scan(
		&s.ID, &s.TournamentID, &s.StageID, &s.GroupID, &s.ParticipantID, &s.ParticipantName, &s.ParticipantIconURL, &s.Seed,
		&s.MatchWins, &s.MatchLosses, &s.SetWins, &s.SetLosses, &s.PointsScored, &s.PointsAgainst, &s.OpponentMatchWins, &s.GroupRank,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGroupStandingNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *groupRepository) UpdateGroupStanding(ctx context.Context, standing *domain.GroupStanding) error {
	query := `
		UPDATE group_standings
		SET match_wins = $1, match_losses = $2, set_wins = $3, set_losses = $4,
		    points_scored = $5, points_against = $6, opponent_match_wins = $7, group_rank = $8
		WHERE id = $9
	`
	result, err := r.db.ExecContext(ctx, query,
		standing.MatchWins, standing.MatchLosses, standing.SetWins, standing.SetLosses,
		standing.PointsScored, standing.PointsAgainst, standing.OpponentMatchWins, standing.GroupRank,
		standing.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrGroupStandingNotFound
	}
	return nil
}

func (r *groupRepository) DeleteGroupStandingsByStage(ctx context.Context, stageID uint64) error {
	query := `DELETE FROM group_standings WHERE stage_id = $1`
	_, err := r.db.ExecContext(ctx, query, stageID)
	return err
}

func (r *groupRepository) DeleteGroupStandingsByGroup(ctx context.Context, groupID uint64) error {
	query := `DELETE FROM group_standings WHERE group_id = $1`
	_, err := r.db.ExecContext(ctx, query, groupID)
	return err
}

func (r *groupRepository) DeleteGroupStandingsByTournament(ctx context.Context, tournamentID uint64) error {
	query := `DELETE FROM group_standings WHERE tournament_id = $1`
	_, err := r.db.ExecContext(ctx, query, tournamentID)
	return err
}

// Stage standings

func (r *groupRepository) CreateStageStandings(ctx context.Context, standings []*domain.StageStanding) error {
	if len(standings) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO stage_standings (
			tournament_id, stage_id, participant_id, group_id, group_rank,
			match_wins, match_losses, set_wins, set_losses, points_scored, points_against, stage_rank, advances
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range standings {
		if err := stmt.QueryRowContext(ctx,
			s.TournamentID, s.StageID, s.ParticipantID, s.GroupID, s.GroupRank,
			s.MatchWins, s.MatchLosses, s.SetWins, s.SetLosses, s.PointsScored, s.PointsAgainst, s.StageRank, s.Advances,
		).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *groupRepository) GetStageStandings(ctx context.Context, stageID uint64) ([]*domain.StageStanding, error) {
	query := `
		SELECT id, tournament_id, stage_id, participant_id, group_id, group_rank,
		       match_wins, match_losses, set_wins, set_losses, points_scored, points_against, stage_rank, advances,
		       created_at, updated_at
		FROM stage_standings
		WHERE stage_id = $1
		ORDER BY COALESCE(stage_rank, 999) ASC, group_rank ASC, match_wins DESC
	`
	rows, err := r.db.QueryContext(ctx, query, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var standings []*domain.StageStanding
	for rows.Next() {
		s := &domain.StageStanding{}
		if err := rows.Scan(
			&s.ID, &s.TournamentID, &s.StageID, &s.ParticipantID, &s.GroupID, &s.GroupRank,
			&s.MatchWins, &s.MatchLosses, &s.SetWins, &s.SetLosses, &s.PointsScored, &s.PointsAgainst, &s.StageRank, &s.Advances,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		standings = append(standings, s)
	}

	return standings, rows.Err()
}

func (r *groupRepository) GetAdvancingParticipants(ctx context.Context, stageID uint64) ([]*domain.StageStanding, error) {
	query := `
		SELECT id, tournament_id, stage_id, participant_id, group_id, group_rank,
		       match_wins, match_losses, set_wins, set_losses, points_scored, points_against, stage_rank, advances,
		       created_at, updated_at
		FROM stage_standings
		WHERE stage_id = $1 AND advances = true
		ORDER BY COALESCE(stage_rank, 999) ASC
	`
	rows, err := r.db.QueryContext(ctx, query, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var standings []*domain.StageStanding
	for rows.Next() {
		s := &domain.StageStanding{}
		if err := rows.Scan(
			&s.ID, &s.TournamentID, &s.StageID, &s.ParticipantID, &s.GroupID, &s.GroupRank,
			&s.MatchWins, &s.MatchLosses, &s.SetWins, &s.SetLosses, &s.PointsScored, &s.PointsAgainst, &s.StageRank, &s.Advances,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		standings = append(standings, s)
	}

	return standings, rows.Err()
}

// GetAdvancingParticipantsByTournament returns all advancing participants for a tournament.
// This is useful for reseeding finals brackets where we need participants that advanced from the previous stage.
func (r *groupRepository) GetAdvancingParticipantsByTournament(ctx context.Context, tournamentID uint64) ([]*domain.StageStanding, error) {
	query := `
		SELECT id, tournament_id, stage_id, participant_id, group_id, group_rank,
		       match_wins, match_losses, set_wins, set_losses, points_scored, points_against, stage_rank, advances,
		       created_at, updated_at
		FROM stage_standings
		WHERE tournament_id = $1 AND advances = true
		ORDER BY COALESCE(stage_rank, 999) ASC
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var standings []*domain.StageStanding
	for rows.Next() {
		s := &domain.StageStanding{}
		if err := rows.Scan(
			&s.ID, &s.TournamentID, &s.StageID, &s.ParticipantID, &s.GroupID, &s.GroupRank,
			&s.MatchWins, &s.MatchLosses, &s.SetWins, &s.SetLosses, &s.PointsScored, &s.PointsAgainst, &s.StageRank, &s.Advances,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		standings = append(standings, s)
	}

	return standings, rows.Err()
}

func (r *groupRepository) UpdateStageStanding(ctx context.Context, standing *domain.StageStanding) error {
	query := `
		UPDATE stage_standings
		SET match_wins = $1, match_losses = $2, set_wins = $3, set_losses = $4,
		    points_scored = $5, points_against = $6, stage_rank = $7, advances = $8
		WHERE id = $9
	`
	_, err := r.db.ExecContext(ctx, query,
		standing.MatchWins, standing.MatchLosses, standing.SetWins, standing.SetLosses,
		standing.PointsScored, standing.PointsAgainst, standing.StageRank, standing.Advances,
		standing.ID,
	)
	return err
}

func (r *groupRepository) DeleteStageStandingsByStage(ctx context.Context, stageID uint64) error {
	query := `DELETE FROM stage_standings WHERE stage_id = $1`
	_, err := r.db.ExecContext(ctx, query, stageID)
	return err
}

func (r *groupRepository) DeleteStageStandingsByTournament(ctx context.Context, tournamentID uint64) error {
	query := `DELETE FROM stage_standings WHERE tournament_id = $1`
	_, err := r.db.ExecContext(ctx, query, tournamentID)
	return err
}

// Group-scoped match queries

func (r *groupRepository) GetMatchesByGroup(ctx context.Context, tournamentID, groupID uint64) ([]*domain.Match, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_id, bracket_type, round, position,
		       participant1_id, participant2_id, participant1_name, participant2_name,
		       participant1_icon_url, participant2_icon_url,
		       seed1, seed2, winner_id, status, scheduled_at, completed_at, next_match_id, loser_match_id,
		       forfeit_winner_id, venue_override, created_at, updated_at
		FROM matches
		WHERE tournament_id = $1 AND group_id = $2
		ORDER BY bracket_type, round, position
	`
	return r.queryMatches(ctx, query, tournamentID, groupID)
}

func (r *groupRepository) GetMatchesByStage(ctx context.Context, tournamentID, stageID uint64) ([]*domain.Match, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_id, bracket_type, round, position,
		       participant1_id, participant2_id, participant1_name, participant2_name,
		       participant1_icon_url, participant2_icon_url,
		       seed1, seed2, winner_id, status, scheduled_at, completed_at, next_match_id, loser_match_id,
		       forfeit_winner_id, venue_override, created_at, updated_at
		FROM matches
		WHERE tournament_id = $1 AND stage_id = $2
		ORDER BY group_id, bracket_type, round, position
	`
	return r.queryMatches(ctx, query, tournamentID, stageID)
}

func (r *groupRepository) queryMatches(ctx context.Context, query string, args ...any) ([]*domain.Match, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*domain.Match
	for rows.Next() {
		m := &domain.Match{}
		if err := rows.Scan(
			&m.ID, &m.TournamentID, &m.StageID, &m.GroupID, &m.BracketType, &m.Round, &m.Position,
			&m.Participant1ID, &m.Participant2ID, &m.Participant1Name, &m.Participant2Name,
			&m.Participant1IconURL, &m.Participant2IconURL,
			&m.Seed1, &m.Seed2, &m.WinnerID, &m.Status,
			&m.ScheduledAt, &m.CompletedAt, &m.NextMatchID, &m.LoserMatchID,
			&m.ForfeitWinnerID, &m.VenueOverride, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}

	return matches, rows.Err()
}
