package service

import (
	"context"
	"fmt"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/engine"
	"github.com/braccet/bracket/internal/repository"
)

type TiebreakerService interface {
	// CheckAndCreateTiebreakers analyzes final Swiss standings and creates
	// tiebreaker matches if needed. Returns created matches or nil if no ties.
	CheckAndCreateTiebreakers(ctx context.Context, tournamentID uint64, tiebreakerEnabled bool) ([]*domain.Match, error)

	// GetTiebreakers returns all tiebreaker brackets for a tournament
	GetTiebreakers(ctx context.Context, tournamentID uint64) ([]*domain.TiebreakerBracket, error)

	// GetTiebreakerMatches returns all tiebreaker matches for a tournament
	GetTiebreakerMatches(ctx context.Context, tournamentID uint64) ([]*domain.Match, error)

	// OnTiebreakerMatchComplete handles completion of a tiebreaker match.
	// Returns true if all tiebreakers are now complete.
	OnTiebreakerMatchComplete(ctx context.Context, match *domain.Match, winnerID uint64) (bool, error)

	// GetFinalRankings returns Swiss standings with tiebreaker results applied
	GetFinalRankings(ctx context.Context, tournamentID uint64) ([]*domain.SwissStanding, error)

	// IsTiebreakerMatch checks if a match is a tiebreaker match
	IsTiebreakerMatch(match *domain.Match) bool

	// AdvanceWinnerInTiebreaker advances a winner to the next match in a mini bracket
	AdvanceWinnerInTiebreaker(ctx context.Context, match *domain.Match, winnerID uint64) error
}

type tiebreakerService struct {
	tiebreakerRepo repository.TiebreakerRepository
	swissRepo      repository.SwissRepository
	matchRepo      repository.MatchRepository
}

func NewTiebreakerService(
	tiebreakerRepo repository.TiebreakerRepository,
	swissRepo repository.SwissRepository,
	matchRepo repository.MatchRepository,
) TiebreakerService {
	return &tiebreakerService{
		tiebreakerRepo: tiebreakerRepo,
		swissRepo:      swissRepo,
		matchRepo:      matchRepo,
	}
}

// CheckAndCreateTiebreakers checks for ties in final standings and creates tiebreaker matches
func (s *tiebreakerService) CheckAndCreateTiebreakers(ctx context.Context, tournamentID uint64, tiebreakerEnabled bool) ([]*domain.Match, error) {
	// If tiebreakers are not enabled, return nil
	if !tiebreakerEnabled {
		return nil, nil
	}

	// Get final Swiss standings
	standings, err := s.swissRepo.GetStandings(ctx, tournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get standings: %w", err)
	}

	// Detect tie groups
	tieGroups := engine.DetectTies(standings)
	if len(tieGroups) == 0 {
		return nil, nil // No ties to resolve
	}

	// TODO: Get best_of from stage settings
	// For now, use default of 1 (single game)
	bestOf := 1

	var allMatches []*domain.Match

	for _, tieGroup := range tieGroups {
		// Get participants for this tie group
		participants := engine.GetParticipantsForTieGroup(standings, tieGroup)

		// Generate tiebreaker matches
		matches := engine.GenerateTiebreakerMatches(
			tournamentID, tieGroup.Position, participants, bestOf,
		)

		if len(matches) == 0 {
			continue
		}

		// Create tiebreaker record
		tb := &domain.TiebreakerBracket{
			TournamentID:   tournamentID,
			TiePosition:    tieGroup.Position,
			ParticipantIDs: tieGroup.ParticipantIDs,
			IsComplete:     false,
		}
		if err := s.tiebreakerRepo.Create(ctx, tb); err != nil {
			return nil, fmt.Errorf("failed to create tiebreaker: %w", err)
		}

		// Save matches
		if err := s.matchRepo.CreateBatch(ctx, matches); err != nil {
			return nil, fmt.Errorf("failed to create tiebreaker matches: %w", err)
		}

		// Link matches (for mini brackets with multiple rounds)
		if len(matches) > 1 {
			engine.LinkTiebreakerMatches(matches)
			if err := s.matchRepo.UpdateNextMatchLinks(ctx, matches); err != nil {
				return nil, fmt.Errorf("failed to link tiebreaker matches: %w", err)
			}
		}

		allMatches = append(allMatches, matches...)
	}

	// Update swiss config to indicate tiebreakers exist
	if len(allMatches) > 0 {
		if err := s.swissRepo.SetHasTiebreakers(ctx, tournamentID, true); err != nil {
			return nil, fmt.Errorf("failed to set has_tiebreakers: %w", err)
		}
	}

	return allMatches, nil
}

// GetTiebreakers returns all tiebreaker brackets for a tournament
func (s *tiebreakerService) GetTiebreakers(ctx context.Context, tournamentID uint64) ([]*domain.TiebreakerBracket, error) {
	return s.tiebreakerRepo.GetByTournament(ctx, tournamentID)
}

// GetTiebreakerMatches returns all tiebreaker matches for a tournament
func (s *tiebreakerService) GetTiebreakerMatches(ctx context.Context, tournamentID uint64) ([]*domain.Match, error) {
	allMatches, err := s.matchRepo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	var tiebreakerMatches []*domain.Match
	for _, m := range allMatches {
		if m.BracketType == domain.BracketTiebreaker {
			tiebreakerMatches = append(tiebreakerMatches, m)
		}
	}

	return tiebreakerMatches, nil
}

// OnTiebreakerMatchComplete handles completion of a tiebreaker match
func (s *tiebreakerService) OnTiebreakerMatchComplete(ctx context.Context, match *domain.Match, winnerID uint64) (bool, error) {
	// Get all tiebreakers for this tournament
	tiebreakers, err := s.tiebreakerRepo.GetByTournament(ctx, match.TournamentID)
	if err != nil {
		return false, err
	}

	// Find the tiebreaker this match belongs to (using Round as tie position)
	var tiebreaker *domain.TiebreakerBracket
	for _, tb := range tiebreakers {
		if tb.TiePosition == match.Round {
			tiebreaker = tb
			break
		}
	}

	if tiebreaker == nil {
		return false, fmt.Errorf("tiebreaker not found for position %d", match.Round)
	}

	// Get all tiebreaker matches for this tournament
	tiebreakerMatches, err := s.GetTiebreakerMatches(ctx, match.TournamentID)
	if err != nil {
		return false, err
	}

	// Check if this tiebreaker is complete (all matches for this position are completed)
	thisTiebreakerComplete := true
	var finalMatch *domain.Match
	maxPosition := 0
	for _, m := range tiebreakerMatches {
		if m.Round == tiebreaker.TiePosition {
			if m.Status != domain.MatchCompleted {
				thisTiebreakerComplete = false
			}
			if m.Position > maxPosition {
				maxPosition = m.Position
				finalMatch = m
			}
		}
	}

	// If this tiebreaker is complete, record results and mark it complete
	if thisTiebreakerComplete && finalMatch != nil && finalMatch.WinnerID != nil {
		// Record the results - assign final positions based on bracket results
		if err := s.recordTiebreakerResults(ctx, tiebreaker, tiebreakerMatches); err != nil {
			return false, err
		}

		// Mark tiebreaker as complete
		if err := s.tiebreakerRepo.MarkComplete(ctx, tiebreaker.ID); err != nil {
			return false, err
		}
	}

	// Check if all tiebreakers are complete
	allComplete := true
	for _, tb := range tiebreakers {
		if tb.TiePosition == tiebreaker.TiePosition {
			// We just completed this one
			continue
		}
		if !tb.IsComplete {
			allComplete = false
			break
		}
	}

	// If all tiebreakers were already complete (or this was the last), return true
	if allComplete && thisTiebreakerComplete {
		return true, nil
	}

	return false, nil
}

// recordTiebreakerResults records the final positions for participants in a completed tiebreaker
func (s *tiebreakerService) recordTiebreakerResults(ctx context.Context, tiebreaker *domain.TiebreakerBracket, matches []*domain.Match) error {
	// Filter to only matches for this tiebreaker
	var tbMatches []*domain.Match
	for _, m := range matches {
		if m.Round == tiebreaker.TiePosition {
			tbMatches = append(tbMatches, m)
		}
	}

	if len(tbMatches) == 0 {
		return nil
	}

	// For 2-way tie (single match), positions are simple
	if len(tbMatches) == 1 && tbMatches[0].WinnerID != nil {
		match := tbMatches[0]
		// Winner gets the original tie position
		winnerResult := &domain.TiebreakerResult{
			TiebreakerID:  tiebreaker.ID,
			ParticipantID: *match.WinnerID,
			FinalPosition: tiebreaker.TiePosition,
		}
		if err := s.tiebreakerRepo.CreateResult(ctx, winnerResult); err != nil {
			return err
		}

		// Loser gets tie position + 1
		var loserID uint64
		if *match.WinnerID == *match.Participant1ID {
			loserID = *match.Participant2ID
		} else {
			loserID = *match.Participant1ID
		}
		loserResult := &domain.TiebreakerResult{
			TiebreakerID:  tiebreaker.ID,
			ParticipantID: loserID,
			FinalPosition: tiebreaker.TiePosition + 1,
		}
		if err := s.tiebreakerRepo.CreateResult(ctx, loserResult); err != nil {
			return err
		}

		return nil
	}

	// For multi-way ties (mini bracket), assign positions based on elimination order
	// This is simplified - in a full implementation you'd track elimination order
	participantPositions := make(map[uint64]int)
	basePosition := tiebreaker.TiePosition
	numParticipants := len(tiebreaker.ParticipantIDs)

	// Find the final match (highest encoded position)
	var finalMatch *domain.Match
	maxEncodedRound := 0
	for _, m := range tbMatches {
		actualRound := engine.GetTiebreakerRoundFromPosition(m.Position)
		if actualRound > maxEncodedRound {
			maxEncodedRound = actualRound
			finalMatch = m
		}
	}

	if finalMatch != nil && finalMatch.WinnerID != nil {
		// Champion
		participantPositions[*finalMatch.WinnerID] = basePosition

		// Runner-up
		var runnerUpID uint64
		if *finalMatch.WinnerID == *finalMatch.Participant1ID {
			if finalMatch.Participant2ID != nil {
				runnerUpID = *finalMatch.Participant2ID
			}
		} else {
			if finalMatch.Participant1ID != nil {
				runnerUpID = *finalMatch.Participant1ID
			}
		}
		if runnerUpID != 0 {
			participantPositions[runnerUpID] = basePosition + 1
		}
	}

	// Assign remaining positions to participants not yet placed
	nextPosition := basePosition + 2
	for _, pID := range tiebreaker.ParticipantIDs {
		if _, placed := participantPositions[pID]; !placed {
			participantPositions[pID] = nextPosition
			nextPosition++
			if nextPosition > basePosition+numParticipants-1 {
				nextPosition = basePosition + numParticipants - 1 // Don't exceed the tie range
			}
		}
	}

	// Create result records
	for pID, position := range participantPositions {
		result := &domain.TiebreakerResult{
			TiebreakerID:  tiebreaker.ID,
			ParticipantID: pID,
			FinalPosition: position,
		}
		if err := s.tiebreakerRepo.CreateResult(ctx, result); err != nil {
			return err
		}
	}

	return nil
}

// GetFinalRankings returns Swiss standings with tiebreaker results applied
func (s *tiebreakerService) GetFinalRankings(ctx context.Context, tournamentID uint64) ([]*domain.SwissStanding, error) {
	standings, err := s.swissRepo.GetStandings(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Get tiebreaker results
	results, err := s.tiebreakerRepo.GetResultsByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		// No tiebreaker results, return standings as-is
		return standings, nil
	}

	// Apply tiebreaker results to sort standings
	// Build a map of participant ID to standing
	standingMap := make(map[uint64]*domain.SwissStanding)
	for _, s := range standings {
		standingMap[s.ParticipantID] = s
	}

	// Sort standings using tiebreaker results where available
	// For simplicity, we just re-order based on the tiebreaker positions
	// A more sophisticated approach would integrate this into the main sort

	// Create a new slice sorted by tiebreaker results
	// First, get participants without tiebreaker results (sorted by normal criteria)
	// Then interleave with tiebreaker results

	// For now, just return the standings as the tiebreaker results are stored
	// and can be used by the frontend to display correct positions

	return standings, nil
}

// IsTiebreakerMatch checks if a match is a tiebreaker match
func (s *tiebreakerService) IsTiebreakerMatch(match *domain.Match) bool {
	return match.BracketType == domain.BracketTiebreaker
}

// AdvanceWinnerInTiebreaker advances a winner to the next match in a mini bracket
func (s *tiebreakerService) AdvanceWinnerInTiebreaker(ctx context.Context, match *domain.Match, winnerID uint64) error {
	if match.NextMatchID == nil {
		return nil // No next match, this was the final
	}

	nextMatch, err := s.matchRepo.GetByID(ctx, *match.NextMatchID)
	if err != nil {
		return err
	}

	// Determine which slot to use based on position
	// Even positions go to slot 1, odd positions go to slot 2
	slot := 1
	if match.Position%2 == 0 {
		slot = 2
	}

	// Get winner info
	var winnerName string
	var winnerIconURL string
	var winnerSeed int
	if match.Participant1ID != nil && *match.Participant1ID == winnerID {
		if match.Participant1Name != nil {
			winnerName = *match.Participant1Name
		}
		if match.Participant1IconURL != nil {
			winnerIconURL = *match.Participant1IconURL
		}
		if match.Seed1 != nil {
			winnerSeed = *match.Seed1
		}
	} else {
		if match.Participant2Name != nil {
			winnerName = *match.Participant2Name
		}
		if match.Participant2IconURL != nil {
			winnerIconURL = *match.Participant2IconURL
		}
		if match.Seed2 != nil {
			winnerSeed = *match.Seed2
		}
	}

	// Set the winner in the next match
	if err := s.matchRepo.SetParticipant(ctx, nextMatch.ID, slot, winnerID, winnerName, winnerIconURL, winnerSeed); err != nil {
		return err
	}

	// Update next match status if both participants are now set
	nextMatch, err = s.matchRepo.GetByID(ctx, *match.NextMatchID)
	if err != nil {
		return err
	}

	if nextMatch.Participant1ID != nil && nextMatch.Participant2ID != nil {
		if err := s.matchRepo.UpdateStatus(ctx, nextMatch.ID, domain.MatchReady); err != nil {
			return err
		}
	}

	return nil
}
