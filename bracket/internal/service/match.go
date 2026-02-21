package service

import (
	"context"
	"errors"

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
	StartMatch(ctx context.Context, matchID uint64) error
	GetBracketState(ctx context.Context, tournamentID uint64) (*BracketState, error)
	ReopenMatch(ctx context.Context, matchID uint64) ([]*domain.Match, error)
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
	repo    repository.MatchRepository
	setRepo repository.SetRepository
}

func NewMatchService(repo repository.MatchRepository, setRepo repository.SetRepository) MatchService {
	return &matchService{repo: repo, setRepo: setRepo}
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

	return nil
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

	// Determine winner's name
	winnerName := ""
	if completedMatch.Participant1ID != nil && *completedMatch.Participant1ID == winnerID {
		if completedMatch.Participant1Name != nil {
			winnerName = *completedMatch.Participant1Name
		}
	} else if completedMatch.Participant2Name != nil {
		winnerName = *completedMatch.Participant2Name
	}

	// Determine which slot in the next match (based on position in current round)
	// Odd positions go to slot 1, even positions go to slot 2
	slot := 1
	if completedMatch.Position%2 == 0 {
		slot = 2
	}

	if err := s.repo.SetParticipant(ctx, nextMatch.ID, slot, winnerID, winnerName); err != nil {
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
