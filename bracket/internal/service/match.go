package service

import (
	"context"
	"errors"
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

	// Find total rounds and current round
	for _, m := range matches {
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

	// Check if complete (final match has winner)
	for _, m := range matches {
		if m.Round == state.TotalRounds && m.WinnerID != nil {
			state.IsComplete = true
			state.ChampionID = m.WinnerID
			break
		}
	}

	return state, nil
}

// advanceWinner places the winner into their next match.
func (s *matchService) advanceWinner(ctx context.Context, completedMatch *domain.Match, winnerID uint64) error {
	nextMatch, err := s.repo.GetByID(ctx, *completedMatch.NextMatchID)
	if err != nil {
		return err
	}

	// Determine winner's name and seed
	winnerName := ""
	winnerSeed := 0
	if completedMatch.Participant1ID != nil && *completedMatch.Participant1ID == winnerID {
		if completedMatch.Participant1Name != nil {
			winnerName = *completedMatch.Participant1Name
		}
		if completedMatch.Seed1 != nil {
			winnerSeed = *completedMatch.Seed1
		}
	} else {
		if completedMatch.Participant2Name != nil {
			winnerName = *completedMatch.Participant2Name
		}
		if completedMatch.Seed2 != nil {
			winnerSeed = *completedMatch.Seed2
		}
	}

	// Determine which slot in the next match (based on position in current round)
	// Odd positions go to slot 1, even positions go to slot 2
	slot := 1
	if completedMatch.Position%2 == 0 {
		slot = 2
	}

	if err := s.repo.SetParticipant(ctx, nextMatch.ID, slot, winnerID, winnerName, winnerSeed); err != nil {
		return err
	}

	// Refresh next match to check if both participants are now set
	nextMatch, err = s.repo.GetByID(ctx, nextMatch.ID)
	if err != nil {
		return err
	}

	// If both participants are set, mark as ready
	if nextMatch.Participant1ID != nil && nextMatch.Participant2ID != nil {
		if err := s.repo.UpdateStatus(ctx, nextMatch.ID, domain.MatchReady); err != nil {
			return err
		}
	}

	return nil
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

	// Delete sets for this match
	if err := s.setRepo.DeleteByMatchID(ctx, match.ID); err != nil {
		return err
	}

	// Reopen this match (clear winner, set status to ready)
	if err := s.repo.ReopenMatch(ctx, match.ID); err != nil {
		return err
	}

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
