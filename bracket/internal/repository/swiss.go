package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/bracket/internal/domain"
)

var (
	ErrSwissConfigNotFound   = errors.New("swiss config not found")
	ErrSwissStandingNotFound = errors.New("swiss standing not found")
)

type SwissRepository interface {
	// Config operations
	CreateConfig(ctx context.Context, config *domain.SwissConfig) error
	GetConfig(ctx context.Context, tournamentID uint64) (*domain.SwissConfig, error)
	UpdateConfig(ctx context.Context, config *domain.SwissConfig) error
	AdvanceRoundAtomic(ctx context.Context, tournamentID uint64, expectedRound, newRound int) (bool, error)

	// Standings operations
	CreateStandings(ctx context.Context, standings []*domain.SwissStanding) error
	GetStandings(ctx context.Context, tournamentID uint64) ([]*domain.SwissStanding, error)
	GetStandingByParticipant(ctx context.Context, tournamentID, participantID uint64) (*domain.SwissStanding, error)
	UpdateStanding(ctx context.Context, standing *domain.SwissStanding) error
	UpdateOpponentWins(ctx context.Context, tournamentID uint64) error

	// Pairing history operations
	CreatePairings(ctx context.Context, pairings []*domain.SwissPairingHistory) error
	GetPairingHistory(ctx context.Context, tournamentID uint64) ([]*domain.SwissPairingHistory, error)
	HavePlayed(ctx context.Context, tournamentID, p1ID, p2ID uint64) (bool, error)
	GetOpponentIDs(ctx context.Context, tournamentID, participantID uint64) ([]uint64, error)

	// Cleanup
	DeleteByTournament(ctx context.Context, tournamentID uint64) error
}

type swissRepository struct {
	db *sql.DB
}

func NewSwissRepository(db *sql.DB) SwissRepository {
	return &swissRepository{db: db}
}

// Config operations

func (r *swissRepository) CreateConfig(ctx context.Context, config *domain.SwissConfig) error {
	query := `
		INSERT INTO swiss_config (tournament_id, total_rounds, current_round, is_complete)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query,
		config.TournamentID, config.TotalRounds, config.CurrentRound, config.IsComplete,
	)
	return err
}

func (r *swissRepository) GetConfig(ctx context.Context, tournamentID uint64) (*domain.SwissConfig, error) {
	query := `
		SELECT tournament_id, total_rounds, current_round, is_complete, created_at, updated_at
		FROM swiss_config
		WHERE tournament_id = $1
	`
	config := &domain.SwissConfig{}
	err := r.db.QueryRowContext(ctx, query, tournamentID).Scan(
		&config.TournamentID, &config.TotalRounds, &config.CurrentRound,
		&config.IsComplete, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSwissConfigNotFound
		}
		return nil, err
	}
	return config, nil
}

func (r *swissRepository) UpdateConfig(ctx context.Context, config *domain.SwissConfig) error {
	query := `
		UPDATE swiss_config
		SET total_rounds = $2, current_round = $3, is_complete = $4
		WHERE tournament_id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		config.TournamentID, config.TotalRounds, config.CurrentRound, config.IsComplete,
	)
	return err
}

// AdvanceRoundAtomic atomically advances the current round if and only if
// the current round matches expectedRound. Returns true if the update succeeded,
// false if the round was already advanced by another process.
func (r *swissRepository) AdvanceRoundAtomic(ctx context.Context, tournamentID uint64, expectedRound, newRound int) (bool, error) {
	query := `
		UPDATE swiss_config
		SET current_round = $3
		WHERE tournament_id = $1 AND current_round = $2
	`
	result, err := r.db.ExecContext(ctx, query, tournamentID, expectedRound, newRound)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

// Standings operations

func (r *swissRepository) CreateStandings(ctx context.Context, standings []*domain.SwissStanding) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO swiss_standings (tournament_id, participant_id, participant_name, participant_icon_url, seed, wins, losses, game_wins, game_losses, opponent_wins, has_bye)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range standings {
		err := stmt.QueryRowContext(ctx,
			s.TournamentID, s.ParticipantID, s.ParticipantName, s.ParticipantIconURL,
			s.Seed, s.Wins, s.Losses, s.GameWins, s.GameLosses, s.OpponentWins, s.HasBye,
		).Scan(&s.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *swissRepository) GetStandings(ctx context.Context, tournamentID uint64) ([]*domain.SwissStanding, error) {
	query := `
		SELECT id, tournament_id, participant_id, participant_name, participant_icon_url,
		       seed, wins, losses, game_wins, game_losses, opponent_wins, has_bye,
		       created_at, updated_at
		FROM swiss_standings
		WHERE tournament_id = $1
		ORDER BY wins DESC, (game_wins - game_losses) DESC, opponent_wins DESC, seed ASC
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var standings []*domain.SwissStanding
	for rows.Next() {
		s := &domain.SwissStanding{}
		err := rows.Scan(
			&s.ID, &s.TournamentID, &s.ParticipantID, &s.ParticipantName, &s.ParticipantIconURL,
			&s.Seed, &s.Wins, &s.Losses, &s.GameWins, &s.GameLosses, &s.OpponentWins, &s.HasBye,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		standings = append(standings, s)
	}

	return standings, rows.Err()
}

func (r *swissRepository) GetStandingByParticipant(ctx context.Context, tournamentID, participantID uint64) (*domain.SwissStanding, error) {
	query := `
		SELECT id, tournament_id, participant_id, participant_name, participant_icon_url,
		       seed, wins, losses, game_wins, game_losses, opponent_wins, has_bye,
		       created_at, updated_at
		FROM swiss_standings
		WHERE tournament_id = $1 AND participant_id = $2
	`
	s := &domain.SwissStanding{}
	err := r.db.QueryRowContext(ctx, query, tournamentID, participantID).Scan(
		&s.ID, &s.TournamentID, &s.ParticipantID, &s.ParticipantName, &s.ParticipantIconURL,
		&s.Seed, &s.Wins, &s.Losses, &s.GameWins, &s.GameLosses, &s.OpponentWins, &s.HasBye,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSwissStandingNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *swissRepository) UpdateStanding(ctx context.Context, standing *domain.SwissStanding) error {
	query := `
		UPDATE swiss_standings
		SET wins = $3, losses = $4, game_wins = $5, game_losses = $6, opponent_wins = $7, has_bye = $8
		WHERE tournament_id = $1 AND participant_id = $2
	`
	_, err := r.db.ExecContext(ctx, query,
		standing.TournamentID, standing.ParticipantID,
		standing.Wins, standing.Losses, standing.GameWins, standing.GameLosses,
		standing.OpponentWins, standing.HasBye,
	)
	return err
}

// UpdateOpponentWins recalculates the Buchholz tiebreaker for all participants
func (r *swissRepository) UpdateOpponentWins(ctx context.Context, tournamentID uint64) error {
	// This query updates each participant's opponent_wins by summing the wins
	// of all their previous opponents
	query := `
		UPDATE swiss_standings s
		SET opponent_wins = COALESCE((
			SELECT SUM(opp.wins)
			FROM swiss_pairing_history ph
			JOIN swiss_standings opp ON (
				(ph.participant1_id = s.participant_id AND ph.participant2_id = opp.participant_id) OR
				(ph.participant2_id = s.participant_id AND ph.participant1_id = opp.participant_id)
			)
			WHERE ph.tournament_id = $1 AND opp.tournament_id = $1
		), 0)
		WHERE s.tournament_id = $1
	`
	_, err := r.db.ExecContext(ctx, query, tournamentID)
	return err
}

// Pairing history operations

func (r *swissRepository) CreatePairings(ctx context.Context, pairings []*domain.SwissPairingHistory) error {
	if len(pairings) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO swiss_pairing_history (tournament_id, participant1_id, participant2_id, round)
		VALUES ($1, $2, $3, $4)
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range pairings {
		// Store with smaller ID first for consistency
		p1, p2 := p.Participant1ID, p.Participant2ID
		if p1 > p2 {
			p1, p2 = p2, p1
		}
		_, err := stmt.ExecContext(ctx, p.TournamentID, p1, p2, p.Round)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *swissRepository) GetPairingHistory(ctx context.Context, tournamentID uint64) ([]*domain.SwissPairingHistory, error) {
	query := `
		SELECT id, tournament_id, participant1_id, participant2_id, round, created_at
		FROM swiss_pairing_history
		WHERE tournament_id = $1
		ORDER BY round, id
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*domain.SwissPairingHistory
	for rows.Next() {
		p := &domain.SwissPairingHistory{}
		err := rows.Scan(&p.ID, &p.TournamentID, &p.Participant1ID, &p.Participant2ID, &p.Round, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		history = append(history, p)
	}

	return history, rows.Err()
}

func (r *swissRepository) HavePlayed(ctx context.Context, tournamentID, p1ID, p2ID uint64) (bool, error) {
	// Normalize order
	if p1ID > p2ID {
		p1ID, p2ID = p2ID, p1ID
	}

	query := `
		SELECT EXISTS(
			SELECT 1 FROM swiss_pairing_history
			WHERE tournament_id = $1 AND participant1_id = $2 AND participant2_id = $3
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, tournamentID, p1ID, p2ID).Scan(&exists)
	return exists, err
}

func (r *swissRepository) GetOpponentIDs(ctx context.Context, tournamentID, participantID uint64) ([]uint64, error) {
	query := `
		SELECT CASE
			WHEN participant1_id = $2 THEN participant2_id
			ELSE participant1_id
		END as opponent_id
		FROM swiss_pairing_history
		WHERE tournament_id = $1 AND (participant1_id = $2 OR participant2_id = $2)
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opponents []uint64
	for rows.Next() {
		var oppID uint64
		if err := rows.Scan(&oppID); err != nil {
			return nil, err
		}
		opponents = append(opponents, oppID)
	}

	return opponents, rows.Err()
}

// Cleanup

func (r *swissRepository) DeleteByTournament(ctx context.Context, tournamentID uint64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete in order: pairing history, standings, config
	queries := []string{
		`DELETE FROM swiss_pairing_history WHERE tournament_id = $1`,
		`DELETE FROM swiss_standings WHERE tournament_id = $1`,
		`DELETE FROM swiss_config WHERE tournament_id = $1`,
	}

	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q, tournamentID); err != nil {
			return err
		}
	}

	return tx.Commit()
}
