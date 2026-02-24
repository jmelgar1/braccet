package service

import (
	"context"
	"fmt"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/engine"
	"github.com/braccet/bracket/internal/repository"
)

type SwissService interface {
	// InitializeSwiss creates standings, config, and round 1 matches
	InitializeSwiss(ctx context.Context, tournamentID uint64, participants []domain.Participant, roundOverride *int) (*domain.SwissBracketState, error)

	// GetState returns full Swiss bracket state
	GetState(ctx context.Context, tournamentID uint64) (*domain.SwissBracketState, error)

	// GetStandings returns current standings sorted by rank
	GetStandings(ctx context.Context, tournamentID uint64) ([]*domain.SwissStanding, error)

	// AdvanceRound generates next round pairings (called after round completes)
	AdvanceRound(ctx context.Context, tournamentID uint64) ([]*domain.Match, error)

	// CheckAndAdvanceRound checks if current round is complete and advances if so
	// Returns the new matches if advanced, nil if not ready to advance
	CheckAndAdvanceRound(ctx context.Context, tournamentID uint64) ([]*domain.Match, error)

	// CheckAndAdvanceRoundWithTiebreakers checks if current round is complete and handles tiebreakers
	// tiebreakerEnabled controls whether tiebreaker matches are created for ties
	// Returns the new matches (regular or tiebreaker) if advanced, nil if not ready
	CheckAndAdvanceRoundWithTiebreakers(ctx context.Context, tournamentID uint64, tiebreakerEnabled bool) ([]*domain.Match, error)

	// Delete removes all Swiss data for a tournament
	Delete(ctx context.Context, tournamentID uint64) error
}

type swissService struct {
	swissRepo      repository.SwissRepository
	matchRepo      repository.MatchRepository
	stageRepo      repository.StageRepository
	tiebreakerRepo repository.TiebreakerRepository
}

func NewSwissService(swissRepo repository.SwissRepository, matchRepo repository.MatchRepository, stageRepo repository.StageRepository) SwissService {
	return &swissService{
		swissRepo: swissRepo,
		matchRepo: matchRepo,
		stageRepo: stageRepo,
	}
}

// NewSwissServiceWithTiebreakers creates a Swiss service with tiebreaker support
func NewSwissServiceWithTiebreakers(
	swissRepo repository.SwissRepository,
	matchRepo repository.MatchRepository,
	stageRepo repository.StageRepository,
	tiebreakerRepo repository.TiebreakerRepository,
) SwissService {
	return &swissService{
		swissRepo:      swissRepo,
		matchRepo:      matchRepo,
		stageRepo:      stageRepo,
		tiebreakerRepo: tiebreakerRepo,
	}
}

// InitializeSwiss creates a Swiss bracket with initial standings and round 1 matches.
func (s *swissService) InitializeSwiss(ctx context.Context, tournamentID uint64, participants []domain.Participant, roundOverride *int) (*domain.SwissBracketState, error) {
	if len(participants) < 2 {
		return nil, fmt.Errorf("need at least 2 participants, got %d", len(participants))
	}

	// Calculate rounds
	totalRounds := engine.CalculateSwissRounds(len(participants))
	if roundOverride != nil && *roundOverride > 0 {
		totalRounds = *roundOverride
	}

	// Create config
	config := &domain.SwissConfig{
		TournamentID: tournamentID,
		TotalRounds:  totalRounds,
		CurrentRound: 1,
		IsComplete:   false,
	}
	if err := s.swissRepo.CreateConfig(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create swiss config: %w", err)
	}

	// Create standings for each participant
	standings := make([]*domain.SwissStanding, len(participants))
	for i, p := range participants {
		standings[i] = &domain.SwissStanding{
			TournamentID:       tournamentID,
			ParticipantID:      p.ID,
			ParticipantName:    p.Name,
			ParticipantIconURL: stringPtr(p.IconURL),
			Seed:               p.Seed,
			Wins:               0,
			Losses:             0,
			GameWins:           0,
			GameLosses:         0,
			OpponentWins:       0,
			HasBye:             false,
		}
	}
	if err := s.swissRepo.CreateStandings(ctx, standings); err != nil {
		return nil, fmt.Errorf("failed to create swiss standings: %w", err)
	}

	// Generate round 1 pairings
	pairings := engine.GenerateRound1Pairings(standings)

	// Create pairing history for non-BYE matches
	var historyRecords []*domain.SwissPairingHistory
	for _, p := range pairings {
		if !p.IsBye {
			historyRecords = append(historyRecords, &domain.SwissPairingHistory{
				TournamentID:   tournamentID,
				Participant1ID: p.Participant1ID,
				Participant2ID: p.Participant2ID,
				Round:          1,
			})
		}
	}
	if err := s.swissRepo.CreatePairings(ctx, historyRecords); err != nil {
		return nil, fmt.Errorf("failed to create pairing history: %w", err)
	}

	// Convert pairings to matches
	matches := engine.PairingsToMatches(tournamentID, 1, pairings)

	// Save matches
	if err := s.matchRepo.CreateBatch(ctx, matches); err != nil {
		return nil, fmt.Errorf("failed to create matches: %w", err)
	}

	// Update standings for BYE matches
	for _, p := range pairings {
		if p.IsBye {
			standing, err := s.swissRepo.GetStandingByParticipant(ctx, tournamentID, p.Participant1ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get standing for BYE participant: %w", err)
			}
			standing.Wins = 1
			standing.HasBye = true
			if err := s.swissRepo.UpdateStanding(ctx, standing); err != nil {
				return nil, fmt.Errorf("failed to update BYE standing: %w", err)
			}
		}
	}

	// Create stage for round 1
	if s.stageRepo != nil {
		if err := s.stageRepo.CreateDefaultStages(ctx, tournamentID, domain.BracketSwiss, totalRounds, false); err != nil {
			return nil, fmt.Errorf("failed to create swiss stages: %w", err)
		}
	}

	// Return the state
	return s.GetState(ctx, tournamentID)
}

// GetState returns the full Swiss bracket state.
func (s *swissService) GetState(ctx context.Context, tournamentID uint64) (*domain.SwissBracketState, error) {
	config, err := s.swissRepo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	standings, err := s.swissRepo.GetStandings(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	matches, err := s.matchRepo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Get stages for best_of settings
	var stages []*domain.BracketStage
	if s.stageRepo != nil {
		stages, err = s.stageRepo.GetByTournament(ctx, tournamentID)
		if err != nil {
			return nil, err
		}
	}

	// Get tiebreakers if they exist
	var tiebreakers []*domain.TiebreakerBracket
	if s.tiebreakerRepo != nil && config.HasTiebreakers {
		tiebreakers, err = s.tiebreakerRepo.GetByTournament(ctx, tournamentID)
		if err != nil {
			return nil, err
		}
	}

	state := engine.GetSwissBracketStateWithStages(tournamentID, config, standings, matches, stages)
	state.Tiebreakers = tiebreakers
	return state, nil
}

// GetStandings returns the current standings.
func (s *swissService) GetStandings(ctx context.Context, tournamentID uint64) ([]*domain.SwissStanding, error) {
	return s.swissRepo.GetStandings(ctx, tournamentID)
}

// AdvanceRound generates the next round of matches.
func (s *swissService) AdvanceRound(ctx context.Context, tournamentID uint64) ([]*domain.Match, error) {
	config, err := s.swissRepo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if config.IsComplete {
		return nil, fmt.Errorf("tournament is already complete")
	}

	if config.CurrentRound >= config.TotalRounds {
		// Mark as complete instead of generating new round
		config.IsComplete = true
		if err := s.swissRepo.UpdateConfig(ctx, config); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Update opponent wins (Buchholz tiebreaker) before pairing
	if err := s.swissRepo.UpdateOpponentWins(ctx, tournamentID); err != nil {
		return nil, fmt.Errorf("failed to update opponent wins: %w", err)
	}

	// Get current standings
	standings, err := s.swissRepo.GetStandings(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Get pairing history
	history, err := s.swissRepo.GetPairingHistory(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	playedPairs := engine.BuildPlayedPairsMap(history)

	// Generate pairings for next round
	nextRound := config.CurrentRound + 1
	pairings, err := engine.GenerateSwissPairings(standings, playedPairs, nextRound)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pairings: %w", err)
	}

	// Record pairing history
	var historyRecords []*domain.SwissPairingHistory
	for _, p := range pairings {
		if !p.IsBye {
			historyRecords = append(historyRecords, &domain.SwissPairingHistory{
				TournamentID:   tournamentID,
				Participant1ID: p.Participant1ID,
				Participant2ID: p.Participant2ID,
				Round:          nextRound,
			})
		}
	}
	if err := s.swissRepo.CreatePairings(ctx, historyRecords); err != nil {
		return nil, fmt.Errorf("failed to create pairing history: %w", err)
	}

	// Create matches
	matches := engine.PairingsToMatches(tournamentID, nextRound, pairings)
	if err := s.matchRepo.CreateBatch(ctx, matches); err != nil {
		return nil, fmt.Errorf("failed to create matches: %w", err)
	}

	// Update standings for BYE matches
	for _, p := range pairings {
		if p.IsBye {
			standing, err := s.swissRepo.GetStandingByParticipant(ctx, tournamentID, p.Participant1ID)
			if err != nil {
				return nil, fmt.Errorf("failed to get standing for BYE participant: %w", err)
			}
			standing.Wins++
			standing.HasBye = true
			if err := s.swissRepo.UpdateStanding(ctx, standing); err != nil {
				return nil, fmt.Errorf("failed to update BYE standing: %w", err)
			}
		}
	}

	// Update config
	config.CurrentRound = nextRound
	if err := s.swissRepo.UpdateConfig(ctx, config); err != nil {
		return nil, err
	}

	return matches, nil
}

// CheckAndAdvanceRound checks if all matches in the current round are complete
// and advances to the next round if so.
func (s *swissService) CheckAndAdvanceRound(ctx context.Context, tournamentID uint64) ([]*domain.Match, error) {
	config, err := s.swissRepo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if config.IsComplete {
		return nil, nil // Already complete
	}

	// Get all matches for the current round
	allMatches, err := s.matchRepo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Check if all matches in current round are complete
	// Important: We must verify that matches EXIST for the round before considering it complete
	roundMatchCount := 0
	currentRoundComplete := true
	for _, m := range allMatches {
		if m.BracketType == domain.BracketSwiss && m.Round == config.CurrentRound {
			roundMatchCount++
			if m.Status != domain.MatchCompleted {
				currentRoundComplete = false
				break
			}
		}
	}

	// If no matches exist for this round yet, don't advance (prevents race condition)
	if roundMatchCount == 0 || !currentRoundComplete {
		return nil, nil // Not ready to advance
	}

	// If we're at the final round, mark as complete
	if config.CurrentRound >= config.TotalRounds {
		config.IsComplete = true
		if err := s.swissRepo.UpdateConfig(ctx, config); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Advance to next round
	return s.AdvanceRound(ctx, tournamentID)
}

// CheckAndAdvanceRoundWithTiebreakers checks if all matches in the current round are complete
// and advances to the next round if so. At the final round, checks for ties and creates
// tiebreaker matches if enabled.
func (s *swissService) CheckAndAdvanceRoundWithTiebreakers(ctx context.Context, tournamentID uint64, tiebreakerEnabled bool) ([]*domain.Match, error) {
	config, err := s.swissRepo.GetConfig(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if config.IsComplete {
		return nil, nil // Already complete
	}

	// If tiebreakers exist but aren't complete, check tiebreaker matches
	if config.HasTiebreakers && !config.TiebreakersComplete {
		// Check if all tiebreaker matches are complete
		allMatches, err := s.matchRepo.GetByTournament(ctx, tournamentID)
		if err != nil {
			return nil, err
		}

		tiebreakerComplete := true
		for _, m := range allMatches {
			if m.BracketType == domain.BracketTiebreaker && m.Status != domain.MatchCompleted {
				tiebreakerComplete = false
				break
			}
		}

		if tiebreakerComplete {
			// All tiebreakers done - mark Swiss as complete
			config.TiebreakersComplete = true
			config.IsComplete = true
			if err := s.swissRepo.UpdateConfig(ctx, config); err != nil {
				return nil, err
			}
		}

		return nil, nil // Waiting for tiebreakers
	}

	// Get all matches for the current round
	allMatches, err := s.matchRepo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Check if all Swiss matches in current round are complete
	roundMatchCount := 0
	currentRoundComplete := true
	for _, m := range allMatches {
		if m.BracketType == domain.BracketSwiss && m.Round == config.CurrentRound {
			roundMatchCount++
			if m.Status != domain.MatchCompleted {
				currentRoundComplete = false
				break
			}
		}
	}

	// If no matches exist for this round yet, don't advance
	if roundMatchCount == 0 || !currentRoundComplete {
		return nil, nil // Not ready to advance
	}

	// If we're at the final round, check for tiebreakers
	if config.CurrentRound >= config.TotalRounds {
		// Update Buchholz scores before checking for ties
		if err := s.swissRepo.UpdateOpponentWins(ctx, tournamentID); err != nil {
			return nil, fmt.Errorf("failed to update opponent wins: %w", err)
		}

		// Check for ties and create tiebreaker matches if enabled
		if tiebreakerEnabled && s.tiebreakerRepo != nil {
			tiebreakerSvc := NewTiebreakerService(s.tiebreakerRepo, s.swissRepo, s.matchRepo)
			tiebreakerMatches, err := tiebreakerSvc.CheckAndCreateTiebreakers(ctx, tournamentID, tiebreakerEnabled)
			if err != nil {
				return nil, fmt.Errorf("failed to create tiebreakers: %w", err)
			}

			if len(tiebreakerMatches) > 0 {
				// Tiebreakers created - don't mark Swiss complete yet
				return tiebreakerMatches, nil
			}
		}

		// No tiebreakers needed - mark as complete
		config.IsComplete = true
		if err := s.swissRepo.UpdateConfig(ctx, config); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Advance to next round
	return s.AdvanceRound(ctx, tournamentID)
}

// Delete removes all Swiss data for a tournament.
func (s *swissService) Delete(ctx context.Context, tournamentID uint64) error {
	return s.swissRepo.DeleteByTournament(ctx, tournamentID)
}

// stringPtr returns a pointer to the string, or nil if empty
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
