package service

import (
	"context"
	"errors"
	"log"
	"math"
	"time"

	"github.com/braccet/community/internal/domain"
	"github.com/braccet/community/internal/repository"
)

var (
	ErrPRSystemNotFound = errors.New("power ranking system not found")
)

// PowerRankingService handles power ranking operations
type PowerRankingService interface {
	// System CRUD
	CreateSystem(ctx context.Context, system *domain.PowerRankingSystem) error
	GetSystem(ctx context.Context, id uint64) (*domain.PowerRankingSystem, error)
	GetSystemsByCommunity(ctx context.Context, communityID uint64) ([]*domain.PowerRankingSystem, error)
	GetDefaultByCommunity(ctx context.Context, communityID uint64) (*domain.PowerRankingSystem, error)
	UpdateSystem(ctx context.Context, system *domain.PowerRankingSystem) error
	DeleteSystem(ctx context.Context, id uint64) error
	SetDefaultSystem(ctx context.Context, communityID, systemID uint64) error

	// Data recording (called from bracket service)
	ProcessPlacement(ctx context.Context, req *domain.ProcessPlacementRequest) error
	ProcessMatch(ctx context.Context, req *domain.ProcessPRMatchRequest) error

	// Calculation & retrieval
	CalculateRankings(ctx context.Context, systemID uint64) error
	GetLeaderboard(ctx context.Context, systemID uint64, limit int) ([]*domain.MemberPowerRanking, error)
	GetMemberRanking(ctx context.Context, memberID, systemID uint64) (*domain.MemberPowerRanking, error)
	GetMemberPlacements(ctx context.Context, memberID, systemID uint64) ([]*domain.TournamentPlacement, error)

	// Reversion
	RevertTournamentPlacements(ctx context.Context, tournamentID uint64) error
}

type powerRankingService struct {
	systemRepo     repository.PowerRankingSystemRepository
	placementRepo  repository.TournamentPlacementRepository
	rankingRepo    repository.MemberPowerRankingRepository
	matchCacheRepo repository.PowerRankingMatchCacheRepository
	memberRepo     repository.MemberRepository
}

func NewPowerRankingService(
	systemRepo repository.PowerRankingSystemRepository,
	placementRepo repository.TournamentPlacementRepository,
	rankingRepo repository.MemberPowerRankingRepository,
	matchCacheRepo repository.PowerRankingMatchCacheRepository,
	memberRepo repository.MemberRepository,
) PowerRankingService {
	return &powerRankingService{
		systemRepo:     systemRepo,
		placementRepo:  placementRepo,
		rankingRepo:    rankingRepo,
		matchCacheRepo: matchCacheRepo,
		memberRepo:     memberRepo,
	}
}

// Base points by tournament class
var classBasePoints = map[domain.TournamentClass]int{
	domain.TournamentClassMajor:       1000,
	domain.TournamentClassWorldLAN:    800,
	domain.TournamentClassContinental: 600,
	domain.TournamentClassRegional:    400,
	domain.TournamentClassOnline:      200,
}

// Placement multipliers (1st through 8th)
var placementMultipliers = []float64{1.0, 0.75, 0.60, 0.45, 0.35, 0.28, 0.22, 0.18}

// =============================================================================
// System CRUD
// =============================================================================

func (s *powerRankingService) CreateSystem(ctx context.Context, system *domain.PowerRankingSystem) error {
	return s.systemRepo.Create(ctx, system)
}

func (s *powerRankingService) GetSystem(ctx context.Context, id uint64) (*domain.PowerRankingSystem, error) {
	return s.systemRepo.GetByID(ctx, id)
}

func (s *powerRankingService) GetSystemsByCommunity(ctx context.Context, communityID uint64) ([]*domain.PowerRankingSystem, error) {
	return s.systemRepo.GetByCommunity(ctx, communityID)
}

func (s *powerRankingService) GetDefaultByCommunity(ctx context.Context, communityID uint64) (*domain.PowerRankingSystem, error) {
	return s.systemRepo.GetDefaultByCommunity(ctx, communityID)
}

func (s *powerRankingService) UpdateSystem(ctx context.Context, system *domain.PowerRankingSystem) error {
	return s.systemRepo.Update(ctx, system)
}

func (s *powerRankingService) DeleteSystem(ctx context.Context, id uint64) error {
	// Delete all rankings for this system first
	if err := s.rankingRepo.DeleteBySystem(ctx, id); err != nil {
		return err
	}
	return s.systemRepo.Delete(ctx, id)
}

func (s *powerRankingService) SetDefaultSystem(ctx context.Context, communityID, systemID uint64) error {
	return s.systemRepo.SetDefault(ctx, communityID, systemID)
}

// =============================================================================
// Data Recording
// =============================================================================

// ProcessPlacement records a tournament placement and calculates raw points
func (s *powerRankingService) ProcessPlacement(ctx context.Context, req *domain.ProcessPlacementRequest) error {
	// Calculate raw points at time of recording
	rawPoints := s.calculatePlacementPoints(
		req.Placement,
		req.ParticipantCount,
		req.TournamentClass,
		req.PrizePoolUSD,
	)

	placement := &domain.TournamentPlacement{
		MemberID:          req.MemberID,
		PRSystemID:        req.PRSystemID,
		TournamentID:      req.TournamentID,
		Placement:         req.Placement,
		ParticipantCount:  req.ParticipantCount,
		TournamentClass:   req.TournamentClass,
		PrizePoolUSD:      req.PrizePoolUSD,
		IsBo3Playoffs:     req.IsBo3Playoffs,
		AvgOpponentRating: req.AvgOpponentRating,
		IsLAN:             req.IsLAN,
		RawPoints:         rawPoints,
		CompletedAt:       req.CompletedAt,
	}

	return s.placementRepo.Upsert(ctx, placement)
}

// ProcessMatch records a match result to the cache for form calculation
func (s *powerRankingService) ProcessMatch(ctx context.Context, req *domain.ProcessPRMatchRequest) error {
	// Create entry for winner
	winnerCache := &domain.PowerRankingMatchCache{
		MemberID:         req.WinnerMemberID,
		PRSystemID:       req.PRSystemID,
		MatchID:          req.MatchID,
		TournamentID:     req.TournamentID,
		IsWinner:         true,
		SetsWon:          req.WinnerSetsWon,
		SetsLost:         req.LoserSetsWon,
		OpponentMemberID: &req.LoserMemberID,
		OpponentRating:   &req.LoserRating,
		IsLAN:            req.IsLAN,
		CompletedAt:      req.CompletedAt,
	}
	if err := s.matchCacheRepo.Upsert(ctx, winnerCache); err != nil {
		return err
	}

	// Create entry for loser
	loserCache := &domain.PowerRankingMatchCache{
		MemberID:         req.LoserMemberID,
		PRSystemID:       req.PRSystemID,
		MatchID:          req.MatchID,
		TournamentID:     req.TournamentID,
		IsWinner:         false,
		SetsWon:          req.LoserSetsWon,
		SetsLost:         req.WinnerSetsWon,
		OpponentMemberID: &req.WinnerMemberID,
		OpponentRating:   &req.WinnerRating,
		IsLAN:            req.IsLAN,
		CompletedAt:      req.CompletedAt,
	}
	return s.matchCacheRepo.Upsert(ctx, loserCache)
}

// =============================================================================
// Calculation
// =============================================================================

// CalculateRankings recalculates all power rankings for a system
func (s *powerRankingService) CalculateRankings(ctx context.Context, systemID uint64) error {
	// Get the system configuration
	system, err := s.systemRepo.GetByID(ctx, systemID)
	if err != nil {
		return err
	}

	// Get all members with activity (placements or matches)
	memberIDs, err := s.rankingRepo.GetMembersWithActivity(ctx, systemID)
	if err != nil {
		return err
	}

	if len(memberIDs) == 0 {
		return nil // No members to calculate
	}

	now := time.Now()

	// Calculate for each member
	for _, memberID := range memberIDs {
		result, err := s.calculateMemberRanking(ctx, memberID, system)
		if err != nil {
			log.Printf("PR: failed to calculate ranking for member %d: %v", memberID, err)
			continue
		}

		// Upsert the ranking
		ranking := &domain.MemberPowerRanking{
			MemberID:              memberID,
			PRSystemID:            systemID,
			TotalPoints:           result.TotalPoints,
			AchievementsScore:     result.AchievementsScore,
			FormScore:             result.FormScore,
			LANScore:              result.LANScore,
			FormWins:              result.FormMetrics.Wins,
			FormSetWins:           result.FormMetrics.SetWins,
			FormSetLosses:         result.FormMetrics.SetLosses,
			FormAvgOpponentRating: result.FormMetrics.AvgOpponentRating,
			FormBestWinRating:     result.FormMetrics.BestWinRating,
			LANPlacementsCount:    result.LANPlacementsCount,
			LastCalculatedAt:      &now,
		}

		if err := s.rankingRepo.Upsert(ctx, ranking); err != nil {
			log.Printf("PR: failed to upsert ranking for member %d: %v", memberID, err)
			continue
		}
	}

	// Recalculate ranks (positions 1, 2, 3, etc.)
	return s.rankingRepo.RecalculateRanks(ctx, systemID)
}

// calculateMemberRanking calculates all components for a single member
func (s *powerRankingService) calculateMemberRanking(
	ctx context.Context,
	memberID uint64,
	system *domain.PowerRankingSystem,
) (*domain.CalculationResult, error) {
	// Calculate each component
	achievementsScore, err := s.calculateAchievementsScore(ctx, memberID, system)
	if err != nil {
		return nil, err
	}

	formScore, formMetrics, err := s.calculateFormScore(ctx, memberID, system)
	if err != nil {
		return nil, err
	}

	lanScore, lanCount, err := s.calculateLANScore(ctx, memberID, system)
	if err != nil {
		return nil, err
	}

	// Calculate weighted total
	totalPoints := (achievementsScore * float64(system.AchievementsWeight) / 100.0) +
		(formScore * float64(system.FormWeight) / 100.0) +
		(lanScore * float64(system.LANWeight) / 100.0)

	return &domain.CalculationResult{
		TotalPoints:        totalPoints,
		AchievementsScore:  achievementsScore,
		FormScore:          formScore,
		LANScore:           lanScore,
		FormMetrics:        *formMetrics,
		LANPlacementsCount: lanCount,
	}, nil
}

// calculateAchievementsScore calculates the achievements component
func (s *powerRankingService) calculateAchievementsScore(
	ctx context.Context,
	memberID uint64,
	system *domain.PowerRankingSystem,
) (float64, error) {
	// Get placements within achievement window
	since := time.Now().AddDate(0, -system.AchievementWindowMonths, 0)
	placements, err := s.placementRepo.GetByMemberSince(ctx, memberID, system.ID, since)
	if err != nil {
		return 0, err
	}

	if len(placements) == 0 {
		return 0, nil
	}

	totalScore := 0.0
	now := time.Now()

	for _, p := range placements {
		// Use the pre-calculated raw_points as base
		basePoints := float64(p.RawPoints)

		// Apply time decay
		monthsAgo := monthsSince(p.CompletedAt, now)
		decayFactor := calculateDecay(monthsAgo, system.AchievementDecayMonths)

		totalScore += basePoints * decayFactor
	}

	return totalScore, nil
}

// calculateFormScore calculates the form component from recent matches
func (s *powerRankingService) calculateFormScore(
	ctx context.Context,
	memberID uint64,
	system *domain.PowerRankingSystem,
) (float64, *domain.FormMetrics, error) {
	since := time.Now().AddDate(0, -system.FormWindowMonths, 0)
	matches, err := s.matchCacheRepo.GetByMemberSince(ctx, memberID, system.ID, since)
	if err != nil {
		return 0, &domain.FormMetrics{}, err
	}

	if len(matches) == 0 {
		return 0, &domain.FormMetrics{}, nil
	}

	metrics := &domain.FormMetrics{}
	totalOpponentRating := 0
	opponentCount := 0

	for _, m := range matches {
		if m.IsWinner {
			metrics.Wins++
			// Track best win (highest-rated opponent defeated)
			if m.OpponentRating != nil {
				if metrics.BestWinRating == nil || *m.OpponentRating > *metrics.BestWinRating {
					rating := *m.OpponentRating
					metrics.BestWinRating = &rating
				}
			}
		}
		metrics.SetWins += m.SetsWon
		metrics.SetLosses += m.SetsLost

		if m.OpponentRating != nil {
			totalOpponentRating += *m.OpponentRating
			opponentCount++
		}
	}

	// Calculate average opponent rating
	if opponentCount > 0 {
		avg := totalOpponentRating / opponentCount
		metrics.AvgOpponentRating = &avg
	}

	// Form score formula:
	// Base: (wins * 100) + (setWins - setLosses) * 10
	// Bonus: avgOpponentRating / 100 + bestWinRating / 200
	score := float64(metrics.Wins*100) + float64(metrics.SetWins-metrics.SetLosses)*10

	if metrics.AvgOpponentRating != nil {
		score += float64(*metrics.AvgOpponentRating) / 100.0
	}
	if metrics.BestWinRating != nil {
		score += float64(*metrics.BestWinRating) / 200.0
	}

	metrics.Score = score
	return score, metrics, nil
}

// calculateLANScore calculates the LAN component
func (s *powerRankingService) calculateLANScore(
	ctx context.Context,
	memberID uint64,
	system *domain.PowerRankingSystem,
) (float64, int, error) {
	// Get last N LAN placements
	placements, err := s.placementRepo.GetLANByMember(ctx, memberID, system.ID, system.LANResultsCount)
	if err != nil {
		return 0, 0, err
	}

	if len(placements) == 0 {
		return 0, 0, nil
	}

	totalScore := 0.0
	for _, p := range placements {
		// Use raw_points with LAN multiplier (1.5x)
		totalScore += float64(p.RawPoints) * 1.5
	}

	// Average across results
	avgScore := totalScore / float64(len(placements))

	return avgScore, len(placements), nil
}

// =============================================================================
// Retrieval
// =============================================================================

// GetLeaderboard returns the power ranking leaderboard, recalculating if needed
func (s *powerRankingService) GetLeaderboard(ctx context.Context, systemID uint64, limit int) ([]*domain.MemberPowerRanking, error) {
	// Recalculate rankings on every request (on-demand calculation)
	if err := s.CalculateRankings(ctx, systemID); err != nil {
		// Log error but continue with potentially stale data
		log.Printf("PR: warning - failed to recalculate rankings for system %d: %v", systemID, err)
	}

	if limit <= 0 {
		limit = 50
	}

	return s.rankingRepo.GetLeaderboard(ctx, systemID, limit)
}

// GetMemberRanking returns a specific member's power ranking
func (s *powerRankingService) GetMemberRanking(ctx context.Context, memberID, systemID uint64) (*domain.MemberPowerRanking, error) {
	return s.rankingRepo.GetByMemberAndSystem(ctx, memberID, systemID)
}

// GetMemberPlacements returns all placements for a member in a system
func (s *powerRankingService) GetMemberPlacements(ctx context.Context, memberID, systemID uint64) ([]*domain.TournamentPlacement, error) {
	return s.placementRepo.GetByMember(ctx, memberID, systemID)
}

// =============================================================================
// Reversion
// =============================================================================

// RevertTournamentPlacements removes all placements and match cache for a tournament
func (s *powerRankingService) RevertTournamentPlacements(ctx context.Context, tournamentID uint64) error {
	// Delete placements
	if err := s.placementRepo.DeleteByTournament(ctx, tournamentID); err != nil {
		return err
	}
	// Delete match cache
	return s.matchCacheRepo.DeleteByTournament(ctx, tournamentID)
}

// =============================================================================
// Helper Functions
// =============================================================================

// calculatePlacementPoints computes raw points for a placement
// Formula: BaseClassPoints * PlacementMultiplier * SizeBonus * PrizeBonus
func (s *powerRankingService) calculatePlacementPoints(
	placement int,
	participantCount int,
	class domain.TournamentClass,
	prizePoolUSD *float64,
) int {
	// Base points by tournament class
	basePoints := classBasePoints[class]
	if basePoints == 0 {
		basePoints = 200 // Default to online if unknown
	}

	// Placement multiplier
	var multiplier float64
	if placement >= 1 && placement <= len(placementMultipliers) {
		multiplier = placementMultipliers[placement-1]
	} else if placement > len(placementMultipliers) {
		// For placements beyond 8th, use diminishing formula
		// Starts at 0.15 for 9th, decreases further
		multiplier = 0.15 / math.Max(1, float64(placement-7))
	} else {
		multiplier = 0 // Invalid placement
	}

	// Size bonus: log2(participantCount) / 4
	// Minimum participant count of 2 to avoid log issues
	sizeBonus := 1.0
	if participantCount >= 2 {
		sizeBonus = 1.0 + math.Log2(float64(participantCount))/4.0
	}

	// Prize pool bonus (optional)
	prizeBonus := 1.0
	if prizePoolUSD != nil && *prizePoolUSD > 0 {
		prizeBonus = 1.0 + math.Log10(*prizePoolUSD)/10.0
	}

	return int(float64(basePoints) * multiplier * sizeBonus * prizeBonus)
}

// monthsSince calculates the number of months between two times
func monthsSince(t time.Time, now time.Time) float64 {
	duration := now.Sub(t)
	return duration.Hours() / (24 * 30) // Approximate months
}

// calculateDecay computes exponential decay factor
// At decayMonths, value is ~50%
// Formula: 0.5^(monthsAgo / decayMonths)
func calculateDecay(monthsAgo float64, decayMonths int) float64 {
	if decayMonths <= 0 {
		return 1.0 // No decay
	}
	return math.Pow(0.5, monthsAgo/float64(decayMonths))
}
