package engine

import (
	"fmt"
	"math"
	"sort"

	"github.com/braccet/bracket/internal/domain"
)

const FormatSwiss Format = "swiss"

// CalculateSwissRounds returns the recommended number of rounds for Swiss.
// Uses ceil(log2(n)) which ensures every possible record has at most one player.
func CalculateSwissRounds(participantCount int) int {
	if participantCount <= 1 {
		return 0
	}
	return int(math.Ceil(math.Log2(float64(participantCount))))
}

// GenerateRound1Pairings creates initial pairings based on seed.
// Pairs: 1v(n/2+1), 2v(n/2+2), etc.
// For 8 players: 1v5, 2v6, 3v7, 4v8
func GenerateRound1Pairings(standings []*domain.SwissStanding) []domain.Pairing {
	if len(standings) < 2 {
		return nil
	}

	// Sort by seed
	sorted := make([]*domain.SwissStanding, len(standings))
	copy(sorted, standings)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Seed < sorted[j].Seed
	})

	n := len(sorted)
	half := n / 2
	var pairings []domain.Pairing

	// Pair top half with bottom half
	for i := 0; i < half; i++ {
		p1 := sorted[i]
		p2 := sorted[half+i]
		pairings = append(pairings, domain.Pairing{
			Participant1ID:      p1.ParticipantID,
			Participant1Name:    p1.ParticipantName,
			Participant1IconURL: p1.ParticipantIconURL,
			Participant1Seed:    p1.Seed,
			Participant2ID:      p2.ParticipantID,
			Participant2Name:    p2.ParticipantName,
			Participant2IconURL: p2.ParticipantIconURL,
			Participant2Seed:    p2.Seed,
			IsBye:               false,
		})
	}

	// Handle odd participant with BYE
	if n%2 == 1 {
		last := sorted[n-1]
		pairings = append(pairings, domain.Pairing{
			Participant1ID:      last.ParticipantID,
			Participant1Name:    last.ParticipantName,
			Participant1IconURL: last.ParticipantIconURL,
			Participant1Seed:    last.Seed,
			IsBye:               true,
		})
	}

	return pairings
}

// GenerateSwissPairings creates pairings for round N (N > 1) based on standings.
// Algorithm:
// 1. Sort standings by: wins (desc), game diff (desc), opponent wins (desc), seed (asc)
// 2. Group players by score (wins count)
// 3. Pair within each score group, avoiding rematches
// 4. Float players down when a score group has an odd number
// 5. If odd participants, lowest unpaired gets BYE (if hasn't had one)
func GenerateSwissPairings(
	standings []*domain.SwissStanding,
	playedPairs map[uint64]map[uint64]bool, // p1ID -> p2ID -> true
	round int,
) ([]domain.Pairing, error) {
	if len(standings) < 2 {
		return nil, fmt.Errorf("need at least 2 participants for pairing")
	}

	// Sort standings by ranking criteria
	sorted := make([]*domain.SwissStanding, len(standings))
	copy(sorted, standings)
	sortStandings(sorted)

	n := len(sorted)
	paired := make(map[uint64]bool)
	var pairings []domain.Pairing

	// Handle BYE first if odd number of participants
	if n%2 == 1 {
		// Find lowest-ranked player without a bye
		for i := n - 1; i >= 0; i-- {
			if !sorted[i].HasBye {
				pairings = append(pairings, domain.Pairing{
					Participant1ID:      sorted[i].ParticipantID,
					Participant1Name:    sorted[i].ParticipantName,
					Participant1IconURL: sorted[i].ParticipantIconURL,
					Participant1Seed:    sorted[i].Seed,
					IsBye:               true,
				})
				paired[sorted[i].ParticipantID] = true
				break
			}
		}
		// If everyone has had a bye (rare), give it to the lowest ranked
		if len(pairings) == 0 {
			last := sorted[n-1]
			pairings = append(pairings, domain.Pairing{
				Participant1ID:      last.ParticipantID,
				Participant1Name:    last.ParticipantName,
				Participant1IconURL: last.ParticipantIconURL,
				Participant1Seed:    last.Seed,
				IsBye:               true,
			})
			paired[last.ParticipantID] = true
		}
	}

	// Group players by score (wins count)
	scoreGroups := groupByScore(sorted, paired)

	// Process each score group from highest to lowest
	var floater *domain.SwissStanding // Player who couldn't be paired in their group
	for _, group := range scoreGroups {
		// Add floater from previous group if any
		if floater != nil {
			group = append([]*domain.SwissStanding{floater}, group...)
			floater = nil
		}

		// Pair within this score group
		groupPairings, leftover := pairScoreGroup(group, playedPairs, paired)
		pairings = append(pairings, groupPairings...)

		// If someone couldn't be paired, they float down to the next group
		if leftover != nil {
			floater = leftover
		}
	}

	// If there's still a floater after all groups (shouldn't happen with valid input)
	if floater != nil {
		// Find any unpaired opponent (allowing cross-score pairing as last resort)
		for i := n - 1; i >= 0; i-- {
			p := sorted[i]
			if paired[p.ParticipantID] || p.ParticipantID == floater.ParticipantID {
				continue
			}
			pairings = append(pairings, domain.Pairing{
				Participant1ID:      floater.ParticipantID,
				Participant1Name:    floater.ParticipantName,
				Participant1IconURL: floater.ParticipantIconURL,
				Participant1Seed:    floater.Seed,
				Participant2ID:      p.ParticipantID,
				Participant2Name:    p.ParticipantName,
				Participant2IconURL: p.ParticipantIconURL,
				Participant2Seed:    p.Seed,
				IsBye:               false,
			})
			paired[floater.ParticipantID] = true
			paired[p.ParticipantID] = true
			break
		}
	}

	return pairings, nil
}

// groupByScore groups standings by wins count, returning groups in descending order of wins.
// Excludes already paired players.
func groupByScore(sorted []*domain.SwissStanding, paired map[uint64]bool) [][]*domain.SwissStanding {
	groups := make(map[int][]*domain.SwissStanding)
	var winCounts []int

	for _, s := range sorted {
		if paired[s.ParticipantID] {
			continue
		}
		if _, exists := groups[s.Wins]; !exists {
			winCounts = append(winCounts, s.Wins)
		}
		groups[s.Wins] = append(groups[s.Wins], s)
	}

	// Sort win counts descending
	sort.Sort(sort.Reverse(sort.IntSlice(winCounts)))

	// Build result in order
	var result [][]*domain.SwissStanding
	for _, wins := range winCounts {
		result = append(result, groups[wins])
	}
	return result
}

// pairScoreGroup pairs players within a score group, avoiding rematches.
// Returns the pairings and any leftover player who couldn't be paired.
func pairScoreGroup(group []*domain.SwissStanding, playedPairs map[uint64]map[uint64]bool, paired map[uint64]bool) ([]domain.Pairing, *domain.SwissStanding) {
	if len(group) == 0 {
		return nil, nil
	}
	if len(group) == 1 {
		return nil, group[0] // Single player floats down
	}

	// Try to find a valid pairing using backtracking
	var pairings []domain.Pairing
	remaining := make([]*domain.SwissStanding, len(group))
	copy(remaining, group)

	// Greedy approach with fallback
	for len(remaining) >= 2 {
		p1 := remaining[0]
		remaining = remaining[1:]

		// Find best opponent (first valid one in sorted order)
		var opponent *domain.SwissStanding
		var opponentIdx int = -1
		for i, p2 := range remaining {
			if !havePlayed(playedPairs, p1.ParticipantID, p2.ParticipantID) {
				opponent = p2
				opponentIdx = i
				break
			}
		}

		if opponent != nil {
			// Found valid opponent
			pairings = append(pairings, domain.Pairing{
				Participant1ID:      p1.ParticipantID,
				Participant1Name:    p1.ParticipantName,
				Participant1IconURL: p1.ParticipantIconURL,
				Participant1Seed:    p1.Seed,
				Participant2ID:      opponent.ParticipantID,
				Participant2Name:    opponent.ParticipantName,
				Participant2IconURL: opponent.ParticipantIconURL,
				Participant2Seed:    opponent.Seed,
				IsBye:               false,
			})
			paired[p1.ParticipantID] = true
			paired[opponent.ParticipantID] = true
			// Remove opponent from remaining
			remaining = append(remaining[:opponentIdx], remaining[opponentIdx+1:]...)
		} else {
			// No valid opponent in this group - p1 floats down
			// But first, try to pair with another unpaired player using backtracking
			if len(remaining) > 0 {
				// Try swapping with next player
				swapped := false
				for i := 0; i < len(remaining); i++ {
					// Check if remaining[i] can pair with someone else in remaining
					for j := i + 1; j < len(remaining); j++ {
						if !havePlayed(playedPairs, remaining[i].ParticipantID, remaining[j].ParticipantID) {
							// remaining[i] can pair with remaining[j], so p1 can pair with... no one
							// This means p1 should float
							break
						}
					}
					// Try p1 with remaining[i] even if they've played (rematch within group)
					if i == len(remaining)-1 {
						// Last chance: allow rematch
						opponent = remaining[i]
						opponentIdx = i
						pairings = append(pairings, domain.Pairing{
							Participant1ID:      p1.ParticipantID,
							Participant1Name:    p1.ParticipantName,
							Participant1IconURL: p1.ParticipantIconURL,
							Participant1Seed:    p1.Seed,
							Participant2ID:      opponent.ParticipantID,
							Participant2Name:    opponent.ParticipantName,
							Participant2IconURL: opponent.ParticipantIconURL,
							Participant2Seed:    opponent.Seed,
							IsBye:               false,
						})
						paired[p1.ParticipantID] = true
						paired[opponent.ParticipantID] = true
						remaining = append(remaining[:opponentIdx], remaining[opponentIdx+1:]...)
						swapped = true
						break
					}
				}
				if !swapped {
					// p1 floats down - put them back at the end
					return pairings, p1
				}
			} else {
				// p1 is alone, they float
				return pairings, p1
			}
		}
	}

	// If one player remains, they float down
	if len(remaining) == 1 {
		return pairings, remaining[0]
	}

	return pairings, nil
}

// PairingsToMatches converts pairings to Match objects for a Swiss round.
func PairingsToMatches(tournamentID uint64, round int, pairings []domain.Pairing) []*domain.Match {
	var matches []*domain.Match

	for i, p := range pairings {
		match := &domain.Match{
			TournamentID: tournamentID,
			BracketType:  domain.BracketSwiss,
			Round:        round,
			Position:     i + 1,
			Status:       domain.MatchPending,
		}

		// Participant 1 (always present)
		match.Participant1ID = &p.Participant1ID
		match.Participant1Name = &p.Participant1Name
		match.Participant1IconURL = p.Participant1IconURL
		match.Seed1 = &p.Participant1Seed

		if p.IsBye {
			// BYE match: auto-complete with p1 as winner
			byeName := "BYE"
			match.Participant2Name = &byeName
			match.WinnerID = match.Participant1ID
			match.Status = domain.MatchCompleted
		} else {
			// Normal match
			match.Participant2ID = &p.Participant2ID
			match.Participant2Name = &p.Participant2Name
			match.Participant2IconURL = p.Participant2IconURL
			match.Seed2 = &p.Participant2Seed
			match.Status = domain.MatchReady
		}

		matches = append(matches, match)
	}

	return matches
}

// BuildPlayedPairsMap builds a lookup map from pairing history.
func BuildPlayedPairsMap(history []*domain.SwissPairingHistory) map[uint64]map[uint64]bool {
	played := make(map[uint64]map[uint64]bool)
	for _, h := range history {
		if played[h.Participant1ID] == nil {
			played[h.Participant1ID] = make(map[uint64]bool)
		}
		if played[h.Participant2ID] == nil {
			played[h.Participant2ID] = make(map[uint64]bool)
		}
		played[h.Participant1ID][h.Participant2ID] = true
		played[h.Participant2ID][h.Participant1ID] = true
	}
	return played
}

// sortStandings sorts standings by Swiss ranking criteria.
func sortStandings(standings []*domain.SwissStanding) {
	sort.Slice(standings, func(i, j int) bool {
		// Primary: wins (descending)
		if standings[i].Wins != standings[j].Wins {
			return standings[i].Wins > standings[j].Wins
		}
		// Secondary: game differential (descending)
		diffI := standings[i].GameWins - standings[i].GameLosses
		diffJ := standings[j].GameWins - standings[j].GameLosses
		if diffI != diffJ {
			return diffI > diffJ
		}
		// Tertiary: opponent wins / Buchholz (descending)
		if standings[i].OpponentWins != standings[j].OpponentWins {
			return standings[i].OpponentWins > standings[j].OpponentWins
		}
		// Quaternary: original seed (ascending - lower is better)
		return standings[i].Seed < standings[j].Seed
	})
}

// havePlayed checks if two participants have played before.
func havePlayed(playedPairs map[uint64]map[uint64]bool, p1, p2 uint64) bool {
	if playedPairs == nil {
		return false
	}
	if playedPairs[p1] != nil && playedPairs[p1][p2] {
		return true
	}
	return false
}

// GetSwissBracketState builds a state summary for a Swiss bracket.
func GetSwissBracketState(tournamentID uint64, config *domain.SwissConfig, standings []*domain.SwissStanding, matches []*domain.Match) *domain.SwissBracketState {
	return &domain.SwissBracketState{
		TournamentID: tournamentID,
		Config:       config,
		Standings:    standings,
		Matches:      matches,
	}
}

// GetSwissBracketStateWithStages builds a state summary for a Swiss bracket including stages.
func GetSwissBracketStateWithStages(tournamentID uint64, config *domain.SwissConfig, standings []*domain.SwissStanding, matches []*domain.Match, stages []*domain.BracketStage) *domain.SwissBracketState {
	return &domain.SwissBracketState{
		TournamentID: tournamentID,
		Config:       config,
		Standings:    standings,
		Matches:      matches,
		Stages:       stages,
	}
}
