package service

import (
	"context"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/engine"
	"github.com/braccet/bracket/internal/repository"
)

type BracketService interface {
	GenerateSingleElimination(ctx context.Context, tournamentID uint64, participants []domain.Participant) (*BracketState, error)
}

type bracketService struct {
	repo      repository.MatchRepository
	stageRepo repository.StageRepository
}

func NewBracketService(repo repository.MatchRepository, stageRepo repository.StageRepository) BracketService {
	return &bracketService{repo: repo, stageRepo: stageRepo}
}

// GenerateSingleElimination creates a single elimination bracket and persists it.
func (s *bracketService) GenerateSingleElimination(ctx context.Context, tournamentID uint64, participants []domain.Participant) (*BracketState, error) {
	// Generate matches in memory
	matches, err := engine.SingleElimination(tournamentID, participants)
	if err != nil {
		return nil, err
	}

	// Save matches to DB (assigns IDs)
	if err := s.repo.CreateBatch(ctx, matches); err != nil {
		return nil, err
	}

	// Link matches now that we have IDs
	engine.LinkMatches(matches)

	// Persist the links
	if err := s.repo.UpdateNextMatchLinks(ctx, matches); err != nil {
		return nil, err
	}

	// Advance bye winners through the bracket
	if err := s.advanceByeWinners(ctx, matches); err != nil {
		return nil, err
	}

	// Create default stages for the bracket
	state := buildBracketState(tournamentID, matches)
	if s.stageRepo != nil && state.TotalRounds > 0 {
		if err := s.stageRepo.CreateDefaultStages(ctx, tournamentID, domain.BracketWinners, state.TotalRounds); err != nil {
			return nil, err
		}
	}

	// Reload matches to get final state
	matches, err = s.repo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	return buildBracketState(tournamentID, matches), nil
}

// advanceByeWinners propagates winners from bye matches to subsequent rounds.
func (s *bracketService) advanceByeWinners(ctx context.Context, matches []*domain.Match) error {
	// Process completed matches (byes) and advance their winners
	for _, match := range matches {
		if match.Status == domain.MatchCompleted && match.WinnerID != nil && match.NextMatchID != nil {
			// Find the next match
			var nextMatch *domain.Match
			for _, m := range matches {
				if m.ID == *match.NextMatchID {
					nextMatch = m
					break
				}
			}

			if nextMatch == nil {
				continue
			}

			// Determine winner's name, icon URL, and seed
			winnerName := ""
			winnerIconURL := ""
			winnerSeed := 0
			if match.Participant1ID != nil && *match.Participant1ID == *match.WinnerID {
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

			// Determine slot based on position
			slot := 1
			if match.Position%2 == 0 {
				slot = 2
			}

			if err := s.repo.SetParticipant(ctx, nextMatch.ID, slot, *match.WinnerID, winnerName, winnerIconURL, winnerSeed); err != nil {
				return err
			}

			// Update in-memory for subsequent iterations
			if slot == 1 {
				nextMatch.Participant1ID = match.WinnerID
				nextMatch.Participant1Name = &winnerName
				nextMatch.Participant1IconURL = &winnerIconURL
				nextMatch.Seed1 = &winnerSeed
			} else {
				nextMatch.Participant2ID = match.WinnerID
				nextMatch.Participant2Name = &winnerName
				nextMatch.Participant2IconURL = &winnerIconURL
				nextMatch.Seed2 = &winnerSeed
			}
		}
	}

	// Check if any next-round matches are now ready (both participants set)
	for _, match := range matches {
		if match.Status == domain.MatchPending && match.Participant1ID != nil && match.Participant2ID != nil {
			if err := s.repo.UpdateStatus(ctx, match.ID, domain.MatchReady); err != nil {
				return err
			}
			match.Status = domain.MatchReady
		}
	}

	// Recursively handle any matches that became completed due to both participants being byes
	// (This handles cases like only 2 participants in an 8-bracket where multiple rounds auto-complete)
	for _, match := range matches {
		if match.Status == domain.MatchReady {
			// Check if this is a bye match (one participant nil)
			if match.Participant1ID == nil || match.Participant2ID == nil {
				// Auto-complete the bye
				var winnerID *uint64
				if match.Participant1ID != nil {
					winnerID = match.Participant1ID
				} else {
					winnerID = match.Participant2ID
				}

				if winnerID != nil {
					// For bye matches, just update the winner directly (no sets needed)
					if err := s.repo.UpdateResult(ctx, match.ID, *winnerID); err != nil {
						return err
					}
					match.Status = domain.MatchCompleted
					match.WinnerID = winnerID

					// Recurse to handle this completed match
					if match.NextMatchID != nil {
						return s.advanceByeWinners(ctx, matches)
					}
				}
			}
		}
	}

	return nil
}

func buildBracketState(tournamentID uint64, matches []*domain.Match) *BracketState {
	if len(matches) == 0 {
		return &BracketState{TournamentID: tournamentID}
	}

	state := &BracketState{
		TournamentID: tournamentID,
		Matches:      matches,
	}

	for _, m := range matches {
		if m.Round > state.TotalRounds {
			state.TotalRounds = m.Round
		}
	}

	state.CurrentRound = state.TotalRounds
	for _, m := range matches {
		if m.Status != domain.MatchCompleted && m.Round < state.CurrentRound {
			state.CurrentRound = m.Round
		}
	}

	for _, m := range matches {
		if m.Round == state.TotalRounds && m.WinnerID != nil {
			state.IsComplete = true
			state.ChampionID = m.WinnerID
			break
		}
	}

	return state
}
