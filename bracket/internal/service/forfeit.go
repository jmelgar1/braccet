package service

import (
	"context"
	"errors"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/repository"
)

var (
	ErrNoOpponentToAdvance = errors.New("no opponent to advance")
	ErrParticipantNotFound = errors.New("participant not found in any pending matches")
)

type ForfeitService interface {
	ProcessWithdrawal(ctx context.Context, tournamentID, participantID uint64) (*ForfeitSummary, error)
}

type ForfeitSummary struct {
	ForfeitedMatches []uint64
	AdvancedWinners  []uint64
}

type forfeitService struct {
	repo repository.MatchRepository
}

func NewForfeitService(repo repository.MatchRepository) ForfeitService {
	return &forfeitService{repo: repo}
}

// ProcessWithdrawal handles a participant withdrawal by forfeiting their pending matches
// and advancing opponents through the bracket.
func (s *forfeitService) ProcessWithdrawal(ctx context.Context, tournamentID, participantID uint64) (*ForfeitSummary, error) {
	// Get all pending/ready/in_progress matches for this participant
	matches, err := s.repo.GetPendingByParticipant(ctx, tournamentID, participantID)
	if err != nil {
		return nil, err
	}

	summary := &ForfeitSummary{
		ForfeitedMatches: make([]uint64, 0),
		AdvancedWinners:  make([]uint64, 0),
	}

	// Process each match - forfeit and advance opponent
	for _, match := range matches {
		opponentID := s.getOpponentID(match, participantID)

		// Skip if no opponent (edge case - shouldn't happen in valid bracket)
		if opponentID == nil {
			continue
		}

		// Record the forfeit
		if err := s.repo.UpdateForfeit(ctx, match.ID, *opponentID); err != nil {
			return nil, err
		}

		summary.ForfeitedMatches = append(summary.ForfeitedMatches, match.ID)
		summary.AdvancedWinners = append(summary.AdvancedWinners, *opponentID)

		// Advance winner to next match if there is one
		if match.NextMatchID != nil {
			if err := s.advanceForfeitWinner(ctx, match, *opponentID); err != nil {
				return nil, err
			}
		}
	}

	return summary, nil
}

// getOpponentID returns the ID of the opponent in a match, or nil if no opponent.
func (s *forfeitService) getOpponentID(match *domain.Match, withdrawnID uint64) *uint64 {
	if match.Participant1ID != nil && *match.Participant1ID == withdrawnID {
		return match.Participant2ID
	}
	if match.Participant2ID != nil && *match.Participant2ID == withdrawnID {
		return match.Participant1ID
	}
	return nil
}

// advanceForfeitWinner places the forfeit winner into their next match.
func (s *forfeitService) advanceForfeitWinner(ctx context.Context, completedMatch *domain.Match, winnerID uint64) error {
	nextMatch, err := s.repo.GetByID(ctx, *completedMatch.NextMatchID)
	if err != nil {
		return err
	}

	// Determine winner's name and seed
	winnerName := s.getParticipantName(completedMatch, winnerID)
	winnerSeed := s.getParticipantSeed(completedMatch, winnerID)

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

// getParticipantName returns the name of a participant in a match.
func (s *forfeitService) getParticipantName(match *domain.Match, participantID uint64) string {
	if match.Participant1ID != nil && *match.Participant1ID == participantID {
		if match.Participant1Name != nil {
			return *match.Participant1Name
		}
	}
	if match.Participant2ID != nil && *match.Participant2ID == participantID {
		if match.Participant2Name != nil {
			return *match.Participant2Name
		}
	}
	return ""
}

func (s *forfeitService) getParticipantSeed(match *domain.Match, participantID uint64) int {
	if match.Participant1ID != nil && *match.Participant1ID == participantID {
		if match.Seed1 != nil {
			return *match.Seed1
		}
	}
	if match.Participant2ID != nil && *match.Participant2ID == participantID {
		if match.Seed2 != nil {
			return *match.Seed2
		}
	}
	return 0
}
