package engine

import (
	"sort"

	"github.com/braccet/bracket/internal/domain"
)

// DetectTies analyzes Swiss standings and returns groups of participants
// who are tied on ALL ranking criteria (wins, game diff, opponent wins).
// Does NOT fall back to seed - that's the point of tiebreakers.
func DetectTies(standings []*domain.SwissStanding) []domain.TieGroup {
	if len(standings) < 2 {
		return nil
	}

	// Sort standings by ranking criteria (same as sortStandings in swiss.go)
	sorted := make([]*domain.SwissStanding, len(standings))
	copy(sorted, standings)
	sortStandings(sorted)

	var tieGroups []domain.TieGroup
	position := 1

	for i := 0; i < len(sorted); {
		// Find all participants tied with sorted[i]
		tiedParticipants := []uint64{sorted[i].ParticipantID}

		for j := i + 1; j < len(sorted); j++ {
			if areTied(sorted[i], sorted[j]) {
				tiedParticipants = append(tiedParticipants, sorted[j].ParticipantID)
			} else {
				break
			}
		}

		// Only create a tie group if there's actually a tie (2+ participants)
		if len(tiedParticipants) > 1 {
			tieGroups = append(tieGroups, domain.TieGroup{
				Position:       position,
				ParticipantIDs: tiedParticipants,
			})
		}

		// Move position forward by the number of tied participants
		position += len(tiedParticipants)
		i += len(tiedParticipants)
	}

	return tieGroups
}

// areTied returns true if two standings have identical ranking criteria (excluding seed)
func areTied(a, b *domain.SwissStanding) bool {
	// Compare wins
	if a.Wins != b.Wins {
		return false
	}

	// Compare game differential
	diffA := a.GameWins - a.GameLosses
	diffB := b.GameWins - b.GameLosses
	if diffA != diffB {
		return false
	}

	// Compare opponent wins (Buchholz)
	if a.OpponentWins != b.OpponentWins {
		return false
	}

	return true
}

// GenerateTiebreakerMatches creates matches to resolve a tie group.
// - 2 participants: single match
// - 3+ participants: single elimination bracket
// Returns matches with BracketType = BracketTiebreaker.
// The Round field stores the tie position being resolved.
func GenerateTiebreakerMatches(
	tournamentID uint64,
	tiePosition int,
	participants []*domain.SwissStanding,
	bestOf int,
) []*domain.Match {
	if len(participants) < 2 {
		return nil
	}

	if len(participants) == 2 {
		return generateTwoWayTiebreaker(tournamentID, tiePosition, participants)
	}

	return generateMultiWayTiebreaker(tournamentID, tiePosition, participants)
}

// generateTwoWayTiebreaker creates a single match between two tied participants
func generateTwoWayTiebreaker(
	tournamentID uint64,
	tiePosition int,
	participants []*domain.SwissStanding,
) []*domain.Match {
	// Sort by seed (lower seed gets participant1 slot)
	sorted := make([]*domain.SwissStanding, len(participants))
	copy(sorted, participants)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Seed < sorted[j].Seed
	})

	p1 := sorted[0]
	p2 := sorted[1]

	match := &domain.Match{
		TournamentID:        tournamentID,
		BracketType:         domain.BracketTiebreaker,
		Round:               tiePosition, // Use tie position as "round" for identification
		Position:            1,
		Participant1ID:      &p1.ParticipantID,
		Participant1Name:    &p1.ParticipantName,
		Participant1IconURL: p1.ParticipantIconURL,
		Seed1:               &p1.Seed,
		Participant2ID:      &p2.ParticipantID,
		Participant2Name:    &p2.ParticipantName,
		Participant2IconURL: p2.ParticipantIconURL,
		Seed2:               &p2.Seed,
		Status:              domain.MatchReady,
	}

	return []*domain.Match{match}
}

// generateMultiWayTiebreaker creates a mini single-elimination bracket
// Seeds within the mini bracket are based on original Swiss seed
func generateMultiWayTiebreaker(
	tournamentID uint64,
	tiePosition int,
	participants []*domain.SwissStanding,
) []*domain.Match {
	// Sort by seed (lower = better)
	sorted := make([]*domain.SwissStanding, len(participants))
	copy(sorted, participants)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Seed < sorted[j].Seed
	})

	// Convert to domain.Participant for reuse of bracket logic
	bracketParticipants := make([]domain.Participant, len(sorted))
	for i, s := range sorted {
		bracketParticipants[i] = domain.Participant{
			ID:      s.ParticipantID,
			Name:    s.ParticipantName,
			IconURL: ptrToString(s.ParticipantIconURL),
			Seed:    i + 1, // Re-seed 1-N within the mini bracket
		}
	}

	bracketSize := CalculateBracketSize(len(bracketParticipants))
	totalRounds := TotalRounds(bracketSize)
	pairings := GenerateSeedPairings(bracketSize)

	// Create seed map (nil for byes)
	seedMap := make(map[int]*domain.Participant)
	for i := range bracketParticipants {
		seedMap[i+1] = &bracketParticipants[i]
	}

	var matches []*domain.Match

	// Round 1 matches
	round1Matches := make([]*domain.Match, len(pairings))
	for i, pair := range pairings {
		match := createTiebreakerMatch(tournamentID, tiePosition, 1, i+1, seedMap[pair[0]], seedMap[pair[1]])
		round1Matches[i] = match
		matches = append(matches, match)
	}

	// Generate subsequent round placeholders
	prevRoundMatches := round1Matches
	for round := 2; round <= totalRounds; round++ {
		numMatches := len(prevRoundMatches) / 2
		roundMatches := make([]*domain.Match, numMatches)

		for i := 0; i < numMatches; i++ {
			match := &domain.Match{
				TournamentID: tournamentID,
				BracketType:  domain.BracketTiebreaker,
				Round:        tiePosition, // Keep tie position as identifier
				Position:     (round-1)*100 + i + 1, // Encode round in position (e.g., 101, 102 for round 2)
				Status:       domain.MatchPending,
			}
			roundMatches[i] = match
			matches = append(matches, match)
		}

		prevRoundMatches = roundMatches
	}

	// Process byes: advance participant automatically when opponent is bye
	for _, match := range round1Matches {
		processTiebreakerBye(match)
	}

	return matches
}

// createTiebreakerMatch creates a tiebreaker match with the given participants
func createTiebreakerMatch(tournamentID uint64, tiePosition, round, position int, p1, p2 *domain.Participant) *domain.Match {
	match := &domain.Match{
		TournamentID: tournamentID,
		BracketType:  domain.BracketTiebreaker,
		Round:        tiePosition, // Use tie position as round identifier
		Position:     (round-1)*100 + position, // Encode actual round in position
		Status:       domain.MatchPending,
	}

	if p1 != nil {
		match.Participant1ID = &p1.ID
		match.Participant1Name = &p1.Name
		if p1.IconURL != "" {
			match.Participant1IconURL = &p1.IconURL
		}
		match.Seed1 = &p1.Seed
	}
	if p2 != nil {
		match.Participant2ID = &p2.ID
		match.Participant2Name = &p2.Name
		if p2.IconURL != "" {
			match.Participant2IconURL = &p2.IconURL
		}
		match.Seed2 = &p2.Seed
	}

	// Set status based on participants
	if p1 != nil && p2 != nil {
		match.Status = domain.MatchReady
	}

	return match
}

// processTiebreakerBye handles a match where one participant is a bye
func processTiebreakerBye(match *domain.Match) {
	hasBye := match.Participant1ID == nil || match.Participant2ID == nil
	if !hasBye {
		return
	}

	if match.Participant1ID == nil && match.Participant2ID == nil {
		return
	}

	if match.Participant1ID != nil {
		match.WinnerID = match.Participant1ID
	} else {
		match.WinnerID = match.Participant2ID
	}

	match.Status = domain.MatchCompleted
}

// LinkTiebreakerMatches sets NextMatchID for tiebreaker bracket matches
// Must be called after matches have been saved and have IDs assigned.
func LinkTiebreakerMatches(matches []*domain.Match) {
	if len(matches) <= 1 {
		return
	}

	// Group by encoded round (position / 100)
	byRound := make(map[int][]*domain.Match)
	for _, m := range matches {
		round := m.Position/100 + 1 // Position 1-99 = round 1, 100-199 = round 2, etc.
		if m.Position < 100 {
			round = 1
		}
		byRound[round] = append(byRound[round], m)
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

// GetTiebreakerRoundFromPosition extracts the actual round number from encoded position
func GetTiebreakerRoundFromPosition(position int) int {
	if position < 100 {
		return 1
	}
	return position/100 + 1
}

// GetParticipantsForTieGroup extracts the SwissStanding entries for participants in a tie group
func GetParticipantsForTieGroup(standings []*domain.SwissStanding, tieGroup domain.TieGroup) []*domain.SwissStanding {
	idSet := make(map[uint64]bool)
	for _, id := range tieGroup.ParticipantIDs {
		idSet[id] = true
	}

	var result []*domain.SwissStanding
	for _, s := range standings {
		if idSet[s.ParticipantID] {
			result = append(result, s)
		}
	}
	return result
}

// ptrToString safely dereferences a string pointer, returning empty string if nil
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
