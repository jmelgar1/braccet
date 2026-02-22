package service

import (
	"context"
	"errors"
	"math"

	"github.com/braccet/community/internal/domain"
	"github.com/braccet/community/internal/repository"
)

var (
	ErrEloSystemNotFound = errors.New("elo system not found")
	ErrMemberNotFound    = errors.New("member not found")
)

// ProcessMatchRequest contains the data needed to process ELO updates for a match
type ProcessMatchRequest struct {
	EloSystemID    uint64
	MatchID        uint64
	TournamentID   uint64
	WinnerMemberID uint64
	LoserMemberID  uint64
}

// ProcessMatchResponse contains the results of ELO processing
type ProcessMatchResponse struct {
	WinnerRatingBefore int
	WinnerRatingAfter  int
	WinnerChange       int
	LoserRatingBefore  int
	LoserRatingAfter   int
	LoserChange        int
}

type EloService interface {
	// System management
	CreateSystem(ctx context.Context, system *domain.EloSystem) error
	GetSystem(ctx context.Context, id uint64) (*domain.EloSystem, error)
	GetSystemsByCommunity(ctx context.Context, communityID uint64) ([]*domain.EloSystem, error)
	UpdateSystem(ctx context.Context, system *domain.EloSystem) error
	DeleteSystem(ctx context.Context, id uint64) error
	SetDefaultSystem(ctx context.Context, communityID, systemID uint64) error

	// Rating operations
	GetMemberRating(ctx context.Context, memberID, systemID uint64) (*domain.MemberEloRating, error)
	GetMemberRatings(ctx context.Context, memberID uint64) ([]*domain.MemberEloRating, error)
	GetLeaderboard(ctx context.Context, systemID uint64, limit int) ([]*domain.MemberEloRating, error)

	// Match result processing
	ProcessMatchResult(ctx context.Context, req ProcessMatchRequest) (*ProcessMatchResponse, error)

	// History
	GetMemberHistory(ctx context.Context, memberID, systemID uint64, limit int) ([]*domain.EloHistory, error)
}

type eloService struct {
	systemRepo  repository.EloSystemRepository
	ratingRepo  repository.MemberEloRatingRepository
	historyRepo repository.EloHistoryRepository
	memberRepo  repository.MemberRepository
}

func NewEloService(
	systemRepo repository.EloSystemRepository,
	ratingRepo repository.MemberEloRatingRepository,
	historyRepo repository.EloHistoryRepository,
	memberRepo repository.MemberRepository,
) EloService {
	return &eloService{
		systemRepo:  systemRepo,
		ratingRepo:  ratingRepo,
		historyRepo: historyRepo,
		memberRepo:  memberRepo,
	}
}

// CreateSystem creates a new ELO system for a community
func (s *eloService) CreateSystem(ctx context.Context, system *domain.EloSystem) error {
	return s.systemRepo.Create(ctx, system)
}

// GetSystem retrieves an ELO system by ID
func (s *eloService) GetSystem(ctx context.Context, id uint64) (*domain.EloSystem, error) {
	return s.systemRepo.GetByID(ctx, id)
}

// GetSystemsByCommunity retrieves all active ELO systems for a community
func (s *eloService) GetSystemsByCommunity(ctx context.Context, communityID uint64) ([]*domain.EloSystem, error) {
	return s.systemRepo.GetByCommunity(ctx, communityID)
}

// UpdateSystem updates an existing ELO system
func (s *eloService) UpdateSystem(ctx context.Context, system *domain.EloSystem) error {
	return s.systemRepo.Update(ctx, system)
}

// DeleteSystem deletes an ELO system
func (s *eloService) DeleteSystem(ctx context.Context, id uint64) error {
	return s.systemRepo.Delete(ctx, id)
}

// SetDefaultSystem sets a system as the default for a community
func (s *eloService) SetDefaultSystem(ctx context.Context, communityID, systemID uint64) error {
	return s.systemRepo.SetDefault(ctx, communityID, systemID)
}

// GetMemberRating retrieves a member's rating for a specific ELO system
func (s *eloService) GetMemberRating(ctx context.Context, memberID, systemID uint64) (*domain.MemberEloRating, error) {
	return s.ratingRepo.GetByMemberAndSystem(ctx, memberID, systemID)
}

// GetMemberRatings retrieves all ratings for a member across all systems
func (s *eloService) GetMemberRatings(ctx context.Context, memberID uint64) ([]*domain.MemberEloRating, error) {
	return s.ratingRepo.GetByMember(ctx, memberID)
}

// GetLeaderboard retrieves the top-rated members for a system
func (s *eloService) GetLeaderboard(ctx context.Context, systemID uint64, limit int) ([]*domain.MemberEloRating, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.ratingRepo.GetLeaderboard(ctx, systemID, limit)
}

// GetMemberHistory retrieves rating history for a member in a system
func (s *eloService) GetMemberHistory(ctx context.Context, memberID, systemID uint64, limit int) ([]*domain.EloHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.historyRepo.GetByMemberAndSystem(ctx, memberID, systemID, limit)
}

// ProcessMatchResult calculates and applies ELO changes for a completed match
func (s *eloService) ProcessMatchResult(ctx context.Context, req ProcessMatchRequest) (*ProcessMatchResponse, error) {
	// Get the ELO system configuration
	system, err := s.systemRepo.GetByID(ctx, req.EloSystemID)
	if err != nil {
		return nil, err
	}

	// Get or create ratings for both players
	winnerRating, err := s.getOrCreateRating(ctx, req.WinnerMemberID, req.EloSystemID, system)
	if err != nil {
		return nil, err
	}

	loserRating, err := s.getOrCreateRating(ctx, req.LoserMemberID, req.EloSystemID, system)
	if err != nil {
		return nil, err
	}

	// Store original ratings before changes
	winnerRatingBefore := winnerRating.Rating
	loserRatingBefore := loserRating.Rating

	// Calculate expected scores using ELO formula
	winnerExpected := s.calculateExpectedScore(winnerRating.Rating, loserRating.Rating)
	loserExpected := 1.0 - winnerExpected

	// Determine K-factors based on provisional status
	winnerK := s.getKFactor(system, winnerRating)
	loserK := s.getKFactor(system, loserRating)

	// Calculate base rating changes
	// Winner gets actualScore = 1.0, Loser gets actualScore = 0.0
	winnerChange := int(math.Round(float64(winnerK) * (1.0 - winnerExpected)))
	loserChange := int(math.Round(float64(loserK) * (0.0 - loserExpected)))

	// Apply win streak bonus
	winStreakBonus := 0
	if system.WinStreakEnabled {
		newStreak := winnerRating.CurrentWinStreak + 1
		if newStreak >= system.WinStreakThreshold {
			winStreakBonus = system.WinStreakBonus
			winnerChange += winStreakBonus
		}
	}

	// Calculate new ratings with floor enforcement
	winnerNewRating := max(winnerRatingBefore+winnerChange, system.FloorRating)
	loserNewRating := max(loserRatingBefore+loserChange, system.FloorRating)

	// Recalculate actual changes after floor enforcement
	actualWinnerChange := winnerNewRating - winnerRatingBefore
	actualLoserChange := loserNewRating - loserRatingBefore

	// Update winner rating
	winnerRating.Rating = winnerNewRating
	winnerRating.GamesPlayed++
	winnerRating.GamesWon++
	winnerRating.CurrentWinStreak++
	winnerRating.HighestRating = max(winnerRating.HighestRating, winnerNewRating)
	if err := s.ratingRepo.Update(ctx, winnerRating); err != nil {
		return nil, err
	}

	// Update loser rating
	loserRating.Rating = loserNewRating
	loserRating.GamesPlayed++
	loserRating.CurrentWinStreak = 0
	loserRating.LowestRating = min(loserRating.LowestRating, loserNewRating)
	if err := s.ratingRepo.Update(ctx, loserRating); err != nil {
		return nil, err
	}

	// Record history for winner
	isWinnerTrue := true
	winnerHistory := &domain.EloHistory{
		MemberID:             req.WinnerMemberID,
		EloSystemID:          req.EloSystemID,
		ChangeType:           domain.EloChangeMatch,
		RatingBefore:         winnerRatingBefore,
		RatingChange:         actualWinnerChange,
		RatingAfter:          winnerNewRating,
		MatchID:              &req.MatchID,
		TournamentID:         &req.TournamentID,
		OpponentMemberID:     &req.LoserMemberID,
		OpponentRatingBefore: &loserRatingBefore,
		IsWinner:             &isWinnerTrue,
		KFactorUsed:          &winnerK,
		ExpectedScore:        &winnerExpected,
		WinStreakBonus:       winStreakBonus,
	}
	if err := s.historyRepo.Create(ctx, winnerHistory); err != nil {
		return nil, err
	}

	// Record history for loser
	isWinnerFalse := false
	loserHistory := &domain.EloHistory{
		MemberID:             req.LoserMemberID,
		EloSystemID:          req.EloSystemID,
		ChangeType:           domain.EloChangeMatch,
		RatingBefore:         loserRatingBefore,
		RatingChange:         actualLoserChange,
		RatingAfter:          loserNewRating,
		MatchID:              &req.MatchID,
		TournamentID:         &req.TournamentID,
		OpponentMemberID:     &req.WinnerMemberID,
		OpponentRatingBefore: &winnerRatingBefore,
		IsWinner:             &isWinnerFalse,
		KFactorUsed:          &loserK,
		ExpectedScore:        &loserExpected,
	}
	if err := s.historyRepo.Create(ctx, loserHistory); err != nil {
		return nil, err
	}

	// Update community_members stats (matches_played, matches_won, elo_rating)
	if err := s.memberRepo.IncrementMatchStats(ctx, req.WinnerMemberID, true, &winnerNewRating); err != nil {
		// Log but don't fail - ELO was already updated successfully
		// This is a denormalized cache field
	}
	if err := s.memberRepo.IncrementMatchStats(ctx, req.LoserMemberID, false, &loserNewRating); err != nil {
		// Log but don't fail
	}

	return &ProcessMatchResponse{
		WinnerRatingBefore: winnerRatingBefore,
		WinnerRatingAfter:  winnerNewRating,
		WinnerChange:       actualWinnerChange,
		LoserRatingBefore:  loserRatingBefore,
		LoserRatingAfter:   loserNewRating,
		LoserChange:        actualLoserChange,
	}, nil
}

// calculateExpectedScore computes the expected score using the ELO formula
// E_A = 1 / (1 + 10^((R_B - R_A) / 400))
func (s *eloService) calculateExpectedScore(ratingA, ratingB int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(ratingB-ratingA)/400.0))
}

// getKFactor returns the appropriate K-factor based on games played
func (s *eloService) getKFactor(system *domain.EloSystem, rating *domain.MemberEloRating) int {
	if rating.GamesPlayed < system.ProvisionalGames {
		return system.ProvisionalKFactor
	}
	return system.KFactor
}

// getOrCreateRating retrieves an existing rating or creates a new one
func (s *eloService) getOrCreateRating(ctx context.Context, memberID, systemID uint64, system *domain.EloSystem) (*domain.MemberEloRating, error) {
	rating, err := s.ratingRepo.GetByMemberAndSystem(ctx, memberID, systemID)
	if err == nil {
		return rating, nil
	}

	if !errors.Is(err, repository.ErrMemberEloRatingNotFound) {
		return nil, err
	}

	// Create new rating with starting values
	newRating := &domain.MemberEloRating{
		MemberID:      memberID,
		EloSystemID:   systemID,
		Rating:        system.StartingRating,
		HighestRating: system.StartingRating,
		LowestRating:  system.StartingRating,
	}
	if err := s.ratingRepo.Create(ctx, newRating); err != nil {
		return nil, err
	}

	// Record initial rating in history
	initialHistory := &domain.EloHistory{
		MemberID:     memberID,
		EloSystemID:  systemID,
		ChangeType:   domain.EloChangeInitial,
		RatingBefore: 0,
		RatingChange: system.StartingRating,
		RatingAfter:  system.StartingRating,
	}
	_ = s.historyRepo.Create(ctx, initialHistory) // Don't fail if history insert fails

	return newRating, nil
}
