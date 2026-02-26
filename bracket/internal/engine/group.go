package engine

import (
	"sort"

	"github.com/braccet/bracket/internal/domain"
)

// GroupBracketParams contains parameters for generating a group bracket
type GroupBracketParams struct {
	TournamentID      uint64
	StageID           uint64
	GroupID           uint64
	Format            string // single_elimination, double_elimination, swiss
	Participants      []domain.Participant
	SwissRounds       *int // Optional override for Swiss format
	WinsToAdvance     *int // Swiss threshold: wins needed to advance (excluding BYEs)
	LossesToEliminate *int // Swiss threshold: losses that eliminate a participant
	SkipFinals        bool // When true, skip final match(es) and determine standings by bracket position
}

// GenerateGroupBracket creates a bracket for a single group
// It uses the existing engines (SingleElimination, DoubleElimination, Swiss) but
// tags all matches with the stage and group IDs.
// If SkipFinals is true, the final match(es) are not generated.
func GenerateGroupBracket(params GroupBracketParams) ([]*domain.Match, error) {
	var matches []*domain.Match
	var err error

	switch params.Format {
	case "single_elimination":
		matches, err = SingleEliminationWithOptions(params.TournamentID, params.Participants, params.SkipFinals)
	case "double_elimination":
		matches, err = DoubleEliminationWithOptions(params.TournamentID, params.Participants, params.SkipFinals)
	case "swiss":
		// For Swiss, we generate round 1 matches
		// Swiss doesn't have a "final" to skip - tiebreakers are handled separately
		standings := make([]*domain.SwissStanding, len(params.Participants))
		for i, p := range params.Participants {
			standings[i] = &domain.SwissStanding{
				TournamentID:    params.TournamentID,
				ParticipantID:   p.ID,
				ParticipantName: p.Name,
				Seed:            p.Seed,
			}
		}
		pairings := GenerateRound1Pairings(standings)
		matches = PairingsToMatches(params.TournamentID, 1, pairings)
	default:
		matches, err = SingleEliminationWithOptions(params.TournamentID, params.Participants, params.SkipFinals)
	}

	if err != nil {
		return nil, err
	}

	// Tag all matches with stage and group IDs
	for _, m := range matches {
		m.StageID = &params.StageID
		m.GroupID = &params.GroupID
	}

	return matches, nil
}

// SerpentineSeeding distributes ranked participants across groups using serpentine order.
// This ensures even distribution of skill across groups.
//
// Example with 9 participants and 3 groups:
// Row 0 (left to right):  Group A gets #1, Group B gets #2, Group C gets #3
// Row 1 (right to left):  Group C gets #4, Group B gets #5, Group A gets #6
// Row 2 (left to right):  Group A gets #7, Group B gets #8, Group C gets #9
//
// Result:
// Group A: 1, 6, 7 (seeds 1, 6, 7)
// Group B: 2, 5, 8 (seeds 2, 5, 8)
// Group C: 3, 4, 9 (seeds 3, 4, 9)
func SerpentineSeeding(rankedParticipants []uint64, numGroups int) map[int][]uint64 {
	result := make(map[int][]uint64)
	for i := 0; i < numGroups; i++ {
		result[i] = make([]uint64, 0)
	}

	for i, participantID := range rankedParticipants {
		row := i / numGroups
		posInRow := i % numGroups

		var groupOrder int
		if row%2 == 0 {
			// Even rows: left to right (0, 1, 2, ...)
			groupOrder = posInRow
		} else {
			// Odd rows: right to left (2, 1, 0, ...)
			groupOrder = numGroups - 1 - posInRow
		}

		result[groupOrder] = append(result[groupOrder], participantID)
	}

	return result
}

// CalculateGroupStandings computes standings for a group based on completed matches
// and the specified ranking criteria (in priority order).
// If format is single/double elimination and criteria is empty, uses bracket placement.
func CalculateGroupStandings(
	standings []*domain.GroupStanding,
	matches []*domain.Match,
	criteria []domain.RankingCriterion,
) []*domain.GroupStanding {
	return CalculateGroupStandingsWithFormat(standings, matches, criteria, "")
}

// CalculateGroupStandingsWithFormat computes standings with format-aware ranking.
// For elimination formats with empty criteria, uses bracket placement (wins/losses).
// For Swiss or when criteria is provided, uses the criteria-based ranking.
func CalculateGroupStandingsWithFormat(
	standings []*domain.GroupStanding,
	matches []*domain.Match,
	criteria []domain.RankingCriterion,
	format string,
) []*domain.GroupStanding {
	// Build a map for quick lookup
	standingMap := make(map[uint64]*domain.GroupStanding)
	for _, s := range standings {
		standingMap[s.ParticipantID] = s
	}

	// Reset stats
	for _, s := range standings {
		s.MatchWins = 0
		s.MatchLosses = 0
		s.SetWins = 0
		s.SetLosses = 0
		s.PointsScored = 0
		s.PointsAgainst = 0
		s.OpponentMatchWins = 0
	}

	// Calculate stats from matches
	for _, m := range matches {
		if m.Status != domain.MatchCompleted {
			continue
		}

		// Get standings for both participants
		var p1Standing, p2Standing *domain.GroupStanding
		if m.Participant1ID != nil {
			p1Standing = standingMap[*m.Participant1ID]
		}
		if m.Participant2ID != nil {
			p2Standing = standingMap[*m.Participant2ID]
		}

		// Calculate set wins/losses and points from sets
		p1Sets, p2Sets := 0, 0
		p1Points, p2Points := 0, 0
		for _, set := range m.Sets {
			if set.Participant1Score > set.Participant2Score {
				p1Sets++
			} else if set.Participant2Score > set.Participant1Score {
				p2Sets++
			}
			p1Points += set.Participant1Score
			p2Points += set.Participant2Score
		}

		// Update standings
		if m.WinnerID != nil {
			if p1Standing != nil && *m.WinnerID == *m.Participant1ID {
				p1Standing.MatchWins++
				if p2Standing != nil {
					p2Standing.MatchLosses++
				}
			} else if p2Standing != nil && m.Participant2ID != nil && *m.WinnerID == *m.Participant2ID {
				p2Standing.MatchWins++
				if p1Standing != nil {
					p1Standing.MatchLosses++
				}
			}
		}

		if p1Standing != nil {
			p1Standing.SetWins += p1Sets
			p1Standing.SetLosses += p2Sets
			p1Standing.PointsScored += p1Points
			p1Standing.PointsAgainst += p2Points
		}
		if p2Standing != nil {
			p2Standing.SetWins += p2Sets
			p2Standing.SetLosses += p1Sets
			p2Standing.PointsScored += p2Points
			p2Standing.PointsAgainst += p1Points
		}
	}

	// Calculate opponent match wins (Buchholz tiebreaker)
	for _, m := range matches {
		if m.Status != domain.MatchCompleted {
			continue
		}

		var p1Standing, p2Standing *domain.GroupStanding
		if m.Participant1ID != nil {
			p1Standing = standingMap[*m.Participant1ID]
		}
		if m.Participant2ID != nil {
			p2Standing = standingMap[*m.Participant2ID]
		}

		if p1Standing != nil && p2Standing != nil {
			p1Standing.OpponentMatchWins += p2Standing.MatchWins
			p2Standing.OpponentMatchWins += p1Standing.MatchWins
		}
	}

	// Sort standings by criteria (or bracket placement for elimination formats)
	useBracketPlacement := len(criteria) == 0 &&
		(format == "single_elimination" || format == "double_elimination")

	if useBracketPlacement {
		SortStandingsByBracketPlacement(standings)
	} else {
		sortStandingsByCriteria(standings, criteria)
	}

	// Assign ranks
	for i, s := range standings {
		rank := i + 1
		s.GroupRank = &rank
	}

	return standings
}

// sortStandingsByCriteria sorts standings using the provided criteria in priority order
func sortStandingsByCriteria(standings []*domain.GroupStanding, criteria []domain.RankingCriterion) {
	// If no criteria provided, use default order for Swiss format
	// For elimination formats, empty criteria means use bracket placement
	// (higher wins + lower losses = better bracket position)
	if len(criteria) == 0 {
		criteria = []domain.RankingCriterion{
			domain.CriterionMatchWins,
			domain.CriterionSetDifferential,
			domain.CriterionPointsDifferential,
		}
	}

	sort.SliceStable(standings, func(i, j int) bool {
		a, b := standings[i], standings[j]

		for _, criterion := range criteria {
			cmp := domain.CompareStandings(a, b, criterion)
			if cmp != 0 {
				return cmp > 0 // Higher is better
			}
		}

		// Final tiebreaker: original seed (lower is better)
		return a.Seed < b.Seed
	})
}

// SortStandingsByBracketPlacement sorts standings based on elimination bracket placement.
// This is used when ranking criteria is empty for single/double elimination formats.
// Rankings are determined by: most wins, then fewest losses, then seed as tiebreaker.
// This approximates bracket position since the winner has most wins and fewest losses.
func SortStandingsByBracketPlacement(standings []*domain.GroupStanding) {
	sort.SliceStable(standings, func(i, j int) bool {
		a, b := standings[i], standings[j]

		// Primary: most match wins (higher = further in bracket)
		if a.MatchWins != b.MatchWins {
			return a.MatchWins > b.MatchWins
		}

		// Secondary: fewest match losses (lower = better; e.g., 2-0 > 2-1)
		if a.MatchLosses != b.MatchLosses {
			return a.MatchLosses < b.MatchLosses
		}

		// Tertiary: set differential as tiebreaker
		aDiff := a.SetWins - a.SetLosses
		bDiff := b.SetWins - b.SetLosses
		if aDiff != bDiff {
			return aDiff > bDiff
		}

		// Final: original seed (lower is better)
		return a.Seed < b.Seed
	})
}

// AggregateStageStandings creates cross-group standings for determining advancement
// It takes group standings from all groups and determines who advances based on:
// 1. Group rank (1st place from each group, then 2nd place, etc.)
// 2. Within same group rank, compare by stats using criteria
func AggregateStageStandings(
	groupStandings map[uint64][]*domain.GroupStanding, // groupID -> standings
	advancingPerGroup int,
	criteria []domain.RankingCriterion,
) []*domain.StageStanding {
	var allStandings []*domain.StageStanding

	// Collect all group standings and convert to stage standings
	for groupID, standings := range groupStandings {
		for _, gs := range standings {
			groupRank := 0
			if gs.GroupRank != nil {
				groupRank = *gs.GroupRank
			}

			ss := &domain.StageStanding{
				TournamentID:  gs.TournamentID,
				StageID:       gs.StageID,
				ParticipantID: gs.ParticipantID,
				GroupID:       groupID,
				GroupRank:     groupRank,
				MatchWins:     gs.MatchWins,
				MatchLosses:   gs.MatchLosses,
				SetWins:       gs.SetWins,
				SetLosses:     gs.SetLosses,
				PointsScored:  gs.PointsScored,
				PointsAgainst: gs.PointsAgainst,
				Advances:      groupRank > 0 && groupRank <= advancingPerGroup,
			}
			allStandings = append(allStandings, ss)
		}
	}

	// Sort by group rank first, then by stats for tiebreaking within same rank
	sort.SliceStable(allStandings, func(i, j int) bool {
		a, b := allStandings[i], allStandings[j]

		// Primary: group rank (lower is better)
		if a.GroupRank != b.GroupRank {
			return a.GroupRank < b.GroupRank
		}

		// Secondary: compare by criteria
		for _, criterion := range criteria {
			// Convert to GroupStanding for comparison
			aGS := &domain.GroupStanding{
				MatchWins:     a.MatchWins,
				MatchLosses:   a.MatchLosses,
				SetWins:       a.SetWins,
				SetLosses:     a.SetLosses,
				PointsScored:  a.PointsScored,
				PointsAgainst: a.PointsAgainst,
			}
			bGS := &domain.GroupStanding{
				MatchWins:     b.MatchWins,
				MatchLosses:   b.MatchLosses,
				SetWins:       b.SetWins,
				SetLosses:     b.SetLosses,
				PointsScored:  b.PointsScored,
				PointsAgainst: b.PointsAgainst,
			}
			cmp := domain.CompareStandings(aGS, bGS, criterion)
			if cmp != 0 {
				return cmp > 0 // Higher is better
			}
		}

		return false
	})

	// Apply cross-group seeding for bracket half distribution
	applyCrossGroupSeeding(allStandings)

	return allStandings
}

// applyCrossGroupSeeding reorders stage standings to ensure cross-group distribution
// across bracket halves. Uses serpentine seeding at the rank-tier level.
//
// The standard bracket seeding maps seeds to halves like this (for 8-team bracket):
// - Top half: seeds 1, 4, 5, 8
// - Bottom half: seeds 2, 3, 6, 7
//
// To achieve cross-seeding (participants from same group on opposite halves), we apply
// serpentine ordering every 2 tiers (super-tiers):
// - Super-tier 0 (tiers 0-1): normal group order
// - Super-tier 1 (tiers 2-3): reversed group order
// - Alternates for subsequent super-tiers
//
// This ensures that 2nd/3rd place finishers are on opposite bracket halves from their
// own group's 1st place finisher, minimizing early rematches.
func applyCrossGroupSeeding(standings []*domain.StageStanding) {
	if len(standings) == 0 {
		return
	}

	// Count unique groups
	groupSet := make(map[uint64]bool)
	for _, s := range standings {
		groupSet[s.GroupID] = true
	}
	numGroups := len(groupSet)

	if numGroups < 2 {
		// No cross-seeding needed for single group, just assign ranks
		for i, ss := range standings {
			rank := i + 1
			ss.StageRank = &rank
		}
		return
	}

	// Group standings by rank tier (group rank)
	tiers := make(map[int][]*domain.StageStanding)
	for _, s := range standings {
		tiers[s.GroupRank] = append(tiers[s.GroupRank], s)
	}

	// Get sorted tier numbers
	tierNums := make([]int, 0, len(tiers))
	for t := range tiers {
		tierNums = append(tierNums, t)
	}
	sort.Ints(tierNums)

	// Rebuild standings with serpentine ordering
	result := make([]*domain.StageStanding, 0, len(standings))

	for i, tierNum := range tierNums {
		tier := tiers[tierNum]

		// Sort tier by group ID to get consistent ordering
		sort.SliceStable(tier, func(a, b int) bool {
			return tier[a].GroupID < tier[b].GroupID
		})

		// Apply serpentine: reverse order for odd super-tiers (every 2 tiers)
		superTier := i / 2
		if superTier%2 == 1 {
			// Reverse the tier
			for left, right := 0, len(tier)-1; left < right; left, right = left+1, right-1 {
				tier[left], tier[right] = tier[right], tier[left]
			}
		}

		result = append(result, tier...)
	}

	// Copy back to original slice and reassign stage ranks
	for i, s := range result {
		standings[i] = s
		rank := i + 1
		standings[i].StageRank = &rank
	}
}

// InitializeGroupStandings creates initial standings for a group
func InitializeGroupStandings(
	tournamentID, stageID, groupID uint64,
	participants []domain.Participant,
) []*domain.GroupStanding {
	standings := make([]*domain.GroupStanding, len(participants))

	for i, p := range participants {
		standings[i] = &domain.GroupStanding{
			TournamentID:    tournamentID,
			StageID:         stageID,
			GroupID:         groupID,
			ParticipantID:   p.ID,
			ParticipantName: p.Name,
			ParticipantIconURL: func() *string {
				if p.IconURL != "" {
					return &p.IconURL
				}
				return nil
			}(),
			Seed:        p.Seed,
			MatchWins:   0,
			MatchLosses: 0,
			SetWins:     0,
			SetLosses:   0,
		}
	}

	return standings
}
