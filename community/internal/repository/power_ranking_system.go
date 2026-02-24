package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/community/internal/domain"
)

var ErrPRSystemNotFound = errors.New("power ranking system not found")

type PowerRankingSystemRepository interface {
	Create(ctx context.Context, s *domain.PowerRankingSystem) error
	GetByID(ctx context.Context, id uint64) (*domain.PowerRankingSystem, error)
	GetByCommunity(ctx context.Context, communityID uint64) ([]*domain.PowerRankingSystem, error)
	GetDefaultByCommunity(ctx context.Context, communityID uint64) (*domain.PowerRankingSystem, error)
	Update(ctx context.Context, s *domain.PowerRankingSystem) error
	Delete(ctx context.Context, id uint64) error
	SetDefault(ctx context.Context, communityID, systemID uint64) error
}

type powerRankingSystemRepository struct {
	db *sql.DB
}

func NewPowerRankingSystemRepository(db *sql.DB) PowerRankingSystemRepository {
	return &powerRankingSystemRepository{db: db}
}

func (r *powerRankingSystemRepository) Create(ctx context.Context, s *domain.PowerRankingSystem) error {
	query := `
		INSERT INTO power_ranking_systems (
			community_id, name, description,
			achievements_weight, form_weight, lan_weight,
			achievement_decay_months, achievement_window_months,
			form_window_months, lan_results_count,
			is_default, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		s.CommunityID, s.Name, s.Description,
		s.AchievementsWeight, s.FormWeight, s.LANWeight,
		s.AchievementDecayMonths, s.AchievementWindowMonths,
		s.FormWindowMonths, s.LANResultsCount,
		s.IsDefault, s.IsActive,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)

	return err
}

func (r *powerRankingSystemRepository) GetByID(ctx context.Context, id uint64) (*domain.PowerRankingSystem, error) {
	query := `
		SELECT id, community_id, name, description,
			achievements_weight, form_weight, lan_weight,
			achievement_decay_months, achievement_window_months,
			form_window_months, lan_results_count,
			is_default, is_active, created_at, updated_at
		FROM power_ranking_systems
		WHERE id = $1
	`
	s := &domain.PowerRankingSystem{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.CommunityID, &s.Name, &s.Description,
		&s.AchievementsWeight, &s.FormWeight, &s.LANWeight,
		&s.AchievementDecayMonths, &s.AchievementWindowMonths,
		&s.FormWindowMonths, &s.LANResultsCount,
		&s.IsDefault, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPRSystemNotFound
		}
		return nil, err
	}

	return s, nil
}

func (r *powerRankingSystemRepository) GetByCommunity(ctx context.Context, communityID uint64) ([]*domain.PowerRankingSystem, error) {
	query := `
		SELECT id, community_id, name, description,
			achievements_weight, form_weight, lan_weight,
			achievement_decay_months, achievement_window_months,
			form_window_months, lan_results_count,
			is_default, is_active, created_at, updated_at
		FROM power_ranking_systems
		WHERE community_id = $1 AND is_active = true
		ORDER BY is_default DESC, name
	`
	rows, err := r.db.QueryContext(ctx, query, communityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var systems []*domain.PowerRankingSystem
	for rows.Next() {
		s := &domain.PowerRankingSystem{}
		err := rows.Scan(
			&s.ID, &s.CommunityID, &s.Name, &s.Description,
			&s.AchievementsWeight, &s.FormWeight, &s.LANWeight,
			&s.AchievementDecayMonths, &s.AchievementWindowMonths,
			&s.FormWindowMonths, &s.LANResultsCount,
			&s.IsDefault, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		systems = append(systems, s)
	}

	return systems, rows.Err()
}

func (r *powerRankingSystemRepository) GetDefaultByCommunity(ctx context.Context, communityID uint64) (*domain.PowerRankingSystem, error) {
	query := `
		SELECT id, community_id, name, description,
			achievements_weight, form_weight, lan_weight,
			achievement_decay_months, achievement_window_months,
			form_window_months, lan_results_count,
			is_default, is_active, created_at, updated_at
		FROM power_ranking_systems
		WHERE community_id = $1 AND is_default = true AND is_active = true
	`
	s := &domain.PowerRankingSystem{}
	err := r.db.QueryRowContext(ctx, query, communityID).Scan(
		&s.ID, &s.CommunityID, &s.Name, &s.Description,
		&s.AchievementsWeight, &s.FormWeight, &s.LANWeight,
		&s.AchievementDecayMonths, &s.AchievementWindowMonths,
		&s.FormWindowMonths, &s.LANResultsCount,
		&s.IsDefault, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPRSystemNotFound
		}
		return nil, err
	}

	return s, nil
}

func (r *powerRankingSystemRepository) Update(ctx context.Context, s *domain.PowerRankingSystem) error {
	query := `
		UPDATE power_ranking_systems SET
			name = $1, description = $2,
			achievements_weight = $3, form_weight = $4, lan_weight = $5,
			achievement_decay_months = $6, achievement_window_months = $7,
			form_window_months = $8, lan_results_count = $9,
			is_active = $10
		WHERE id = $11
	`
	result, err := r.db.ExecContext(ctx, query,
		s.Name, s.Description,
		s.AchievementsWeight, s.FormWeight, s.LANWeight,
		s.AchievementDecayMonths, s.AchievementWindowMonths,
		s.FormWindowMonths, s.LANResultsCount,
		s.IsActive, s.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPRSystemNotFound
	}

	return nil
}

func (r *powerRankingSystemRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM power_ranking_systems WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPRSystemNotFound
	}

	return nil
}

func (r *powerRankingSystemRepository) SetDefault(ctx context.Context, communityID, systemID uint64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing default
	_, err = tx.ExecContext(ctx, `
		UPDATE power_ranking_systems SET is_default = false
		WHERE community_id = $1 AND is_default = true
	`, communityID)
	if err != nil {
		return err
	}

	// Set new default
	result, err := tx.ExecContext(ctx, `
		UPDATE power_ranking_systems SET is_default = true
		WHERE id = $1 AND community_id = $2
	`, systemID, communityID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPRSystemNotFound
	}

	return tx.Commit()
}
