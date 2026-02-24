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
// 2. For each unpaired player (top to bottom), find highest-ranked unpaired opponent not played before
// 3. If odd participants, lowest unpaired gets BYE (if hasn't had one)
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

	// Pair remaining participants
	for i := 0; i < n; i++ {
		p1 := sorted[i]
		if paired[p1.ParticipantID] {
			continue
		}

		// Find the highest-ranked unpaired opponent who hasn't played p1
		var opponent *domain.SwissStanding
		for j := i + 1; j < n; j++ {
			p2 := sorted[j]
			if paired[p2.ParticipantID] {
				continue
			}
			if havePlayed(playedPairs, p1.ParticipantID, p2.ParticipantID) {
				continue
			}
			opponent = p2
			break
		}

		// If no valid opponent found, find any unpaired opponent (allowing rematch)
		if opponent == nil {
			for j := i + 1; j < n; j++ {
				p2 := sorted[j]
				if !paired[p2.ParticipantID] {
					opponent = p2
					break
				}
			}
		}

		if opponent == nil {
			// This shouldn't happen with valid input
			return nil, fmt.Errorf("could not find opponent for participant %d", p1.ParticipantID)
		}

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
