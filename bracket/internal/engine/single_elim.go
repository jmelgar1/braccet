package engine

import (
	"fmt"
	"sort"

	"github.com/braccet/bracket/internal/domain"
)

// SingleElimination generates a single elimination bracket for the given participants.
// Participants are seeded by their Seed field (lower = better).
// Returns all matches including round 1 and placeholder matches for subsequent rounds.
func SingleElimination(tournamentID uint64, participants []domain.Participant) ([]*domain.Match, error) {
	if len(participants) < 2 {
		return nil, fmt.Errorf("need at least 2 participants, got %d", len(participants))
	}

	// Sort participants by seed
	sorted := make([]domain.Participant, len(participants))
	copy(sorted, participants)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Seed < sorted[j].Seed
	})

	bracketSize := CalculateBracketSize(len(participants))
	totalRounds := TotalRounds(bracketSize)
	pairings := GenerateSeedPairings(bracketSize)

	// Create a map of seed -> participant (nil for byes)
	seedMap := make(map[int]*domain.Participant)
	for i := range sorted {
		seedMap[i+1] = &sorted[i]
	}
	// Seeds beyond participant count are byes (nil in map)

	// Generate all matches
	var matches []*domain.Match

	// Round 1 matches
	round1Matches := make([]*domain.Match, len(pairings))
	for i, pair := range pairings {
		match := createMatch(tournamentID, 1, i+1, seedMap[pair[0]], seedMap[pair[1]])
		round1Matches[i] = match
		matches = append(matches, match)
	}

	// Generate subsequent round placeholders
	// Each round has half the matches of the previous
	prevRoundMatches := round1Matches
	for round := 2; round <= totalRounds; round++ {
		numMatches := len(prevRoundMatches) / 2
		roundMatches := make([]*domain.Match, numMatches)

		for i := range numMatches {
			match := &domain.Match{
				TournamentID: tournamentID,
				BracketType:  domain.BracketWinners,
				Round:        round,
				Position:     i + 1,
				Status:       domain.MatchPending,
			}
			roundMatches[i] = match
			matches = append(matches, match)

			// Link previous round matches to this one
			prevRoundMatches[i*2].NextMatchID = nil   // Will be set after IDs assigned
			prevRoundMatches[i*2+1].NextMatchID = nil // Will be set after IDs assigned
		}

		prevRoundMatches = roundMatches
	}

	// Process byes: advance participant automatically when opponent is bye
	for _, match := range round1Matches {
		processBye(match)
	}

	return matches, nil
}

// createMatch creates a round 1 match with the given participants.
// p1 or p2 can be nil to indicate a bye.
func createMatch(tournamentID uint64, round, position int, p1, p2 *domain.Participant) *domain.Match {
	match := &domain.Match{
		TournamentID: tournamentID,
		BracketType:  domain.BracketWinners,
		Round:        round,
		Position:     position,
		Status:       domain.MatchPending,
	}

	if p1 != nil {
		match.Participant1ID = &p1.ID
		match.Participant1Name = &p1.Name
		match.Seed1 = &p1.Seed
	}
	if p2 != nil {
		match.Participant2ID = &p2.ID
		match.Participant2Name = &p2.Name
		match.Seed2 = &p2.Seed
	}

	// Set status based on participants
	if p1 != nil && p2 != nil {
		match.Status = domain.MatchReady
	}

	return match
}

// processBye handles a match where one participant is a bye.
// The non-bye participant automatically wins.
func processBye(match *domain.Match) {
	hasBye := match.Participant1ID == nil || match.Participant2ID == nil
	if !hasBye {
		return
	}

	// If both are nil, this is an empty match (shouldn't happen in valid bracket)
	if match.Participant1ID == nil && match.Participant2ID == nil {
		return
	}

	// Determine winner (the non-bye participant)
	if match.Participant1ID != nil {
		match.WinnerID = match.Participant1ID
	} else {
		match.WinnerID = match.Participant2ID
	}

	match.Status = domain.MatchCompleted
}

// LinkMatches sets NextMatchID for all matches based on bracket structure.
// Must be called after matches have been saved and have IDs assigned.
func LinkMatches(matches []*domain.Match) {
	// Group matches by round
	byRound := make(map[int][]*domain.Match)
	for _, m := range matches {
		byRound[m.Round] = append(byRound[m.Round], m)
	}

	// Sort each round by position
	for round := range byRound {
		sort.Slice(byRound[round], func(i, j int) bool {
			return byRound[round][i].Position < byRound[round][j].Position
		})
	}

	// Find max round
	maxRound := 0
	for round := range byRound {
		if round > maxRound {
			maxRound = round
		}
	}

	// Link each match to the next round
	for round := 1; round < maxRound; round++ {
		currentRound := byRound[round]
		nextRound := byRound[round+1]

		for i, match := range currentRound {
			nextMatchIdx := i / 2
			if nextMatchIdx < len(nextRound) {
				nextMatchID := nextRound[nextMatchIdx].ID
				match.NextMatchID = &nextMatchID
			}
		}
	}
}

// GetBracketState builds a BracketState summary from a list of matches.
func GetBracketState(tournamentID uint64, matches []*domain.Match) *BracketState {
	if len(matches) == 0 {
		return &BracketState{TournamentID: tournamentID}
	}

	state := &BracketState{
		TournamentID: tournamentID,
		Format:       FormatSingleElim,
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

	return state
}

// Format represents the bracket format.
type Format string

const (
	FormatSingleElim Format = "single_elimination"
	FormatDoubleElim Format = "double_elimination"
)

// BracketState represents the current state of a tournament bracket.
type BracketState struct {
	TournamentID uint64
	Format       Format
	TotalRounds  int
	CurrentRound int
	Matches      []*domain.Match
	IsComplete   bool
	ChampionID   *uint64
}
