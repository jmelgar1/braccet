package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/braccet/community/internal/domain"
)

var ErrEloSystemNotFound = errors.New("elo system not found")

type EloSystemRepository interface {
	Create(ctx context.Context, s *domain.EloSystem) error
	GetByID(ctx context.Context, id uint64) (*domain.EloSystem, error)
	GetByCommunity(ctx context.Context, communityID uint64) ([]*domain.EloSystem, error)
	GetDefaultByCommunity(ctx context.Context, communityID uint64) (*domain.EloSystem, error)
	Update(ctx context.Context, s *domain.EloSystem) error
	Delete(ctx context.Context, id uint64) error
	SetDefault(ctx context.Context, communityID, systemID uint64) error
}

type eloSystemRepository struct {
	db *sql.DB
}

func NewEloSystemRepository(db *sql.DB) EloSystemRepository {
	return &eloSystemRepository{db: db}
}

func (r *eloSystemRepository) Create(ctx context.Context, s *domain.EloSystem) error {
	query := `
		INSERT INTO elo_systems (
			community_id, name, description,
			starting_rating, k_factor, floor_rating,
			provisional_games, provisional_k_factor,
			win_streak_enabled, win_streak_threshold, win_streak_bonus,
			decay_enabled, decay_days, decay_amount, decay_floor,
			is_default, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		s.CommunityID, s.Name, s.Description,
		s.StartingRating, s.KFactor, s.FloorRating,
		s.ProvisionalGames, s.ProvisionalKFactor,
		s.WinStreakEnabled, s.WinStreakThreshold, s.WinStreakBonus,
		s.DecayEnabled, s.DecayDays, s.DecayAmount, s.DecayFloor,
		s.IsDefault, s.IsActive,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)

	return err
}

func (r *eloSystemRepository) GetByID(ctx context.Context, id uint64) (*domain.EloSystem, error) {
	query := `
		SELECT id, community_id, name, description,
			starting_rating, k_factor, floor_rating,
			provisional_games, provisional_k_factor,
			win_streak_enabled, win_streak_threshold, win_streak_bonus,
			decay_enabled, decay_days, decay_amount, decay_floor,
			is_default, is_active, created_at, updated_at
		FROM elo_systems
		WHERE id = $1
	`
	s := &domain.EloSystem{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.CommunityID, &s.Name, &s.Description,
		&s.StartingRating, &s.KFactor, &s.FloorRating,
		&s.ProvisionalGames, &s.ProvisionalKFactor,
		&s.WinStreakEnabled, &s.WinStreakThreshold, &s.WinStreakBonus,
		&s.DecayEnabled, &s.DecayDays, &s.DecayAmount, &s.DecayFloor,
		&s.IsDefault, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEloSystemNotFound
		}
		return nil, err
	}

	return s, nil
}

func (r *eloSystemRepository) GetByCommunity(ctx context.Context, communityID uint64) ([]*domain.EloSystem, error) {
	query := `
		SELECT id, community_id, name, description,
			starting_rating, k_factor, floor_rating,
			provisional_games, provisional_k_factor,
			win_streak_enabled, win_streak_threshold, win_streak_bonus,
			decay_enabled, decay_days, decay_amount, decay_floor,
			is_default, is_active, created_at, updated_at
		FROM elo_systems
		WHERE community_id = $1 AND is_active = true
		ORDER BY is_default DESC, name
	`
	rows, err := r.db.QueryContext(ctx, query, communityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var systems []*domain.EloSystem
	for rows.Next() {
		s := &domain.EloSystem{}
		err := rows.Scan(
			&s.ID, &s.CommunityID, &s.Name, &s.Description,
			&s.StartingRating, &s.KFactor, &s.FloorRating,
			&s.ProvisionalGames, &s.ProvisionalKFactor,
			&s.WinStreakEnabled, &s.WinStreakThreshold, &s.WinStreakBonus,
			&s.DecayEnabled, &s.DecayDays, &s.DecayAmount, &s.DecayFloor,
			&s.IsDefault, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		systems = append(systems, s)
	}

	return systems, rows.Err()
}

func (r *eloSystemRepository) GetDefaultByCommunity(ctx context.Context, communityID uint64) (*domain.EloSystem, error) {
	query := `
		SELECT id, community_id, name, description,
			starting_rating, k_factor, floor_rating,
			provisional_games, provisional_k_factor,
			win_streak_enabled, win_streak_threshold, win_streak_bonus,
			decay_enabled, decay_days, decay_amount, decay_floor,
			is_default, is_active, created_at, updated_at
		FROM elo_systems
		WHERE community_id = $1 AND is_default = true AND is_active = true
	`
	s := &domain.EloSystem{}
	err := r.db.QueryRowContext(ctx, query, communityID).Scan(
		&s.ID, &s.CommunityID, &s.Name, &s.Description,
		&s.StartingRating, &s.KFactor, &s.FloorRating,
		&s.ProvisionalGames, &s.ProvisionalKFactor,
		&s.WinStreakEnabled, &s.WinStreakThreshold, &s.WinStreakBonus,
		&s.DecayEnabled, &s.DecayDays, &s.DecayAmount, &s.DecayFloor,
		&s.IsDefault, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEloSystemNotFound
		}
		return nil, err
	}

	return s, nil
}

func (r *eloSystemRepository) Update(ctx context.Context, s *domain.EloSystem) error {
	query := `
		UPDATE elo_systems SET
			name = $1, description = $2,
			starting_rating = $3, k_factor = $4, floor_rating = $5,
			provisional_games = $6, provisional_k_factor = $7,
			win_streak_enabled = $8, win_streak_threshold = $9, win_streak_bonus = $10,
			decay_enabled = $11, decay_days = $12, decay_amount = $13, decay_floor = $14,
			is_active = $15
		WHERE id = $16
	`
	result, err := r.db.ExecContext(ctx, query,
		s.Name, s.Description,
		s.StartingRating, s.KFactor, s.FloorRating,
		s.ProvisionalGames, s.ProvisionalKFactor,
		s.WinStreakEnabled, s.WinStreakThreshold, s.WinStreakBonus,
		s.DecayEnabled, s.DecayDays, s.DecayAmount, s.DecayFloor,
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
		return ErrEloSystemNotFound
	}

	return nil
}

func (r *eloSystemRepository) Delete(ctx context.Context, id uint64) error {
	query := `DELETE FROM elo_systems WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEloSystemNotFound
	}

	return nil
}

func (r *eloSystemRepository) SetDefault(ctx context.Context, communityID, systemID uint64) error {
	// Use a transaction to clear old default and set new one
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing default
	_, err = tx.ExecContext(ctx, `
		UPDATE elo_systems SET is_default = false
		WHERE community_id = $1 AND is_default = true
	`, communityID)
	if err != nil {
		return err
	}

	// Set new default
	result, err := tx.ExecContext(ctx, `
		UPDATE elo_systems SET is_default = true
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
		return ErrEloSystemNotFound
	}

	return tx.Commit()
}
