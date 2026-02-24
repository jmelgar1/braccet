package engine

import (
	"fmt"
	"sort"

	"github.com/braccet/bracket/internal/domain"
)

// DoubleElimination generates a double elimination bracket for the given participants.
// Returns all matches: winners bracket, losers bracket, and grand final.
// Participants are seeded by their Seed field (lower = better).
func DoubleElimination(tournamentID uint64, participants []domain.Participant) ([]*domain.Match, error) {
	return DoubleEliminationWithOptions(tournamentID, participants, false)
}

// DoubleEliminationWithOptions generates a double elimination bracket with configurable options.
// If skipFinals is true, the grand final match is not created.
func DoubleEliminationWithOptions(tournamentID uint64, participants []domain.Participant, skipFinals bool) ([]*domain.Match, error) {
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
	winnersRounds := TotalRounds(bracketSize)
	losersRounds := LosersRounds(winnersRounds)
	pairings := GenerateSeedPairings(bracketSize)

	// Create a map of seed -> participant (nil for byes)
	seedMap := make(map[int]*domain.Participant)
	for i := range sorted {
		seedMap[i+1] = &sorted[i]
	}

	var allMatches []*domain.Match

	// Generate winners bracket (same as single elim)
	winnersMatches := generateWinnersBracket(tournamentID, winnersRounds, pairings, seedMap)
	allMatches = append(allMatches, winnersMatches...)

	// Identify W-R1 BYE positions before generating losers bracket
	winnersR1ByePositions := getWinnersR1ByePositions(winnersMatches)

	// Generate losers bracket with BYE awareness
	losersMatches := generateLosersBracket(tournamentID, bracketSize, losersRounds, winnersR1ByePositions)
	allMatches = append(allMatches, losersMatches...)

	// Generate grand final (unless skipFinals is true)
	if !skipFinals {
		grandFinal := &domain.Match{
			TournamentID: tournamentID,
			BracketType:  domain.BracketGrandFinal,
			Round:        1,
			Position:     1,
			Status:       domain.MatchPending,
		}
		allMatches = append(allMatches, grandFinal)
	}

	// Process byes in round 1 of winners bracket
	for _, match := range winnersMatches {
		if match.Round == 1 {
			processByeDoubleElim(match)
		}
	}

	// Process byes in losers bracket round 1 (based on W-R1 BYEs)
	processLosersBracketByes(winnersMatches, losersMatches)

	return allMatches, nil
}

// generateWinnersBracket creates all winners bracket matches.
func generateWinnersBracket(tournamentID uint64, totalRounds int, pairings [][2]int, seedMap map[int]*domain.Participant) []*domain.Match {
	var matches []*domain.Match

	// Round 1 matches
	round1Matches := make([]*domain.Match, len(pairings))
	for i, pair := range pairings {
		match := createWinnersMatch(tournamentID, 1, i+1, seedMap[pair[0]], seedMap[pair[1]])
		round1Matches[i] = match
		matches = append(matches, match)
	}

	// Generate subsequent round placeholders
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
		}

		prevRoundMatches = roundMatches
	}

	return matches
}

// getWinnersR1ByePositions identifies which W-R1 positions are BYEs.
// A position is a BYE if one or both participants are missing.
// Returns a set (map) of 1-indexed positions that are BYEs.
func getWinnersR1ByePositions(winnersMatches []*domain.Match) map[int]bool {
	byePositions := make(map[int]bool)

	for _, m := range winnersMatches {
		if m.Round != 1 {
			continue
		}
		// A match is a BYE if either participant is nil
		if m.Participant1ID == nil || m.Participant2ID == nil {
			byePositions[m.Position] = true
		}
	}

	return byePositions
}

// generateLosersBracket creates all losers bracket matches.
// Losers bracket has 2*(N-1) rounds for N winners rounds.
// Pattern alternates: major rounds (losers vs losers) and minor rounds (drop-ins).
//
// For L-R1, we skip creating matches where BOTH source W-R1 matches are BYEs,
// since no participants will ever reach those matches.
func generateLosersBracket(tournamentID uint64, bracketSize, losersRounds int, winnersR1ByePositions map[int]bool) []*domain.Match {
	var matches []*domain.Match

	for round := 1; round <= losersRounds; round++ {
		numMatches := LosersMatchesInRound(bracketSize, round)

		for pos := 1; pos <= numMatches; pos++ {
			// For L-R1, check if this match is unreachable
			if round == 1 {
				// L-R1 match at position P receives losers from W-R1 positions (2P-1) and (2P)
				source1Pos := 2*pos - 1
				source2Pos := 2 * pos

				source1IsBye := winnersR1ByePositions[source1Pos]
				source2IsBye := winnersR1ByePositions[source2Pos]

				// Skip this L-R1 match if BOTH sources are BYEs
				if source1IsBye && source2IsBye {
					continue
				}
			}

			match := &domain.Match{
				TournamentID: tournamentID,
				BracketType:  domain.BracketLosers,
				Round:        round,
				Position:     pos,
				Status:       domain.MatchPending,
			}
			matches = append(matches, match)
		}
	}

	return matches
}

// createWinnersMatch creates a round 1 winners bracket match with the given participants.
func createWinnersMatch(tournamentID uint64, round, position int, p1, p2 *domain.Participant) *domain.Match {
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

// processByeDoubleElim handles a bye in double elimination.
// Unlike single elim, we DON'T advance the winner yet because we need IDs first.
// The bye winner will be advanced when LinkDoubleElimMatches is called.
func processByeDoubleElim(match *domain.Match) {
	hasBye := match.Participant1ID == nil || match.Participant2ID == nil
	if !hasBye {
		return
	}

	// If both are nil, this is an empty match
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

// LosersRounds returns the number of rounds in the losers bracket.
// Formula: 2 * (winnersRounds - 1)
func LosersRounds(winnersRounds int) int {
	if winnersRounds <= 1 {
		return 0
	}
	return 2 * (winnersRounds - 1)
}

// LosersMatchesInRound returns the number of matches in a losers bracket round.
// Pattern: R1 and R2 have bracketSize/4 matches, R3 and R4 have bracketSize/8, etc.
// Each pair of rounds halves the match count.
func LosersMatchesInRound(bracketSize, round int) int {
	if round < 1 || bracketSize < 4 {
		return 0
	}

	// Pair number: rounds 1-2 are pair 1, rounds 3-4 are pair 2, etc.
	pairNum := (round + 1) / 2

	// Matches in this pair = bracketSize / 2^(pairNum+1)
	matches := bracketSize >> uint(pairNum+1)

	if matches < 1 {
		return 1
	}
	return matches
}

// LinkDoubleElimMatches sets NextMatchID and LoserMatchID for all matches.
// Must be called after matches have been saved and have IDs assigned.
// Handles the case where grand final is not present (skip_finals mode).
func LinkDoubleElimMatches(matches []*domain.Match) {
	// Group matches by bracket type and round
	winners := filterByBracketType(matches, domain.BracketWinners)
	losers := filterByBracketType(matches, domain.BracketLosers)
	grandFinals := filterByBracketType(matches, domain.BracketGrandFinal)

	// Sort each group by round then position
	sortByRoundAndPosition(winners)
	sortByRoundAndPosition(losers)

	// Link winners bracket (NextMatchID within winners)
	linkWinnersNextMatch(winners)

	// Link losers bracket (NextMatchID within losers)
	linkLosersNextMatch(losers)

	// Link winners to losers (LoserMatchID)
	linkWinnersToLosers(winners, losers)

	// Link finals to grand final (only if grand final exists)
	if len(grandFinals) > 0 {
		grandFinal := grandFinals[0]

		// Winners final -> Grand Final (slot 1)
		winnersFinal := getMatchByRoundAndPosition(winners, maxRound(winners), 1)
		if winnersFinal != nil {
			winnersFinal.NextMatchID = &grandFinal.ID
		}

		// Losers final -> Grand Final (slot 2)
		losersFinal := getMatchByRoundAndPosition(losers, maxRound(losers), 1)
		if losersFinal != nil {
			losersFinal.NextMatchID = &grandFinal.ID
		}
	}
	// If no grand final (skip_finals mode), winners and losers finals have no NextMatchID
}

// linkWinnersNextMatch links winners bracket matches to their next match.
func linkWinnersNextMatch(winners []*domain.Match) {
	byRound := groupByRound(winners)
	maxR := maxRound(winners)

	for round := 1; round < maxR; round++ {
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

// linkLosersNextMatch links losers bracket matches to their next match.
func linkLosersNextMatch(losers []*domain.Match) {
	byRound := groupByRound(losers)
	maxR := maxRound(losers)

	for round := 1; round < maxR; round++ {
		currentRound := byRound[round]
		nextRound := byRound[round+1]

		// Build a position map for next round matches (handles gaps)
		nextByPosition := make(map[int]*domain.Match)
		for _, nm := range nextRound {
			nextByPosition[nm.Position] = nm
		}

		isOddRound := round%2 == 1

		for _, match := range currentRound {
			var nextMatchPos int

			if isOddRound {
				// Odd rounds (major): L-R(odd) match at position P goes to L-R(odd+1) at position P
				// The drop-in from winners fills the other slot
				nextMatchPos = match.Position
			} else {
				// Even rounds (minor with drop-ins): matches halve for next major round
				// L-R(even) match at position P goes to L-R(even+1) at position ceil(P/2)
				nextMatchPos = (match.Position + 1) / 2
			}

			nextMatch := nextByPosition[nextMatchPos]
			if nextMatch != nil {
				match.NextMatchID = &nextMatch.ID
			}
		}
	}
}

// linkWinnersToLosers sets LoserMatchID on winners bracket matches.
// Maps losers from each winners round to the appropriate losers round.
func linkWinnersToLosers(winners, losers []*domain.Match) {
	winnersByRound := groupByRound(winners)
	losersByRound := groupByRound(losers)

	maxWinnersRound := maxRound(winners)

	for winnersRound := 1; winnersRound <= maxWinnersRound; winnersRound++ {
		winnersMatches := winnersByRound[winnersRound]

		// Map winners round to losers round:
		// W-R1 losers -> L-R1
		// W-R2 losers -> L-R2
		// W-R3 losers -> L-R4
		// W-R(N) losers -> L-R(2*(N-1)) for N > 1
		var losersRound int
		if winnersRound == 1 {
			losersRound = 1
		} else {
			losersRound = 2 * (winnersRound - 1)
		}

		losersMatches := losersByRound[losersRound]
		if len(losersMatches) == 0 {
			continue
		}

		// Build a position map for losers matches in this round
		// This handles gaps when L-R1 matches are skipped due to double-BYEs
		losersByPosition := make(map[int]*domain.Match)
		for _, lm := range losersMatches {
			losersByPosition[lm.Position] = lm
		}

		// Determine how winners matches map to losers matches
		for i, wMatch := range winnersMatches {
			var losersMatchPos int
			var losersSlot int // 1 or 2

			if winnersRound == 1 {
				// W-R1: Pairs of losers face each other in L-R1
				// W-R1 matches at positions 1,2 -> L-R1 match at position 1
				// W-R1 matches at positions 3,4 -> L-R1 match at position 2
				// Use position-based calculation (positions are 1-indexed)
				losersMatchPos = (wMatch.Position + 1) / 2
				// Odd position -> slot 1, even position -> slot 2
				if wMatch.Position%2 == 1 {
					losersSlot = 1
				} else {
					losersSlot = 2
				}
			} else {
				// W-R(N>1): Losers drop into L-R(2*(N-1)) as slot 1 (top)
				// Alternating pattern to avoid rematches:
				// - Even winners rounds (2, 4, 6...): flip index (opposite side)
				// - Odd winners rounds (3, 5, 7...): same index (respective side)
				shouldFlip := winnersRound%2 == 0
				if shouldFlip {
					losersMatchPos = len(losersMatches) - i
				} else {
					losersMatchPos = i + 1
				}
				losersSlot = 1
			}

			// Look up the losers match by position (may not exist if skipped)
			loserMatch := losersByPosition[losersMatchPos]
			if loserMatch != nil {
				wMatch.LoserMatchID = &loserMatch.ID

				// Store the slot info in a way we can use during advancement
				// We'll compute this during match completion based on position parity
				_ = losersSlot // Used during runtime advancement
			}
			// If loserMatch is nil, this W-R1 match's loser has nowhere to go
			// because the corresponding L-R1 match was skipped (both sources were BYEs)
		}
	}
}

// Helper functions

func filterByBracketType(matches []*domain.Match, bt domain.BracketType) []*domain.Match {
	var result []*domain.Match
	for _, m := range matches {
		if m.BracketType == bt {
			result = append(result, m)
		}
	}
	return result
}

func sortByRoundAndPosition(matches []*domain.Match) {
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Round != matches[j].Round {
			return matches[i].Round < matches[j].Round
		}
		return matches[i].Position < matches[j].Position
	})
}

func groupByRound(matches []*domain.Match) map[int][]*domain.Match {
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
	return byRound
}

func maxRound(matches []*domain.Match) int {
	max := 0
	for _, m := range matches {
		if m.Round > max {
			max = m.Round
		}
	}
	return max
}

func getMatchByRoundAndPosition(matches []*domain.Match, round, position int) *domain.Match {
	for _, m := range matches {
		if m.Round == round && m.Position == position {
			return m
		}
	}
	return nil
}

// processLosersBracketByes marks losers bracket slots as BYEs based on missing participants.
//
// For L-R1:
// - When a W-R1 match is a BYE (one participant missing), no loser drops down
// - If one slot is a BYE, mark slot 2 as BYE for consistent display
// - Double-BYE cases are skipped during generation, so only single-BYE cases remain
//
// For L-R2:
// - If the source L-R1 match doesn't exist (was skipped), slot 2 is a BYE
//
// Mapping: L-R1 match at position P receives losers from:
//   - W-R1 match (2P-1) → normally slot 1
//   - W-R1 match (2P)   → normally slot 2
func processLosersBracketByes(winners, losers []*domain.Match) {
	winnersR1 := getMatchesByRound(winners, 1)
	losersR1 := getMatchesByRound(losers, 1)
	losersR2 := getMatchesByRound(losers, 2)

	// Build a set of existing L-R1 positions for cascade checking
	losersR1Positions := make(map[int]bool)
	for _, lm := range losersR1 {
		losersR1Positions[lm.Position] = true
	}

	// Process L-R1 single-BYE cases
	if len(winnersR1) > 0 && len(losersR1) > 0 {
		// Index W-R1 matches by position for lookup
		winnersR1ByPos := make(map[int]*domain.Match)
		for _, wm := range winnersR1 {
			winnersR1ByPos[wm.Position] = wm
		}

		for _, lMatch := range losersR1 {
			// L-R1 match at position P receives losers from W-R1 positions (2P-1) and (2P)
			slot1WPos := 2*lMatch.Position - 1
			slot2WPos := 2 * lMatch.Position

			slot1IsBye := false
			slot2IsBye := false

			if wm := winnersR1ByPos[slot1WPos]; wm != nil {
				slot1IsBye = isByeMatch(wm)
			}
			if wm := winnersR1ByPos[slot2WPos]; wm != nil {
				slot2IsBye = isByeMatch(wm)
			}

			// Note: double-BYE L-R1 matches are now skipped during generation,
			// so we only handle single-BYE cases here
			if slot1IsBye || slot2IsBye {
				// One BYE - always put it in slot 2 (bottom) for consistent display
				// The real loser will go to slot 1 (top) when they drop in
				byeName := "BYE"
				lMatch.Participant2Name = &byeName
			}
		}
	}

	// Process L-R2 cascade: if source L-R1 match doesn't exist, slot 2 is a BYE
	for _, lMatch := range losersR2 {
		// L-R2 match at position P receives winner from L-R1 at position P (slot 2)
		sourceL1Pos := lMatch.Position
		if !losersR1Positions[sourceL1Pos] {
			// Source L-R1 match doesn't exist (was skipped due to double-BYE)
			// Mark slot 2 as BYE - the W-R2 drop-in will be in slot 1
			byeName := "BYE"
			lMatch.Participant2Name = &byeName
		}
	}
}

// isByeMatch returns true if the match was a BYE (completed with one participant missing).
func isByeMatch(m *domain.Match) bool {
	return m.Status == domain.MatchCompleted &&
		(m.Participant1ID == nil || m.Participant2ID == nil)
}

// getMatchesByRound returns matches for a specific round, sorted by position.
func getMatchesByRound(matches []*domain.Match, round int) []*domain.Match {
	var result []*domain.Match
	for _, m := range matches {
		if m.Round == round {
			result = append(result, m)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Position < result[j].Position
	})
	return result
}

// GetDoubleElimBracketState builds a BracketState summary from a list of matches for double elimination.
func GetDoubleElimBracketState(tournamentID uint64, matches []*domain.Match) *BracketState {
	if len(matches) == 0 {
		return &BracketState{TournamentID: tournamentID}
	}

	state := &BracketState{
		TournamentID: tournamentID,
		Format:       FormatDoubleElim,
		Matches:      matches,
	}

	// Find total rounds (from winners bracket)
	winners := filterByBracketType(matches, domain.BracketWinners)
	for _, m := range winners {
		if m.Round > state.TotalRounds {
			state.TotalRounds = m.Round
		}
	}

	// Current round is the lowest round with non-completed matches (across all bracket types)
	state.CurrentRound = state.TotalRounds
	for _, m := range matches {
		if m.Status != domain.MatchCompleted && m.Round < state.CurrentRound {
			// Only consider winners bracket for current round calculation
			if m.BracketType == domain.BracketWinners {
				state.CurrentRound = m.Round
			}
		}
	}

	// Check if complete (grand final has winner)
	for _, m := range matches {
		if m.BracketType == domain.BracketGrandFinal && m.WinnerID != nil {
			state.IsComplete = true
			state.ChampionID = m.WinnerID
			break
		}
	}

	return state
}

// DetermineLoserSlot determines which slot (1 or 2) a loser should occupy in their losers bracket match.
// This is based on the winners match position and the target losers round.
func DetermineLoserSlot(winnersMatch *domain.Match, losersRound int) int {
	// For W-R1 losers entering L-R1 (major round):
	// Odd position winners match -> loser goes to slot 1
	// Even position winners match -> loser goes to slot 2
	if winnersMatch.Round == 1 {
		if winnersMatch.Position%2 == 1 {
			return 1
		}
		return 2
	}

	// For W-R(N>1) losers entering L-R(2*(N-1)) (drop-in round):
	// They always enter as slot 1 (top), with L-bracket winners in slot 2 (bottom)
	return 1
}
