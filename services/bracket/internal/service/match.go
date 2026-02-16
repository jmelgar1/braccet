package service

import (
	"context"
	"errors"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/repository"
)

var (
	ErrMatchNotReady       = errors.New("match is not ready for result reporting")
	ErrInvalidWinner       = errors.New("winner must be a participant in the match")
	ErrMatchAlreadyComplete = errors.New("match has already been completed")
)

type MatchService interface {
	ReportResult(ctx context.Context, matchID uint64, result domain.MatchResult) error
	StartMatch(ctx context.Context, matchID uint64) error
	GetBracketState(ctx context.Context, tournamentID uint64) (*BracketState, error)
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
	repo repository.MatchRepository
}

func NewMatchService(repo repository.MatchRepository) MatchService {
	return &matchService{repo: repo}
}

// ReportResult records the result of a match and advances the winner.
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

	// Validate winner is a participant
	if !isParticipant(match, result.WinnerID) {
		return ErrInvalidWinner
	}

	// Update the match result
	if err := s.repo.UpdateResult(ctx, matchID, result); err != nil {
		return err
	}

	// Advance winner to next match if there is one
	if match.NextMatchID != nil {
		if err := s.advanceWinner(ctx, match, result.WinnerID); err != nil {
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
