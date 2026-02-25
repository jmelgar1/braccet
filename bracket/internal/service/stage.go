package service

import (
	"context"
	"errors"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/repository"
)

var (
	ErrInvalidBestOf = errors.New("best_of must be 1, 3, 5, 7, or 9")
	ErrStageNotFound = errors.New("stage not found")
)

type StageService interface {
	GetStages(ctx context.Context, tournamentID uint64) ([]*domain.BracketStage, error)
	GetGroupStages(ctx context.Context, tournamentID, stageID, groupID uint64) ([]*domain.BracketStage, error)
	UpdateStage(ctx context.Context, tournamentID uint64, bracketType domain.BracketType, round int, req UpdateStageRequest) (*domain.BracketStage, error)
	UpdateGroupStage(ctx context.Context, tournamentID, stageID, groupID uint64, bracketType domain.BracketType, round int, req UpdateStageRequest) (*domain.BracketStage, error)
}

type UpdateStageRequest struct {
	StageName *string `json:"stage_name,omitempty"`
	BestOf    *int    `json:"best_of,omitempty"`
}

type stageService struct {
	stageRepo repository.StageRepository
}

func NewStageService(stageRepo repository.StageRepository) StageService {
	return &stageService{stageRepo: stageRepo}
}

func (s *stageService) GetStages(ctx context.Context, tournamentID uint64) ([]*domain.BracketStage, error) {
	return s.stageRepo.GetByTournament(ctx, tournamentID)
}

func (s *stageService) UpdateStage(ctx context.Context, tournamentID uint64, bracketType domain.BracketType, round int, req UpdateStageRequest) (*domain.BracketStage, error) {
	// Validate best_of if provided
	if req.BestOf != nil {
		if !isValidBestOf(*req.BestOf) {
			return nil, ErrInvalidBestOf
		}
	}

	// Get existing stage or create a new one
	stage, err := s.stageRepo.GetByRound(ctx, tournamentID, bracketType, round)
	if err != nil {
		return nil, err
	}

	if stage == nil {
		// Create new stage with defaults
		stage = &domain.BracketStage{
			TournamentID: tournamentID,
			BracketType:  bracketType,
			Round:        round,
			BestOf:       1,
		}
	}

	// Apply updates
	if req.StageName != nil {
		stage.StageName = req.StageName
	}
	if req.BestOf != nil {
		stage.BestOf = *req.BestOf
	}

	// Upsert the stage
	if err := s.stageRepo.Upsert(ctx, stage); err != nil {
		return nil, err
	}

	return stage, nil
}

func (s *stageService) GetGroupStages(ctx context.Context, tournamentID, stageID, groupID uint64) ([]*domain.BracketStage, error) {
	return s.stageRepo.GetByTournamentStageGroup(ctx, tournamentID, stageID, groupID)
}

func (s *stageService) UpdateGroupStage(ctx context.Context, tournamentID, stageID, groupID uint64, bracketType domain.BracketType, round int, req UpdateStageRequest) (*domain.BracketStage, error) {
	// Validate best_of if provided
	if req.BestOf != nil {
		if !isValidBestOf(*req.BestOf) {
			return nil, ErrInvalidBestOf
		}
	}

	// Get existing stage or create a new one
	stage, err := s.stageRepo.GetByRoundInGroup(ctx, tournamentID, stageID, groupID, bracketType, round)
	if err != nil {
		return nil, err
	}

	if stage == nil {
		// Create new stage with defaults
		stage = &domain.BracketStage{
			TournamentID: tournamentID,
			StageID:      &stageID,
			GroupID:      &groupID,
			BracketType:  bracketType,
			Round:        round,
			BestOf:       1,
		}
	}

	// Apply updates
	if req.StageName != nil {
		stage.StageName = req.StageName
	}
	if req.BestOf != nil {
		stage.BestOf = *req.BestOf
	}

	// Upsert the stage
	if err := s.stageRepo.UpsertForGroup(ctx, stage); err != nil {
		return nil, err
	}

	return stage, nil
}

func isValidBestOf(bestOf int) bool {
	return bestOf == 1 || bestOf == 3 || bestOf == 5 || bestOf == 7 || bestOf == 9
}
