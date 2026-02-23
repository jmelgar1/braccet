package repository

import (
	"context"
	"database/sql"

	"github.com/braccet/bracket/internal/domain"
)

type StageRepository interface {
	GetByTournament(ctx context.Context, tournamentID uint64) ([]*domain.BracketStage, error)
	GetByRound(ctx context.Context, tournamentID uint64, bracketType domain.BracketType, round int) (*domain.BracketStage, error)
	Upsert(ctx context.Context, stage *domain.BracketStage) error
	CreateDefaultStages(ctx context.Context, tournamentID uint64, bracketType domain.BracketType, totalRounds int) error
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

func (r *stageRepository) CreateDefaultStages(ctx context.Context, tournamentID uint64, bracketType domain.BracketType, totalRounds int) error {
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
		defaultName := domain.DefaultStageName(round, totalRounds)
		_, err := stmt.ExecContext(ctx, tournamentID, bracketType, round, defaultName, 1)
		if err != nil {
			return err
		}
	}

	return nil
}
