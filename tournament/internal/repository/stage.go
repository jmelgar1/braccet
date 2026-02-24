package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/tournament/internal/domain"
)

var (
	ErrStageNotFound = errors.New("stage not found")
	ErrGroupNotFound = errors.New("group not found")
)

// StageRepository handles database operations for multi-stage tournaments
type StageRepository interface {
	// Stage CRUD
	CreateStage(ctx context.Context, stage *domain.TournamentStage) error
	GetStagesByTournament(ctx context.Context, tournamentID uint64) ([]*domain.TournamentStage, error)
	GetStageByID(ctx context.Context, id uint64) (*domain.TournamentStage, error)
	GetActiveStage(ctx context.Context, tournamentID uint64) (*domain.TournamentStage, error)
	UpdateStage(ctx context.Context, stage *domain.TournamentStage) error
	DeleteStagesByTournament(ctx context.Context, tournamentID uint64) error

	// Group CRUD
	CreateGroup(ctx context.Context, group *domain.StageGroup) error
	CreateGroups(ctx context.Context, groups []*domain.StageGroup) error
	GetGroupsByStage(ctx context.Context, stageID uint64) ([]*domain.StageGroup, error)
	GetGroupByID(ctx context.Context, id uint64) (*domain.StageGroup, error)
	DeleteGroupsByStage(ctx context.Context, stageID uint64) error

	// Group Assignments
	CreateAssignments(ctx context.Context, assignments []*domain.GroupAssignment) error
	GetAssignmentsByGroup(ctx context.Context, groupID uint64) ([]*domain.GroupAssignment, error)
	GetAssignmentsByStage(ctx context.Context, stageID uint64) ([]*domain.GroupAssignment, error)
	DeleteAssignmentsByStage(ctx context.Context, stageID uint64) error

	// Ranking Criteria
	CreateRankingCriteria(ctx context.Context, criteria []*domain.StageRankingCriteria) error
	GetRankingCriteria(ctx context.Context, stageID uint64) ([]*domain.StageRankingCriteria, error)
	DeleteRankingCriteria(ctx context.Context, stageID uint64) error

	// Placement Config
	CreatePlacementConfig(ctx context.Context, config *domain.PlacementMatchConfig) error
	GetPlacementConfig(ctx context.Context, stageID uint64) (*domain.PlacementMatchConfig, error)
	UpdatePlacementConfig(ctx context.Context, config *domain.PlacementMatchConfig) error
	DeletePlacementConfig(ctx context.Context, stageID uint64) error

	// Seed Assignments (pre-start manual seeding)
	GetSeedAssignmentsByStage(ctx context.Context, stageID uint64) ([]*domain.StageSeedAssignment, error)
	UpdateSeedAssignments(ctx context.Context, stageID, tournamentID uint64, assignments []*domain.StageSeedAssignment) error
	DeleteSeedAssignmentsByStage(ctx context.Context, stageID uint64) error
}

type stageRepository struct {
	db *sql.DB
}

func NewStageRepository(db *sql.DB) StageRepository {
	return &stageRepository{db: db}
}

// Stage CRUD

func (r *stageRepository) CreateStage(ctx context.Context, stage *domain.TournamentStage) error {
	query := `
		INSERT INTO tournament_stages (tournament_id, stage_order, stage_type, format, participants_per_group, advancing_per_group, swiss_rounds, is_active, is_complete)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		stage.TournamentID, stage.StageOrder, stage.StageType, stage.Format,
		stage.ParticipantsPerGroup, stage.AdvancingPerGroup, stage.SwissRounds,
		stage.IsActive, stage.IsComplete,
	).Scan(&stage.ID, &stage.CreatedAt, &stage.UpdatedAt)
}

func (r *stageRepository) GetStagesByTournament(ctx context.Context, tournamentID uint64) ([]*domain.TournamentStage, error) {
	query := `
		SELECT id, tournament_id, stage_order, stage_type, format, participants_per_group, advancing_per_group, swiss_rounds, is_active, is_complete, created_at, updated_at
		FROM tournament_stages
		WHERE tournament_id = $1
		ORDER BY stage_order ASC
	`
	rows, err := r.db.QueryContext(ctx, query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []*domain.TournamentStage
	for rows.Next() {
		s := &domain.TournamentStage{}
		if err := rows.Scan(
			&s.ID, &s.TournamentID, &s.StageOrder, &s.StageType, &s.Format,
			&s.ParticipantsPerGroup, &s.AdvancingPerGroup, &s.SwissRounds,
			&s.IsActive, &s.IsComplete, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		stages = append(stages, s)
	}

	return stages, rows.Err()
}

func (r *stageRepository) GetStageByID(ctx context.Context, id uint64) (*domain.TournamentStage, error) {
	query := `
		SELECT id, tournament_id, stage_order, stage_type, format, participants_per_group, advancing_per_group, swiss_rounds, is_active, is_complete, created_at, updated_at
		FROM tournament_stages
		WHERE id = $1
	`
	s := &domain.TournamentStage{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.TournamentID, &s.StageOrder, &s.StageType, &s.Format,
		&s.ParticipantsPerGroup, &s.AdvancingPerGroup, &s.SwissRounds,
		&s.IsActive, &s.IsComplete, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStageNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *stageRepository) GetActiveStage(ctx context.Context, tournamentID uint64) (*domain.TournamentStage, error) {
	query := `
		SELECT id, tournament_id, stage_order, stage_type, format, participants_per_group, advancing_per_group, swiss_rounds, is_active, is_complete, created_at, updated_at
		FROM tournament_stages
		WHERE tournament_id = $1 AND is_active = true
		LIMIT 1
	`
	s := &domain.TournamentStage{}
	err := r.db.QueryRowContext(ctx, query, tournamentID).Scan(
		&s.ID, &s.TournamentID, &s.StageOrder, &s.StageType, &s.Format,
		&s.ParticipantsPerGroup, &s.AdvancingPerGroup, &s.SwissRounds,
		&s.IsActive, &s.IsComplete, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No active stage is not an error
		}
		return nil, err
	}
	return s, nil
}

func (r *stageRepository) UpdateStage(ctx context.Context, stage *domain.TournamentStage) error {
	query := `
		UPDATE tournament_stages
		SET format = $1, participants_per_group = $2, advancing_per_group = $3,
		    swiss_rounds = $4, is_active = $5, is_complete = $6
		WHERE id = $7
		RETURNING updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		stage.Format, stage.ParticipantsPerGroup, stage.AdvancingPerGroup,
		stage.SwissRounds, stage.IsActive, stage.IsComplete, stage.ID,
	).Scan(&stage.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrStageNotFound
		}
		return err
	}
	return nil
}

func (r *stageRepository) DeleteStagesByTournament(ctx context.Context, tournamentID uint64) error {
	query := `DELETE FROM tournament_stages WHERE tournament_id = $1`
	_, err := r.db.ExecContext(ctx, query, tournamentID)
	return err
}

// Group CRUD

func (r *stageRepository) CreateGroup(ctx context.Context, group *domain.StageGroup) error {
	query := `
		INSERT INTO stage_groups (tournament_id, stage_id, group_name, group_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		group.TournamentID, group.StageID, group.GroupName, group.GroupOrder,
	).Scan(&group.ID, &group.CreatedAt)
}

func (r *stageRepository) CreateGroups(ctx context.Context, groups []*domain.StageGroup) error {
	if len(groups) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO stage_groups (tournament_id, stage_id, group_name, group_order)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, g := range groups {
		if err := stmt.QueryRowContext(ctx,
			g.TournamentID, g.StageID, g.GroupName, g.GroupOrder,
		).Scan(&g.ID, &g.CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *stageRepository) GetGroupsByStage(ctx context.Context, stageID uint64) ([]*domain.StageGroup, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_name, group_order, created_at
		FROM stage_groups
		WHERE stage_id = $1
		ORDER BY group_order ASC
	`
	rows, err := r.db.QueryContext(ctx, query, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*domain.StageGroup
	for rows.Next() {
		g := &domain.StageGroup{}
		if err := rows.Scan(
			&g.ID, &g.TournamentID, &g.StageID, &g.GroupName, &g.GroupOrder, &g.CreatedAt,
		); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}

	return groups, rows.Err()
}

func (r *stageRepository) GetGroupByID(ctx context.Context, id uint64) (*domain.StageGroup, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_name, group_order, created_at
		FROM stage_groups
		WHERE id = $1
	`
	g := &domain.StageGroup{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&g.ID, &g.TournamentID, &g.StageID, &g.GroupName, &g.GroupOrder, &g.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, err
	}
	return g, nil
}

func (r *stageRepository) DeleteGroupsByStage(ctx context.Context, stageID uint64) error {
	query := `DELETE FROM stage_groups WHERE stage_id = $1`
	_, err := r.db.ExecContext(ctx, query, stageID)
	return err
}

// Group Assignments

func (r *stageRepository) CreateAssignments(ctx context.Context, assignments []*domain.GroupAssignment) error {
	if len(assignments) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO group_assignments (tournament_id, stage_id, group_id, participant_id, seed_in_group)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, a := range assignments {
		if err := stmt.QueryRowContext(ctx,
			a.TournamentID, a.StageID, a.GroupID, a.ParticipantID, a.SeedInGroup,
		).Scan(&a.ID, &a.CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *stageRepository) GetAssignmentsByGroup(ctx context.Context, groupID uint64) ([]*domain.GroupAssignment, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_id, participant_id, seed_in_group, created_at
		FROM group_assignments
		WHERE group_id = $1
		ORDER BY seed_in_group ASC NULLS LAST, id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []*domain.GroupAssignment
	for rows.Next() {
		a := &domain.GroupAssignment{}
		if err := rows.Scan(
			&a.ID, &a.TournamentID, &a.StageID, &a.GroupID, &a.ParticipantID, &a.SeedInGroup, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}

	return assignments, rows.Err()
}

func (r *stageRepository) GetAssignmentsByStage(ctx context.Context, stageID uint64) ([]*domain.GroupAssignment, error) {
	query := `
		SELECT id, tournament_id, stage_id, group_id, participant_id, seed_in_group, created_at
		FROM group_assignments
		WHERE stage_id = $1
		ORDER BY group_id ASC, seed_in_group ASC NULLS LAST, id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []*domain.GroupAssignment
	for rows.Next() {
		a := &domain.GroupAssignment{}
		if err := rows.Scan(
			&a.ID, &a.TournamentID, &a.StageID, &a.GroupID, &a.ParticipantID, &a.SeedInGroup, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}

	return assignments, rows.Err()
}

func (r *stageRepository) DeleteAssignmentsByStage(ctx context.Context, stageID uint64) error {
	query := `DELETE FROM group_assignments WHERE stage_id = $1`
	_, err := r.db.ExecContext(ctx, query, stageID)
	return err
}

// Ranking Criteria

func (r *stageRepository) CreateRankingCriteria(ctx context.Context, criteria []*domain.StageRankingCriteria) error {
	if len(criteria) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO stage_ranking_criteria (stage_id, criterion, priority)
		VALUES ($1, $2, $3)
		RETURNING id
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range criteria {
		if err := stmt.QueryRowContext(ctx, c.StageID, c.Criterion, c.Priority).Scan(&c.ID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *stageRepository) GetRankingCriteria(ctx context.Context, stageID uint64) ([]*domain.StageRankingCriteria, error) {
	query := `
		SELECT id, stage_id, criterion, priority
		FROM stage_ranking_criteria
		WHERE stage_id = $1
		ORDER BY priority ASC
	`
	rows, err := r.db.QueryContext(ctx, query, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var criteria []*domain.StageRankingCriteria
	for rows.Next() {
		c := &domain.StageRankingCriteria{}
		if err := rows.Scan(&c.ID, &c.StageID, &c.Criterion, &c.Priority); err != nil {
			return nil, err
		}
		criteria = append(criteria, c)
	}

	return criteria, rows.Err()
}

func (r *stageRepository) DeleteRankingCriteria(ctx context.Context, stageID uint64) error {
	query := `DELETE FROM stage_ranking_criteria WHERE stage_id = $1`
	_, err := r.db.ExecContext(ctx, query, stageID)
	return err
}

// Placement Config

func (r *stageRepository) CreatePlacementConfig(ctx context.Context, config *domain.PlacementMatchConfig) error {
	query := `
		INSERT INTO placement_match_config (stage_id, enabled, depth)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	return r.db.QueryRowContext(ctx, query, config.StageID, config.Enabled, config.Depth).Scan(&config.ID)
}

func (r *stageRepository) GetPlacementConfig(ctx context.Context, stageID uint64) (*domain.PlacementMatchConfig, error) {
	query := `
		SELECT id, stage_id, enabled, depth
		FROM placement_match_config
		WHERE stage_id = $1
	`
	c := &domain.PlacementMatchConfig{}
	err := r.db.QueryRowContext(ctx, query, stageID).Scan(&c.ID, &c.StageID, &c.Enabled, &c.Depth)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No config is not an error
		}
		return nil, err
	}
	return c, nil
}

func (r *stageRepository) UpdatePlacementConfig(ctx context.Context, config *domain.PlacementMatchConfig) error {
	query := `
		UPDATE placement_match_config
		SET enabled = $1, depth = $2
		WHERE stage_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, config.Enabled, config.Depth, config.StageID)
	return err
}

func (r *stageRepository) DeletePlacementConfig(ctx context.Context, stageID uint64) error {
	query := `DELETE FROM placement_match_config WHERE stage_id = $1`
	_, err := r.db.ExecContext(ctx, query, stageID)
	return err
}

// Seed Assignments (pre-start manual seeding)

func (r *stageRepository) GetSeedAssignmentsByStage(ctx context.Context, stageID uint64) ([]*domain.StageSeedAssignment, error) {
	query := `
		SELECT id, tournament_id, stage_id, participant_id, target_group_order, created_at, updated_at
		FROM stage_seed_assignments
		WHERE stage_id = $1
		ORDER BY target_group_order ASC, id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []*domain.StageSeedAssignment
	for rows.Next() {
		a := &domain.StageSeedAssignment{}
		if err := rows.Scan(
			&a.ID, &a.TournamentID, &a.StageID, &a.ParticipantID,
			&a.TargetGroupOrder, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}

	return assignments, rows.Err()
}

func (r *stageRepository) UpdateSeedAssignments(ctx context.Context, stageID, tournamentID uint64, assignments []*domain.StageSeedAssignment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing assignments for this stage
	_, err = tx.ExecContext(ctx, `DELETE FROM stage_seed_assignments WHERE stage_id = $1`, stageID)
	if err != nil {
		return err
	}

	// Insert new assignments if any
	if len(assignments) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO stage_seed_assignments (tournament_id, stage_id, participant_id, target_group_order)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, updated_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, a := range assignments {
			a.TournamentID = tournamentID
			a.StageID = stageID
			if err := stmt.QueryRowContext(ctx,
				a.TournamentID, a.StageID, a.ParticipantID, a.TargetGroupOrder,
			).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *stageRepository) DeleteSeedAssignmentsByStage(ctx context.Context, stageID uint64) error {
	query := `DELETE FROM stage_seed_assignments WHERE stage_id = $1`
	_, err := r.db.ExecContext(ctx, query, stageID)
	return err
}
