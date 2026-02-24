package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/braccet/bracket/internal/client"
	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/repository"
)

var (
	ErrMatchNotReady        = errors.New("match is not ready for result reporting")
	ErrInvalidWinner        = errors.New("winner must be a participant in the match")
	ErrMatchAlreadyComplete = errors.New("match has already been completed")
	ErrNoSets               = errors.New("at least one set is required")
	ErrSetsTied             = errors.New("sets are tied - there must be a clear winner")
	ErrMatchNotCompleted    = errors.New("match is not completed")
)

type MatchService interface {
	ReportResult(ctx context.Context, matchID uint64, result domain.MatchResult) error
	EditResult(ctx context.Context, matchID uint64, result domain.MatchResult) (*EditResultResponse, error)
	StartMatch(ctx context.Context, matchID uint64) error
	GetBracketState(ctx context.Context, tournamentID uint64) (*BracketState, error)
	ReopenMatch(ctx context.Context, matchID uint64) ([]*domain.Match, error)
	GetEliminationStandings(ctx context.Context, tournamentID uint64) ([]*domain.EliminationStanding, error)
}

type EditResultResponse struct {
	Match          *domain.Match
	CascadeMatches []*domain.Match
	WinnerChanged  bool
}

type BracketState struct {
	TournamentID uint64
	TotalRounds  int
	CurrentRound int
	Matches      []*domain.Match
	IsComplete   bool
	ChampionID   *uint64
}

type matchService struct {
	repo              repository.MatchRepository
	setRepo           repository.SetRepository
	swissRepo         repository.SwissRepository
	tiebreakerRepo    repository.TiebreakerRepository
	tournamentClient  client.TournamentClient
	communityClient   client.CommunityClient
}

func NewMatchService(
	repo repository.MatchRepository,
	setRepo repository.SetRepository,
	tournamentClient client.TournamentClient,
	communityClient client.CommunityClient,
) MatchService {
	return &matchService{
		repo:             repo,
		setRepo:          setRepo,
		tournamentClient: tournamentClient,
		communityClient:  communityClient,
	}
}

// NewMatchServiceWithSwiss creates a match service with Swiss support
func NewMatchServiceWithSwiss(
	repo repository.MatchRepository,
	setRepo repository.SetRepository,
	swissRepo repository.SwissRepository,
	tournamentClient client.TournamentClient,
	communityClient client.CommunityClient,
) MatchService {
	return &matchService{
		repo:             repo,
		setRepo:          setRepo,
		swissRepo:        swissRepo,
		tournamentClient: tournamentClient,
		communityClient:  communityClient,
	}
}

// NewMatchServiceWithTiebreakers creates a match service with Swiss and tiebreaker support
func NewMatchServiceWithTiebreakers(
	repo repository.MatchRepository,
	setRepo repository.SetRepository,
	swissRepo repository.SwissRepository,
	tiebreakerRepo repository.TiebreakerRepository,
	tournamentClient client.TournamentClient,
	communityClient client.CommunityClient,
) MatchService {
	return &matchService{
		repo:             repo,
		setRepo:          setRepo,
		swissRepo:        swissRepo,
		tiebreakerRepo:   tiebreakerRepo,
		tournamentClient: tournamentClient,
		communityClient:  communityClient,
	}
}

// ReportResult records the result of a match and advances the winner.
// Winner is computed from the sets (whoever wins the most sets).
func (s *matchService) ReportResult(ctx context.Context, matchID uint64, result domain.MatchResult) error {
	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return err
	}

	// Validate match can receive a result
	if match.Status == domain.MatchCompleted {
		return ErrMatchAlreadyComplete
	}
	if match.Status == domain.MatchPending {
		return ErrMatchNotReady
	}

	// Validate sets
	if len(result.Sets) == 0 {
		return ErrNoSets
	}

	// Compute winner from sets
	winnerID, err := computeWinnerFromSets(match, result.Sets)
	if err != nil {
		return err
	}

	// Save the sets
	if err := s.setRepo.CreateBatch(ctx, matchID, result.Sets); err != nil {
		return err
	}

	// Update the match result with computed winner
	if err := s.repo.UpdateResult(ctx, matchID, winnerID); err != nil {
		return err
	}

	// Advance winner to next match if there is one
	if match.NextMatchID != nil {
		if err := s.advanceWinner(ctx, match, winnerID); err != nil {
			return err
		}
	}

	// For double elimination: advance loser to losers bracket
	if match.BracketType == domain.BracketWinners && match.LoserMatchID != nil {
		loserID := s.getLoserID(match, winnerID)
		if loserID != 0 {
			if err := s.advanceLoser(ctx, match, loserID); err != nil {
				return err
			}
		}
	}

	// For Swiss: update standings and check for round advancement
	if match.BracketType == domain.BracketSwiss && s.swissRepo != nil {
		if err := s.handleSwissMatchComplete(ctx, match, result.Sets, winnerID); err != nil {
			return err
		}
	}

	// For Tiebreakers: advance winner in mini bracket and check for completion
	if match.BracketType == domain.BracketTiebreaker && s.tiebreakerRepo != nil {
		if err := s.handleTiebreakerMatchComplete(ctx, match, winnerID); err != nil {
			return err
		}
	}

	// Process ELO update asynchronously (don't fail the match if ELO fails)
	go s.processEloUpdate(context.Background(), match, winnerID)

	return nil
}

// EditResult allows editing the result of a completed match.
// If the winner changes and has advanced to future matches, those matches are cascade-reset.
func (s *matchService) EditResult(ctx context.Context, matchID uint64, result domain.MatchResult) (*EditResultResponse, error) {
	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	// Can only edit completed matches
	if match.Status != domain.MatchCompleted {
		return nil, ErrMatchNotCompleted
	}

	// Validate sets
	if len(result.Sets) == 0 {
		return nil, ErrNoSets
	}

	// Compute new winner from sets
	newWinnerID, err := computeWinnerFromSets(match, result.Sets)
	if err != nil {
		return nil, err
	}

	response := &EditResultResponse{
		WinnerChanged:  false,
		CascadeMatches: []*domain.Match{},
	}

	// Check if winner changed
	winnerChanged := match.WinnerID == nil || *match.WinnerID != newWinnerID

	if winnerChanged {
		response.WinnerChanged = true

		// If old winner was advanced to next match, we need to cascade
		if match.NextMatchID != nil && match.WinnerID != nil {
			nextMatch, err := s.repo.GetByID(ctx, *match.NextMatchID)
			if err != nil {
				return nil, err
			}

			// Determine which slot the old winner occupied
			slot := 1
			if match.Position%2 == 0 {
				slot = 2
			}

			// Check if old winner was actually placed in next match
			oldWinnerInNextMatch := false
			if slot == 1 && nextMatch.Participant1ID != nil && *nextMatch.Participant1ID == *match.WinnerID {
				oldWinnerInNextMatch = true
			} else if slot == 2 && nextMatch.Participant2ID != nil && *nextMatch.Participant2ID == *match.WinnerID {
				oldWinnerInNextMatch = true
			}

			if oldWinnerInNextMatch {
				// If next match was completed, recursively reopen it first
				if nextMatch.Status == domain.MatchCompleted {
					if err := s.reopenMatchCascade(ctx, nextMatch, &response.CascadeMatches); err != nil {
						return nil, err
					}
				}

				// Clear the old winner's slot in next match
				if err := s.repo.ClearParticipant(ctx, *match.NextMatchID, slot); err != nil {
					return nil, err
				}

				// Update next match status
				if err := s.updateMatchStatusAfterClear(ctx, *match.NextMatchID); err != nil {
					return nil, err
				}
			}
		}
	}

	// Delete existing sets and create new ones
	if err := s.setRepo.DeleteByMatchID(ctx, match.ID); err != nil {
		return nil, err
	}
	if err := s.setRepo.CreateBatch(ctx, matchID, result.Sets); err != nil {
		return nil, err
	}

	// Update match with new winner (also clears forfeit_winner_id if it was a forfeit)
	if err := s.repo.UpdateResult(ctx, matchID, newWinnerID); err != nil {
		return nil, err
	}

	// If winner changed and there's a next match, advance the new winner
	if winnerChanged && match.NextMatchID != nil {
		// Update match reference with new winner for advanceWinner
		match.WinnerID = &newWinnerID
		if err := s.advanceWinner(ctx, match, newWinnerID); err != nil {
			return nil, err
		}
	}

	// Fetch the updated match to return
	updatedMatch, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	// Load sets for the updated match
	sets, err := s.setRepo.GetByMatchID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	updatedMatch.Sets = sets

	response.Match = updatedMatch
	return response, nil
}

// StartMatch transitions a match from ready to in_progress.
func (s *matchService) StartMatch(ctx context.Context, matchID uint64) error {
	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return err
	}

	if match.Status != domain.MatchReady {
		return ErrMatchNotReady
	}

	return s.repo.UpdateStatus(ctx, matchID, domain.MatchInProgress)
}

// GetBracketState returns the current state of a tournament bracket.
func (s *matchService) GetBracketState(ctx context.Context, tournamentID uint64) (*BracketState, error) {
	matches, err := s.repo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return &BracketState{TournamentID: tournamentID}, nil
	}

	state := &BracketState{
		TournamentID: tournamentID,
		Matches:      matches,
	}

	// Check if this is a double elimination bracket (has grand final match)
	var grandFinalMatch *domain.Match
	for _, m := range matches {
		if m.BracketType == domain.BracketGrandFinal {
			grandFinalMatch = m
			break
		}
	}

	// Find total rounds (from winners bracket only for double elim)
	for _, m := range matches {
		// For double elim, only count winners bracket rounds for totalRounds
		if grandFinalMatch != nil && m.BracketType != domain.BracketWinners {
			continue
		}
		if m.Round > state.TotalRounds {
			state.TotalRounds = m.Round
		}
	}

	// Current round is the lowest round with non-completed matches
	state.CurrentRound = state.TotalRounds
	for _, m := range matches {
		if m.Status != domain.MatchCompleted && m.Round < state.CurrentRound {
			state.CurrentRound = m.Round
		}
	}

	// Check if complete
	if grandFinalMatch != nil {
		// Double elimination: complete when grand final has a winner
		if grandFinalMatch.WinnerID != nil {
			state.IsComplete = true
			state.ChampionID = grandFinalMatch.WinnerID
		}
	} else {
		// Single elimination: complete when final match (highest round) has winner
		for _, m := range matches {
			if m.Round == state.TotalRounds && m.WinnerID != nil {
				state.IsComplete = true
				state.ChampionID = m.WinnerID
				break
			}
		}
	}

	return state, nil
}

// advanceWinner places the winner into their next match.
func (s *matchService) advanceWinner(ctx context.Context, completedMatch *domain.Match, winnerID uint64) error {
	// For double elimination, also determine slot based on bracket type
	return s.advanceParticipantToMatch(ctx, completedMatch, winnerID, *completedMatch.NextMatchID, false)
}

// advanceLoser places the loser from a winners bracket match into the losers bracket.
// Only applies to winners bracket matches with a LoserMatchID set.
func (s *matchService) advanceLoser(ctx context.Context, completedMatch *domain.Match, loserID uint64) error {
	if completedMatch.LoserMatchID == nil {
		return nil // No losers bracket destination
	}

	return s.advanceParticipantToMatch(ctx, completedMatch, loserID, *completedMatch.LoserMatchID, true)
}

// advanceParticipantToMatch places a participant into a specified match slot.
// isLoser determines slot assignment for double elimination brackets.
// If the target match has a BYE in the other slot, the participant auto-advances.
func (s *matchService) advanceParticipantToMatch(ctx context.Context, sourceMatch *domain.Match, participantID uint64, targetMatchID uint64, isLoser bool) error {
	targetMatch, err := s.repo.GetByID(ctx, targetMatchID)
	if err != nil {
		return err
	}

	// Determine participant's name, icon URL, and seed
	name := ""
	iconURL := ""
	seed := 0
	if sourceMatch.Participant1ID != nil && *sourceMatch.Participant1ID == participantID {
		if sourceMatch.Participant1Name != nil {
			name = *sourceMatch.Participant1Name
		}
		if sourceMatch.Participant1IconURL != nil {
			iconURL = *sourceMatch.Participant1IconURL
		}
		if sourceMatch.Seed1 != nil {
			seed = *sourceMatch.Seed1
		}
	} else {
		if sourceMatch.Participant2Name != nil {
			name = *sourceMatch.Participant2Name
		}
		if sourceMatch.Participant2IconURL != nil {
			iconURL = *sourceMatch.Participant2IconURL
		}
		if sourceMatch.Seed2 != nil {
			seed = *sourceMatch.Seed2
		}
	}

	// Determine which slot in the target match
	slot := s.determineSlot(sourceMatch, targetMatch, isLoser)

	if err := s.repo.SetParticipant(ctx, targetMatch.ID, slot, participantID, name, iconURL, seed); err != nil {
		return err
	}

	// Refresh target match to check current state
	targetMatch, err = s.repo.GetByID(ctx, targetMatch.ID)
	if err != nil {
		return err
	}

	// Check if the other slot is a BYE (name == "BYE" and ID == nil)
	otherSlotIsBye := s.isSlotBye(targetMatch, 3-slot) // 3-slot gives us the other slot (1->2, 2->1)

	if otherSlotIsBye {
		// Auto-advance: this participant wins the match immediately
		if err := s.repo.UpdateResult(ctx, targetMatch.ID, participantID); err != nil {
			return err
		}

		// Advance winner to next match if there is one
		if targetMatch.NextMatchID != nil {
			// Update targetMatch with winner info for advanceWinner
			targetMatch.WinnerID = &participantID
			if err := s.advanceWinner(ctx, targetMatch, participantID); err != nil {
				return err
			}
		}
		return nil
	}

	// If both participants are set (no BYE), mark as ready
	if targetMatch.Participant1ID != nil && targetMatch.Participant2ID != nil {
		if err := s.repo.UpdateStatus(ctx, targetMatch.ID, domain.MatchReady); err != nil {
			return err
		}
	}

	return nil
}

// isSlotBye checks if a specific slot in a match is a BYE.
// A slot is a BYE if it has the name "BYE" and no participant ID.
func (s *matchService) isSlotBye(match *domain.Match, slot int) bool {
	if slot == 1 {
		return match.Participant1ID == nil &&
			match.Participant1Name != nil &&
			*match.Participant1Name == "BYE"
	}
	return match.Participant2ID == nil &&
		match.Participant2Name != nil &&
		*match.Participant2Name == "BYE"
}

// determineSlot determines which slot (1 or 2) a participant should occupy in the target match.
func (s *matchService) determineSlot(sourceMatch, targetMatch *domain.Match, isLoser bool) int {
	// For losers advancing to losers bracket from winners bracket
	if isLoser && sourceMatch.BracketType == domain.BracketWinners {
		// Check if target match has a BYE in slot 2 - if so, real loser goes to slot 1
		if s.isSlotBye(targetMatch, 2) {
			return 1
		}

		// W-R1 losers face each other in L-R1: use position parity
		if sourceMatch.Round == 1 {
			if sourceMatch.Position%2 == 1 {
				return 1
			}
			return 2
		}
		// W-R(N>1) losers drop into even losers rounds as slot 1 (top)
		// They face advancing losers bracket winners in slot 2 (bottom)
		return 1
	}

	// For grand final: winners final winner → slot 1, losers final winner → slot 2
	if targetMatch.BracketType == domain.BracketGrandFinal {
		if sourceMatch.BracketType == domain.BracketWinners {
			return 1
		}
		// Losers bracket winner goes to slot 2
		return 2
	}

	// For winners advancing within losers bracket to even rounds (drop-in rounds)
	// They should fill slot 2 (bottom), letting the drop-in fill slot 1 (top)
	if sourceMatch.BracketType == domain.BracketLosers && targetMatch.BracketType == domain.BracketLosers {
		if targetMatch.Round%2 == 0 {
			return 2
		}
	}

	// For winners advancing within winners bracket or losers bracket (odd rounds)
	// Standard rule: odd positions go to slot 1, even positions go to slot 2
	if sourceMatch.Position%2 == 1 {
		return 1
	}
	return 2
}

func isParticipant(match *domain.Match, participantID uint64) bool {
	if match.Participant1ID != nil && *match.Participant1ID == participantID {
		return true
	}
	if match.Participant2ID != nil && *match.Participant2ID == participantID {
		return true
	}
	return false
}

// getLoserID returns the ID of the losing participant (the one who isn't the winner).
func (s *matchService) getLoserID(match *domain.Match, winnerID uint64) uint64 {
	if match.Participant1ID != nil && *match.Participant1ID == winnerID {
		if match.Participant2ID != nil {
			return *match.Participant2ID
		}
	} else if match.Participant2ID != nil && *match.Participant2ID == winnerID {
		if match.Participant1ID != nil {
			return *match.Participant1ID
		}
	}
	return 0
}

// computeWinnerFromSets determines the winner based on sets won.
// Returns the winner ID or an error if sets are tied.
func computeWinnerFromSets(match *domain.Match, sets []domain.SetScore) (uint64, error) {
	var p1Wins, p2Wins int

	for _, set := range sets {
		if set.Participant1Score > set.Participant2Score {
			p1Wins++
		} else if set.Participant2Score > set.Participant1Score {
			p2Wins++
		}
		// Tied sets don't count for either participant
	}

	if p1Wins > p2Wins && match.Participant1ID != nil {
		return *match.Participant1ID, nil
	}
	if p2Wins > p1Wins && match.Participant2ID != nil {
		return *match.Participant2ID, nil
	}

	return 0, ErrSetsTied
}

// CountSetsWon returns the number of sets won by each participant.
func CountSetsWon(sets []domain.Set) (p1Sets, p2Sets int) {
	for _, set := range sets {
		if set.Participant1Score > set.Participant2Score {
			p1Sets++
		} else if set.Participant2Score > set.Participant1Score {
			p2Sets++
		}
	}
	return
}

// processEloUpdate handles ELO rating updates after a match completion.
// This runs asynchronously and logs errors rather than failing the match.
func (s *matchService) processEloUpdate(ctx context.Context, match *domain.Match, winnerID uint64) {
	// Skip if clients are not configured
	if s.tournamentClient == nil || s.communityClient == nil {
		return
	}

	// Get tournament to check if ELO is configured
	tournament, err := s.tournamentClient.GetTournament(ctx, match.TournamentID)
	if err != nil {
		log.Printf("ELO: failed to get tournament %d: %v", match.TournamentID, err)
		return
	}

	// Skip if no ELO system configured
	if tournament.EloSystemID == nil {
		return
	}

	// Determine loser ID
	var loserID uint64
	if match.Participant1ID != nil && *match.Participant1ID == winnerID {
		if match.Participant2ID != nil {
			loserID = *match.Participant2ID
		}
	} else if match.Participant2ID != nil {
		loserID = *match.Participant2ID
		if match.Participant1ID != nil && *match.Participant1ID != winnerID {
			loserID = *match.Participant1ID
		}
	}

	if loserID == 0 {
		log.Printf("ELO: could not determine loser for match %d", match.ID)
		return
	}

	// Get winner participant to get community_member_id
	winnerParticipant, err := s.tournamentClient.GetParticipant(ctx, winnerID)
	if err != nil {
		log.Printf("ELO: failed to get winner participant %d: %v", winnerID, err)
		return
	}
	if winnerParticipant.CommunityMemberID == nil {
		log.Printf("ELO: winner participant %d has no community_member_id", winnerID)
		return
	}

	// Get loser participant to get community_member_id
	loserParticipant, err := s.tournamentClient.GetParticipant(ctx, loserID)
	if err != nil {
		log.Printf("ELO: failed to get loser participant %d: %v", loserID, err)
		return
	}
	if loserParticipant.CommunityMemberID == nil {
		log.Printf("ELO: loser participant %d has no community_member_id", loserID)
		return
	}

	// Process ELO update
	result, err := s.communityClient.ProcessMatchElo(ctx, client.ProcessMatchEloRequest{
		EloSystemID:    *tournament.EloSystemID,
		MatchID:        match.ID,
		TournamentID:   match.TournamentID,
		WinnerMemberID: *winnerParticipant.CommunityMemberID,
		LoserMemberID:  *loserParticipant.CommunityMemberID,
	})
	if err != nil {
		log.Printf("ELO: failed to process match %d: %v", match.ID, err)
		return
	}

	log.Printf("ELO: match %d processed - winner %d: %d→%d (+%d), loser %d: %d→%d (%d)",
		match.ID,
		*winnerParticipant.CommunityMemberID, result.WinnerRatingBefore, result.WinnerRatingAfter, result.WinnerChange,
		*loserParticipant.CommunityMemberID, result.LoserRatingBefore, result.LoserRatingAfter, result.LoserChange,
	)
}

// revertMatchElo reverts ELO changes for a reopened match.
// This runs asynchronously and logs errors rather than failing the reopen.
func (s *matchService) revertMatchElo(ctx context.Context, matchID uint64) {
	if s.communityClient == nil {
		return
	}

	if err := s.communityClient.RevertMatchElo(ctx, matchID); err != nil {
		log.Printf("ELO: failed to revert match %d: %v", matchID, err)
		return
	}

	log.Printf("ELO: match %d reverted", matchID)
}

// ReopenMatch reopens a completed match, clearing its result and cascading
// the changes to all downstream matches that were affected.
func (s *matchService) ReopenMatch(ctx context.Context, matchID uint64) ([]*domain.Match, error) {
	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return nil, err
	}

	if match.Status != domain.MatchCompleted {
		return nil, ErrMatchNotCompleted
	}

	reopenedMatches := []*domain.Match{}

	if err := s.reopenMatchCascade(ctx, match, &reopenedMatches); err != nil {
		return nil, err
	}

	return reopenedMatches, nil
}

// reopenMatchCascade recursively reopens a match and all downstream affected matches.
func (s *matchService) reopenMatchCascade(ctx context.Context, match *domain.Match, reopened *[]*domain.Match) error {
	// If this match has a next match and had a winner, handle cascade
	if match.NextMatchID != nil && match.WinnerID != nil {
		nextMatch, err := s.repo.GetByID(ctx, *match.NextMatchID)
		if err != nil {
			return err
		}

		// Determine which slot the winner occupied in next match
		slot := 1
		if match.Position%2 == 0 {
			slot = 2
		}

		// Check if winner was actually placed in next match
		winnerInNextMatch := false
		if slot == 1 && nextMatch.Participant1ID != nil && *nextMatch.Participant1ID == *match.WinnerID {
			winnerInNextMatch = true
		} else if slot == 2 && nextMatch.Participant2ID != nil && *nextMatch.Participant2ID == *match.WinnerID {
			winnerInNextMatch = true
		}

		if winnerInNextMatch {
			// If next match was completed, recursively reopen it first
			if nextMatch.Status == domain.MatchCompleted {
				if err := s.reopenMatchCascade(ctx, nextMatch, reopened); err != nil {
					return err
				}
			}

			// Clear the participant slot in next match
			if err := s.repo.ClearParticipant(ctx, *match.NextMatchID, slot); err != nil {
				return err
			}

			// Update next match status
			if err := s.updateMatchStatusAfterClear(ctx, *match.NextMatchID); err != nil {
				return err
			}
		}
	}

	// For double elimination: cascade the loser to losers bracket
	if match.BracketType == domain.BracketWinners && match.LoserMatchID != nil && match.WinnerID != nil {
		loserID := s.getLoserID(match, *match.WinnerID)
		if loserID != 0 {
			loserMatch, err := s.repo.GetByID(ctx, *match.LoserMatchID)
			if err != nil {
				return err
			}

			// Determine which slot the loser occupied in loser match
			// W-R1 losers: position parity (odd → slot 1, even → slot 2)
			// W-R(N>1) losers: always slot 1 (top)
			var loserSlot int
			if match.Round == 1 {
				if match.Position%2 == 1 {
					loserSlot = 1
				} else {
					loserSlot = 2
				}
			} else {
				loserSlot = 1
			}

			// Check if loser was actually placed in loser match
			loserInLoserMatch := false
			if loserSlot == 1 && loserMatch.Participant1ID != nil && *loserMatch.Participant1ID == loserID {
				loserInLoserMatch = true
			} else if loserSlot == 2 && loserMatch.Participant2ID != nil && *loserMatch.Participant2ID == loserID {
				loserInLoserMatch = true
			}

			if loserInLoserMatch {
				// If loser match was completed, recursively reopen it first
				if loserMatch.Status == domain.MatchCompleted {
					if err := s.reopenMatchCascade(ctx, loserMatch, reopened); err != nil {
						return err
					}
				}

				// Clear the participant slot in loser match
				if err := s.repo.ClearParticipant(ctx, *match.LoserMatchID, loserSlot); err != nil {
					return err
				}

				// Update loser match status
				if err := s.updateMatchStatusAfterClear(ctx, *match.LoserMatchID); err != nil {
					return err
				}
			}
		}
	}

	// Delete sets for this match
	if err := s.setRepo.DeleteByMatchID(ctx, match.ID); err != nil {
		return err
	}

	// Reopen this match (clear winner, set status to ready)
	if err := s.repo.ReopenMatch(ctx, match.ID); err != nil {
		return err
	}

	// Revert ELO for this match (async, best-effort like processEloUpdate)
	go s.revertMatchElo(context.Background(), match.ID)

	// Track this match as reopened (update local state for return)
	match.WinnerID = nil
	match.ForfeitWinnerID = nil
	match.Status = domain.MatchReady
	match.Sets = nil
	*reopened = append(*reopened, match)

	return nil
}

// updateMatchStatusAfterClear updates a match's status after a participant is cleared.
func (s *matchService) updateMatchStatusAfterClear(ctx context.Context, matchID uint64) error {
	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		return err
	}

	// If either participant is missing, status should be pending
	if match.Participant1ID == nil || match.Participant2ID == nil {
		return s.repo.UpdateStatus(ctx, matchID, domain.MatchPending)
	}

	// Both participants are set, status should be ready
	return s.repo.UpdateStatus(ctx, matchID, domain.MatchReady)
}

// handleSwissMatchComplete updates Swiss standings and checks for round advancement.
func (s *matchService) handleSwissMatchComplete(ctx context.Context, match *domain.Match, sets []domain.SetScore, winnerID uint64) error {
	// Determine loser
	loserID := s.getLoserID(match, winnerID)

	// Calculate game wins/losses from sets
	var winnerGameWins, loserGameWins int
	for _, set := range sets {
		if match.Participant1ID != nil && *match.Participant1ID == winnerID {
			winnerGameWins += set.Participant1Score
			loserGameWins += set.Participant2Score
		} else {
			winnerGameWins += set.Participant2Score
			loserGameWins += set.Participant1Score
		}
	}

	// Update winner's standing
	winnerStanding, err := s.swissRepo.GetStandingByParticipant(ctx, match.TournamentID, winnerID)
	if err != nil {
		return err
	}
	winnerStanding.Wins++
	winnerStanding.GameWins += winnerGameWins
	winnerStanding.GameLosses += loserGameWins
	if err := s.swissRepo.UpdateStanding(ctx, winnerStanding); err != nil {
		return err
	}

	// Update loser's standing (if not a BYE match)
	if loserID != 0 {
		loserStanding, err := s.swissRepo.GetStandingByParticipant(ctx, match.TournamentID, loserID)
		if err != nil {
			return err
		}
		loserStanding.Losses++
		loserStanding.GameWins += loserGameWins
		loserStanding.GameLosses += winnerGameWins
		if err := s.swissRepo.UpdateStanding(ctx, loserStanding); err != nil {
			return err
		}
	}

	// Round advancement is manual - organizer clicks "Advance Round" button
	// which calls SwissService.AdvanceRound()

	return nil
}

// handleTiebreakerMatchComplete handles completion of a tiebreaker match.
// Advances winner to next match in mini bracket and checks for tiebreaker completion.
func (s *matchService) handleTiebreakerMatchComplete(ctx context.Context, match *domain.Match, winnerID uint64) error {
	// Create tiebreaker service
	tiebreakerSvc := NewTiebreakerService(s.tiebreakerRepo, s.swissRepo, s.repo)

	// Advance winner to next match if there is one
	if match.NextMatchID != nil {
		if err := tiebreakerSvc.AdvanceWinnerInTiebreaker(ctx, match, winnerID); err != nil {
			return err
		}
	}

	// Check if this tiebreaker (and all others) is complete
	allComplete, err := tiebreakerSvc.OnTiebreakerMatchComplete(ctx, match, winnerID)
	if err != nil {
		return err
	}

	// If all tiebreakers are complete, mark Swiss as complete
	if allComplete && s.swissRepo != nil {
		if err := s.swissRepo.SetTiebreakersComplete(ctx, match.TournamentID, true); err != nil {
			return err
		}

		// Get and update config to mark as complete
		config, err := s.swissRepo.GetConfig(ctx, match.TournamentID)
		if err != nil {
			return err
		}
		config.TiebreakersComplete = true
		config.IsComplete = true
		if err := s.swissRepo.UpdateConfig(ctx, config); err != nil {
			return err
		}
	}

	return nil
}

// attachSetsToMatches loads sets for all matches and attaches them.
// This is used for calculating game differential in standings.
func (s *matchService) attachSetsToMatches(ctx context.Context, matches []*domain.Match) error {
	if len(matches) == 0 {
		return nil
	}

	// Collect match IDs
	matchIDs := make([]uint64, len(matches))
	for i, m := range matches {
		matchIDs[i] = m.ID
	}

	// Load all sets in one query
	setsMap, err := s.setRepo.GetByMatchIDs(ctx, matchIDs)
	if err != nil {
		return err
	}

	// Attach sets to matches
	for _, m := range matches {
		if sets, ok := setsMap[m.ID]; ok {
			m.Sets = sets
		}
	}

	return nil
}

// participantStats tracks a participant's results during elimination standings calculation
type participantStats struct {
	id          uint64
	name        string
	iconURL     *string
	seed        int
	wins        int
	losses      int
	gameWins    int
	gameLosses  int
	finalRound  int
	bracketType domain.BracketType
	isChampion  bool
	isRunnerUp  bool
	// For double elim: track highest round reached in each bracket
	winnersRound int
	losersRound  int
}

// GetEliminationStandings calculates standings for single/double elimination brackets.
// Rankings are based on: how far each participant advanced in the bracket.
func (s *matchService) GetEliminationStandings(ctx context.Context, tournamentID uint64) ([]*domain.EliminationStanding, error) {
	matches, err := s.repo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return []*domain.EliminationStanding{}, nil
	}

	// Load sets for all matches to calculate game differential
	if err := s.attachSetsToMatches(ctx, matches); err != nil {
		return nil, err
	}

	// Determine if this is double elimination
	isDoubleElim := false
	for _, m := range matches {
		if m.BracketType == domain.BracketLosers || m.BracketType == domain.BracketGrandFinal {
			isDoubleElim = true
			break
		}
	}

	stats := make(map[uint64]*participantStats)

	// Helper to get or create participant stats
	getStats := func(id uint64, name *string, iconURL *string, seed *int) *participantStats {
		if id == 0 {
			return nil
		}
		if _, ok := stats[id]; !ok {
			n := ""
			if name != nil {
				n = *name
			}
			seedVal := 0
			if seed != nil {
				seedVal = *seed
			}
			stats[id] = &participantStats{
				id:          id,
				name:        n,
				iconURL:     iconURL,
				seed:        seedVal,
				bracketType: domain.BracketWinners,
			}
		}
		return stats[id]
	}

	// Process all matches
	for _, m := range matches {
		// Skip BYE matches (no opponent)
		if m.Participant1ID == nil || m.Participant2ID == nil {
			// But still track the real participant
			if m.Participant1ID != nil {
				getStats(*m.Participant1ID, m.Participant1Name, m.Participant1IconURL, m.Seed1)
			}
			if m.Participant2ID != nil {
				getStats(*m.Participant2ID, m.Participant2Name, m.Participant2IconURL, m.Seed2)
			}
			continue
		}

		p1Stats := getStats(*m.Participant1ID, m.Participant1Name, m.Participant1IconURL, m.Seed1)
		p2Stats := getStats(*m.Participant2ID, m.Participant2Name, m.Participant2IconURL, m.Seed2)

		// Track game wins/losses from sets
		for _, set := range m.Sets {
			if p1Stats != nil {
				p1Stats.gameWins += set.Participant1Score
				p1Stats.gameLosses += set.Participant2Score
			}
			if p2Stats != nil {
				p2Stats.gameWins += set.Participant2Score
				p2Stats.gameLosses += set.Participant1Score
			}
		}

		// Only process completed matches for W/L stats
		if m.Status != domain.MatchCompleted {
			continue
		}

		// Update win/loss records
		if m.WinnerID != nil {
			winnerStats := stats[*m.WinnerID]
			if winnerStats != nil {
				winnerStats.wins++
			}

			// Determine loser
			var loserID uint64
			if *m.WinnerID == *m.Participant1ID {
				loserID = *m.Participant2ID
			} else {
				loserID = *m.Participant1ID
			}
			loserStats := stats[loserID]
			if loserStats != nil {
				loserStats.losses++
				// Track where they were eliminated
				loserStats.finalRound = m.Round
				loserStats.bracketType = m.BracketType
			}

			// Track bracket progression
			if winnerStats != nil {
				switch m.BracketType {
				case domain.BracketWinners:
					if m.Round > winnerStats.winnersRound {
						winnerStats.winnersRound = m.Round
					}
				case domain.BracketLosers:
					if m.Round > winnerStats.losersRound {
						winnerStats.losersRound = m.Round
					}
				case domain.BracketGrandFinal:
					winnerStats.isChampion = true
				}
			}
		}
	}

	// Find champion and runner-up
	var grandFinalMatch *domain.Match
	for _, m := range matches {
		if m.BracketType == domain.BracketGrandFinal {
			grandFinalMatch = m
			break
		}
	}

	// For single elim, find the final match
	if grandFinalMatch == nil {
		maxRound := 0
		for _, m := range matches {
			if m.BracketType == domain.BracketWinners && m.Round > maxRound {
				maxRound = m.Round
			}
		}
		for _, m := range matches {
			if m.BracketType == domain.BracketWinners && m.Round == maxRound {
				grandFinalMatch = m
				break
			}
		}
	}

	if grandFinalMatch != nil && grandFinalMatch.WinnerID != nil {
		if championStats := stats[*grandFinalMatch.WinnerID]; championStats != nil {
			championStats.isChampion = true
		}
		// Runner-up is the loser of grand final
		var runnerUpID uint64
		if *grandFinalMatch.WinnerID == *grandFinalMatch.Participant1ID {
			runnerUpID = *grandFinalMatch.Participant2ID
		} else {
			runnerUpID = *grandFinalMatch.Participant1ID
		}
		if runnerUpStats := stats[runnerUpID]; runnerUpStats != nil {
			runnerUpStats.isRunnerUp = true
		}
	}

	// Calculate total rounds for placement calculations
	totalWinnersRounds := 0
	totalLosersRounds := 0
	for _, m := range matches {
		if m.BracketType == domain.BracketWinners && m.Round > totalWinnersRounds {
			totalWinnersRounds = m.Round
		}
		if m.BracketType == domain.BracketLosers && m.Round > totalLosersRounds {
			totalLosersRounds = m.Round
		}
	}

	// Get actual participant count (excluding BYEs)
	participantCount := len(stats)

	// Convert to standings with placements
	standings := make([]*domain.EliminationStanding, 0, len(stats))
	for _, ps := range stats {
		standing := &domain.EliminationStanding{
			ParticipantID:   ps.id,
			ParticipantName: ps.name,
			IconURL:         ps.iconURL,
			Seed:            ps.seed,
			Wins:            ps.wins,
			Losses:          ps.losses,
			GameWins:        ps.gameWins,
			GameLosses:      ps.gameLosses,
			FinalRound:      ps.finalRound,
			BracketType:     ps.bracketType,
		}

		// Calculate placement
		standing.Placement = calculatePlacement(ps, isDoubleElim, totalWinnersRounds, totalLosersRounds, participantCount)

		standings = append(standings, standing)
	}

	// Sort standings by placement
	sortEliminationStandings(standings)

	// Assign ranks
	for i, s := range standings {
		s.Rank = i + 1
	}

	return standings, nil
}

// calculatePlacement returns a human-readable placement string
func calculatePlacement(ps *participantStats, isDoubleElim bool, totalWinnersRounds, totalLosersRounds, participantCount int) string {
	if ps.isChampion {
		return "Champion"
	}
	if ps.isRunnerUp {
		return "2nd"
	}

	// If participant hasn't lost any matches yet, no placement
	if ps.losses == 0 {
		return ""
	}

	// In double elimination, losing in winners bracket just drops you to losers - not eliminated.
	// Participant is only eliminated when they lose in losers bracket (or grand final).
	// bracketType tracks the bracket where they LAST lost, so if it's still "winners",
	// they haven't lost in losers yet and are still competing.
	if isDoubleElim && ps.bracketType == domain.BracketWinners {
		return ""
	}

	// For single elimination, placement is based on round eliminated
	if !isDoubleElim {
		// If finalRound is 0, they lost in round 1
		if ps.finalRound == 0 {
			ps.finalRound = 1
		}
		roundsFromFinal := totalWinnersRounds - ps.finalRound
		return placementFromRoundsEliminated(roundsFromFinal, participantCount)
	}

	// For double elimination, placement is based on where they were eliminated
	// bracketType == losers means they lost their last match in losers bracket
	if ps.bracketType == domain.BracketLosers {
		// Use finalRound (the round they lost in) not losersRound (the round they last won)
		roundsFromFinal := totalLosersRounds - ps.finalRound
		return placementFromLosersRound(roundsFromFinal, totalLosersRounds, participantCount)
	}

	// bracketType == grand_final and not champion/runner-up shouldn't happen,
	// but fall through to a default placement just in case
	return ""
}

// placementFromRoundsEliminated calculates placement for single elimination.
// roundsFromFinal: 0 = lost in final (2nd), 1 = lost in semis (3rd-4th), etc.
// participantCount: actual number of participants to cap the placement range.
func placementFromRoundsEliminated(roundsFromFinal, participantCount int) string {
	// Calculate the standard placement range based on round
	var low, high int
	switch roundsFromFinal {
	case 0:
		return "2nd" // Lost in final
	case 1:
		low, high = 3, 4 // Lost in semifinals
	case 2:
		low, high = 5, 8 // Lost in quarterfinals
	case 3:
		low, high = 9, 16
	case 4:
		low, high = 17, 32
	default:
		low = 1 << roundsFromFinal
		high = 1 << (roundsFromFinal + 1)
		low++
	}

	// Cap the high end at the actual participant count
	if high > participantCount {
		high = participantCount
	}

	// If low and high are the same, just show one number
	if low == high {
		return ordinal(low)
	}

	return formatPlacementRange(low, high)
}

// placementFromLosersRound calculates placement for double elimination losers bracket.
func placementFromLosersRound(roundsFromFinal, totalLosersRounds, participantCount int) string {
	// In double elim losers bracket:
	// - Loser of losers final = 3rd
	// - Each earlier round has increasing placement
	var low, high int
	switch roundsFromFinal {
	case 0:
		return "3rd" // Lost losers final
	case 1:
		return "4th" // Lost losers semifinal
	case 2:
		low, high = 5, 6
	case 3:
		low, high = 7, 8
	default:
		low = 1 << (roundsFromFinal - 1)
		high = 1 << roundsFromFinal
		low += 3
		high += 2
	}

	// Cap the high end at the actual participant count
	if high > participantCount {
		high = participantCount
	}

	// If low and high are the same, just show one number
	if low == high {
		return ordinal(low)
	}

	return formatPlacementRange(low, high)
}

func formatPlacementRange(low, high int) string {
	return ordinal(low) + "-" + ordinal(high)
}

func ordinal(n int) string {
	suffix := "th"
	switch n % 10 {
	case 1:
		if n%100 != 11 {
			suffix = "st"
		}
	case 2:
		if n%100 != 12 {
			suffix = "nd"
		}
	case 3:
		if n%100 != 13 {
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

// parsePlacementOrder extracts the numeric order from a placement string.
// Empty placement (still competing) -> 0 (shown first)
// "Champion" -> 1, "2nd" -> 2, "3rd-4th" -> 3, "9th-12th" -> 9, etc.
func parsePlacementOrder(placement string) int {
	if placement == "" {
		return 0 // Still competing, shown first
	}
	if placement == "Champion" {
		return 1
	}

	// Extract the first number from the placement string
	num := 0
	for _, c := range placement {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		} else if num > 0 {
			break // Stop at first non-digit after we've found digits
		}
	}

	if num == 0 {
		return 9999 // Unknown placement goes last
	}
	return num
}

// sortEliminationStandings sorts by placement (champion first, then by seed as tiebreaker)
func sortEliminationStandings(standings []*domain.EliminationStanding) {
	// Sort using bubble sort (simple, works for small lists)
	for i := 0; i < len(standings)-1; i++ {
		for j := i + 1; j < len(standings); j++ {
			orderI := parsePlacementOrder(standings[i].Placement)
			orderJ := parsePlacementOrder(standings[j].Placement)

			shouldSwap := false
			if orderI > orderJ {
				shouldSwap = true
			} else if orderI == orderJ {
				// Same placement tier - sort by wins desc, then seed asc
				if standings[i].Wins < standings[j].Wins {
					shouldSwap = true
				} else if standings[i].Wins == standings[j].Wins && standings[i].Seed > standings[j].Seed {
					shouldSwap = true
				}
			}

			if shouldSwap {
				standings[i], standings[j] = standings[j], standings[i]
			}
		}
	}
}

