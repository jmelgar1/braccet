package service

import (
	"context"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/repository"
)

// StandingsService provides unified standings calculation for all bracket formats.
// It handles loading sets and calculating game differentials consistently.
type StandingsService interface {
	// LoadMatchesWithSets loads all matches for a tournament with their sets attached
	LoadMatchesWithSets(ctx context.Context, tournamentID uint64) ([]*domain.Match, error)

	// CalculateGameStats computes game wins/losses for each participant from match sets
	CalculateGameStats(matches []*domain.Match) map[uint64]*domain.GameStats

	// AttachSetsToMatches loads sets for the given matches and attaches them
	AttachSetsToMatches(ctx context.Context, matches []*domain.Match) error
}

type standingsService struct {
	matchRepo repository.MatchRepository
	setRepo   repository.SetRepository
}

// NewStandingsService creates a new standings service
func NewStandingsService(matchRepo repository.MatchRepository, setRepo repository.SetRepository) StandingsService {
	return &standingsService{
		matchRepo: matchRepo,
		setRepo:   setRepo,
	}
}

// LoadMatchesWithSets loads all matches for a tournament with their sets attached
func (s *standingsService) LoadMatchesWithSets(ctx context.Context, tournamentID uint64) ([]*domain.Match, error) {
	matches, err := s.matchRepo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if err := s.AttachSetsToMatches(ctx, matches); err != nil {
		return nil, err
	}

	return matches, nil
}

// AttachSetsToMatches loads sets for the given matches and attaches them
func (s *standingsService) AttachSetsToMatches(ctx context.Context, matches []*domain.Match) error {
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

// CalculateGameStats computes game wins/losses for each participant from match sets.
// This is the shared logic that both Swiss and Elimination formats use.
func (s *standingsService) CalculateGameStats(matches []*domain.Match) map[uint64]*domain.GameStats {
	stats := make(map[uint64]*domain.GameStats)

	getStats := func(id uint64) *domain.GameStats {
		if id == 0 {
			return nil
		}
		if _, ok := stats[id]; !ok {
			stats[id] = &domain.GameStats{
				ParticipantID: id,
			}
		}
		return stats[id]
	}

	for _, m := range matches {
		// Need both participants to calculate game differential
		if m.Participant1ID == nil || m.Participant2ID == nil {
			continue
		}

		p1Stats := getStats(*m.Participant1ID)
		p2Stats := getStats(*m.Participant2ID)

		// Accumulate game wins/losses from sets
		for _, set := range m.Sets {
			if p1Stats != nil {
				p1Stats.GameWins += set.Participant1Score
				p1Stats.GameLosses += set.Participant2Score
			}
			if p2Stats != nil {
				p2Stats.GameWins += set.Participant2Score
				p2Stats.GameLosses += set.Participant1Score
			}
		}
	}

	return stats
}
