package repository

import (
	"context"
	"database/sql"

	"github.com/braccet/bracket/internal/domain"
)

type StageRepository interface {
	GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.BracketStage, error)
	GetByTournamentStageGroup(ctx context.Context, tournamentID, stageID, groupID uint64) ([]*domain.BracketStage, error)
	GetByRound(ctx context.Context, tournamentID uint64, bracketType domain.BracketType, round int) (*domain.BracketStage, error)
	GetByRoundInGroup(ctx context.Context, tournamentID, stageID, groupID uint64, bracketType domain.BracketType, round int) (*domain.BracketStage, error)
	Upsert(ctx context.Context, stage *domain.BracketStage) error
	UpsertForGroup(ctx context.Context, stage *domain.BracketStage) error
	CreateDefaultStages(ctx context.Context, tournamentID uint64, bracketType domain.BracketType, totalRounds int, isDoubleElim bool) error
	CreateDefaultStagesForGroup(ctx context.Context, tournamentID, stageID, groupID uint64, bracketType domain.BracketType, totalRounds int) error
	DeleteByTournament(ctx context.Context, tournamentID uint64) error
}

type stageRepository struct {
	db *sql.DB
}

func NewStageRepository(db *sql.DB) StageRepository {
	return &stageRepository{db: db}
}

func (r *stageRepository) GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.BracketStage, error) {
	query := `
		SELECT id, tournament_id, bracket_type, round, stage_name, best_of, created_at, updated_at
		FROM bracket_stages
		WHERE tournament_id = $1
		ORDER BY bracket_type, round
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []*domain.BracketStage
	for rows.Next() {
		s := &domain.BracketStage{}
		err := rows.Scan(
			&s.ID, &s.TournamentID, &s.BracketType, &s.Round,
			&s.StageName, &s.BestOf, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		stages = append(stages, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stages, nil
}

func (r *stageRepository) GetByRound(ctx context.Context, tournamentID uint64, bracketType domain.BracketType, round int) (*domain.BracketStage, error) {
	query := `
		SELECT id, tournament_id, bracket_type, round, stage_name, best_of, created_at, updated_at
		FROM bracket_stages
		WHERE tournament_id = $1 AND bracket_type = $2 AND round = $3
	`
	s := &domain.BracketStage{}
	err := r.db.QueryRowContext(ctx, query, tournamentID, bracketType, round).Scan(
		&s.ID, &s.TournamentID, &s.BracketType, &s.Round,
		&s.StageName, &s.BestOf, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *stageRepository) Upsert(ctx context.Context, stage *domain.BracketStage) error {
	query := `
		INSERT INTO bracket_stages (tournament_id, bracket_type, round, stage_name, best_of)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tournament_id, bracket_type, round)
		DO UPDATE SET stage_name = $4, best_of = $5, updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		stage.TournamentID, stage.BracketType, stage.Round,
		stage.StageName, stage.BestOf,
	).Scan(&stage.ID, &stage.CreatedAt, &stage.UpdatedAt)
}

func (r *stageRepository) CreateDefaultStages(ctx context.Context, tournamentID uint64, bracketType domain.BracketType, totalRounds int, isDoubleElim bool) error {
	query := `
		INSERT INTO bracket_stages (tournament_id, bracket_type, round, stage_name, best_of)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tournament_id, bracket_type, round) DO NOTHING
	`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for round := 1; round <= totalRounds; round++ {
		// For single elimination, use empty bracket type to get simple names (Final, Semifinals, etc.)
		// For double elimination, use the actual bracket type to get prefixed names (Winners Final, Losers Final, etc.)
		namingType := bracketType
		if !isDoubleElim {
			namingType = ""
		}
		defaultName := domain.DefaultStageName(namingType, round, totalRounds)
		_, err := stmt.ExecContext(ctx, tournamentID, bracketType, round, defaultName, 1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *stageRepository) DeleteByTournament(ctx context.Context, tournamentID uint64) error {
	query := `DELETE FROM bracket_stages WHERE tournament_id = $1`
	_, err := r.db.ExecContext(ctx, query, tournamentID)
	return err
}

func (r *stageRepository) GetByTournamentStageGroup(ctx context.Context, tournamentID, stageID, groupID uint64) ([]*domain.BracketStage, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_id, bracket_type, round, stage_name, best_of, created_at, updated_at
		FROM bracket_stages
		WHERE tournament_id = $1 AND stage_id = $2 AND group_id = $3
		ORDER BY bracket_type, round
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID, stageID, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []*domain.BracketStage
	for rows.Next() {
		s := &domain.BracketStage{}
		err := rows.Scan(
			&s.ID, &s.TournamentID, &s.StageID, &s.GroupID, &s.BracketType, &s.Round,
			&s.StageName, &s.BestOf, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		stages = append(stages, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stages, nil
}

func (r *stageRepository) GetByRoundInGroup(ctx context.Context, tournamentID, stageID, groupID uint64, bracketType domain.BracketType, round int) (*domain.BracketStage, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_id, bracket_type, round, stage_name, best_of, created_at, updated_at
		FROM bracket_stages
		WHERE tournament_id = $1 AND stage_id = $2 AND group_id = $3 AND bracket_type = $4 AND round = $5
	`
	s := &domain.BracketStage{}
	err := r.db.QueryRowContext(ctx, query, tournamentID, stageID, groupID, bracketType, round).Scan(
		&s.ID, &s.TournamentID, &s.StageID, &s.GroupID, &s.BracketType, &s.Round,
		&s.StageName, &s.BestOf, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *stageRepository) UpsertForGroup(ctx context.Context, stage *domain.BracketStage) error {
	query := `
		INSERT INTO bracket_stages (tournament_id, stage_id, group_id, bracket_type, round, stage_name, best_of)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tournament_id, COALESCE(stage_id, 0), COALESCE(group_id, 0), bracket_type, round)
		DO UPDATE SET stage_name = $6, best_of = $7, updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		stage.TournamentID, stage.StageID, stage.GroupID, stage.BracketType, stage.Round,
		stage.StageName, stage.BestOf,
	).Scan(&stage.ID, &stage.CreatedAt, &stage.UpdatedAt)
}

func (r *stageRepository) CreateDefaultStagesForGroup(ctx context.Context, tournamentID, stageID, groupID uint64, bracketType domain.BracketType, totalRounds int) error {
	query := `
		INSERT INTO bracket_stages (tournament_id, stage_id, group_id, bracket_type, round, stage_name, best_of)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tournament_id, COALESCE(stage_id, 0), COALESCE(group_id, 0), bracket_type, round) DO NOTHING
	`
	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for round := 1; round <= totalRounds; round++ {
		// For Swiss brackets in groups, use simple round names
		defaultName := domain.DefaultSwissStageName(round, totalRounds)
		_, err := stmt.ExecContext(ctx, tournamentID, stageID, groupID, bracketType, round, defaultName, 1)
		if err != nil {
			return err
		}
	}

	return nil
}
