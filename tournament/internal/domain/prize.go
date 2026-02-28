package domain

import (
	"fmt"
	"math"
)

// SwissRecordDistribution holds the player count for a specific win-loss record
type SwissRecordDistribution struct {
	Wins       int
	Losses     int
	Count      int
	IsAdvanced bool
}

// DoubleElimRecordDistribution holds the player count for a specific win-loss record in double elimination.
// All eliminated players have exactly 2 losses; wins vary based on how far they progressed.
type DoubleElimRecordDistribution struct {
	Wins       int
	Losses     int // Always 2 for eliminated players
	Count      int
	IsAdvanced bool
}

// binomialCoeff calculates "n choose k" = n! / (k! * (n-k)!)
func binomialCoeff(n, k int) int {
	if k > n || k < 0 {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}
	// Use symmetry property for efficiency
	if k > n-k {
		k = n - k
	}
	result := 1
	for i := 0; i < k; i++ {
		result = result * (n - i) / (i + 1)
	}
	return result
}

// calculateSwissDistribution calculates how many players will finish at each record
// in a Swiss tournament with the given thresholds.
// Uses the formula: players at record (w, l) = C(w+l-1, w) * n / 2^(w+l)
func calculateSwissDistribution(n, winsToAdvance, lossesToEliminate int) []SwissRecordDistribution {
	var distribution []SwissRecordDistribution

	// Calculate advanced records: W-0, W-1, W-2, ..., W-(L-1)
	// These are players who reached W wins before L losses
	for j := 0; j < lossesToEliminate; j++ {
		rounds := winsToAdvance + j
		coeff := binomialCoeff(rounds-1, j)
		count := float64(coeff) * float64(n) / math.Pow(2, float64(rounds))
		distribution = append(distribution, SwissRecordDistribution{
			Wins:       winsToAdvance,
			Losses:     j,
			Count:      int(math.Round(count)),
			IsAdvanced: true,
		})
	}

	// Calculate eliminated records: (W-1)-L, (W-2)-L, ..., 1-L, 0-L
	// These are players who reached L losses before W wins
	// Order from best eliminated (most wins) to worst eliminated (least wins)
	for k := winsToAdvance - 1; k >= 0; k-- {
		rounds := k + lossesToEliminate
		coeff := binomialCoeff(rounds-1, k)
		count := float64(coeff) * float64(n) / math.Pow(2, float64(rounds))
		distribution = append(distribution, SwissRecordDistribution{
			Wins:       k,
			Losses:     lossesToEliminate,
			Count:      int(math.Round(count)),
			IsAdvanced: false,
		})
	}

	// Adjust counts to ensure total equals n (handle rounding errors)
	adjustDistributionToTotal(distribution, n)

	return distribution
}

// adjustDistributionToTotal ensures the sum of counts equals the total participants
func adjustDistributionToTotal(dist []SwissRecordDistribution, total int) {
	sum := 0
	for _, d := range dist {
		sum += d.Count
	}

	diff := total - sum

	// Distribute the difference across groups (rounding errors should be small)
	for diff != 0 {
		if diff > 0 {
			// Find largest group and add 1
			maxIdx := 0
			for i := range dist {
				if dist[i].Count > dist[maxIdx].Count {
					maxIdx = i
				}
			}
			dist[maxIdx].Count++
			diff--
		} else {
			// Find largest group with count > 0 and subtract 1
			maxIdx := -1
			for i := range dist {
				if dist[i].Count > 0 && (maxIdx == -1 || dist[i].Count > dist[maxIdx].Count) {
					maxIdx = i
				}
			}
			if maxIdx >= 0 {
				dist[maxIdx].Count--
				diff++
			} else {
				break // Can't reduce further
			}
		}
	}
}

// calculateBracketSize returns the smallest power of 2 >= n (for bracket sizing)
func calculateBracketSize(n int) int {
	if n <= 1 {
		return 1
	}
	size := 1
	for size < n {
		size *= 2
	}
	return size
}

// calculateDoubleElimDistribution calculates how many players finish with each W-L record
// in a double elimination bracket. All eliminated players have exactly 2 losses.
// The advancing parameter specifies how many players advance (0 = all finish, used for finals).
// Byes do NOT count as wins.
func calculateDoubleElimDistribution(participants, advancing int) []DoubleElimRecordDistribution {
	if participants < 2 {
		return nil
	}
	return simulateDoubleElimRecords(participants, advancing)
}

// simulateDoubleElimRecords simulates a double elimination bracket and returns the distribution
// of final W-L records for eliminated players.
func simulateDoubleElimRecords(participants, advancing int) []DoubleElimRecordDistribution {
	if participants < 2 {
		return nil
	}

	bracketSize := calculateBracketSize(participants)
	byes := bracketSize - participants

	// Track eliminated players by their win count
	eliminatedByWins := make(map[int]int)

	// Calculate winners bracket rounds
	winnersRounds := 0
	for size := bracketSize; size > 1; size /= 2 {
		winnersRounds++
	}

	// Losers bracket has 2*(winnersRounds-1) rounds
	losersRounds := 2 * (winnersRounds - 1)

	// In double elimination:
	// - All eliminated players have exactly 2 losses
	// - Their win count depends on how far they got in losers bracket
	// - Minimum wins = floor((losersRound-1)/2) for round where they were eliminated

	currentEliminated := 0
	targetEliminated := participants - advancing

	// When advancing=0 (finals), champion and runner-up are not "eliminated" with 2 losses
	// So we need to subtract 2 from the target
	if advancing == 0 && participants >= 2 {
		targetEliminated = participants - 2 // Exclude champion and runner-up
	}

	// Calculate eliminations per losers bracket round
	for round := 1; round <= losersRounds && currentEliminated < targetEliminated; round++ {
		var roundEliminations int

		// Major rounds (odd): losers play losers
		// Minor rounds (even): losers play drop-ins from winners

		divisor := 1 << ((round + 1) / 2) // 2^ceil(round/2)
		roundEliminations = bracketSize / (2 * divisor)

		// Adjust for byes - byes reduce early round eliminations
		if round <= 2 && byes > 0 {
			// Early rounds affected by byes
			byeAdjustment := byes / (1 << round)
			roundEliminations -= byeAdjustment
			if roundEliminations < 0 {
				roundEliminations = 0
			}
		}

		// Don't exceed target eliminated
		if currentEliminated+roundEliminations > targetEliminated {
			roundEliminations = targetEliminated - currentEliminated
		}

		// Calculate win count for this round
		// Minimum wins = matches won in losers bracket = floor((round-1)/2)
		minWins := (round - 1) / 2

		// Players may also have wins from winners bracket before dropping
		// Average additional wins from WB depends on which WR they dropped from

		// For simplicity, use the minimum wins for the tier
		// This groups all eliminated in this LR round together
		winCount := minWins

		eliminatedByWins[winCount] += roundEliminations
		currentEliminated += roundEliminations
	}

	// If we haven't reached target eliminated, add remaining to highest win count
	if currentEliminated < targetEliminated {
		remaining := targetEliminated - currentEliminated
		// These would be the semi-finalists/finalists level
		maxWins := (losersRounds - 1) / 2
		eliminatedByWins[maxWins] += remaining
	}

	// Convert to distribution slice, sorted by wins descending (best first)
	var distribution []DoubleElimRecordDistribution

	// Find max wins
	maxWins := 0
	for wins := range eliminatedByWins {
		if wins > maxWins {
			maxWins = wins
		}
	}

	// Add in descending order
	for wins := maxWins; wins >= 0; wins-- {
		count := eliminatedByWins[wins]
		if count > 0 {
			distribution = append(distribution, DoubleElimRecordDistribution{
				Wins:       wins,
				Losses:     2,
				Count:      count,
				IsAdvanced: false,
			})
		}
	}

	// Adjust total to match expected eliminated count
	adjustDoubleElimDistributionToTotal(distribution, targetEliminated)

	return distribution
}

// adjustDoubleElimDistributionToTotal ensures the sum of counts equals the expected eliminated count
func adjustDoubleElimDistributionToTotal(dist []DoubleElimRecordDistribution, total int) {
	if len(dist) == 0 {
		return
	}

	sum := 0
	for _, d := range dist {
		sum += d.Count
	}

	diff := total - sum

	for diff != 0 {
		if diff > 0 {
			// Add to largest group
			maxIdx := 0
			for i := range dist {
				if dist[i].Count > dist[maxIdx].Count {
					maxIdx = i
				}
			}
			dist[maxIdx].Count++
			diff--
		} else {
			// Subtract from largest group with count > 1
			maxIdx := -1
			for i := range dist {
				if dist[i].Count > 1 && (maxIdx == -1 || dist[i].Count > dist[maxIdx].Count) {
					maxIdx = i
				}
			}
			if maxIdx >= 0 {
				dist[maxIdx].Count--
				diff++
			} else {
				break
			}
		}
	}
}

// TierGenerationConfig provides parameters for generating placement tiers
// that account for tournament format and configuration
type TierGenerationConfig struct {
	Format            TournamentFormat
	ParticipantCount  int
	WinsToAdvance     *int              // Swiss threshold mode
	LossesToEliminate *int              // Swiss threshold mode
	Stages            []StageTierConfig // For multi-stage tournaments
}

// StageTierConfig describes a stage for tier calculation in multi-stage tournaments
type StageTierConfig struct {
	StageOrder        int
	StageType         StageType
	Format            TournamentFormat
	Participants      int  // How many enter this stage
	Advancing         int  // How many advance (0 for final stage)
	WinsToAdvance     *int // Swiss threshold mode
	LossesToEliminate *int // Swiss threshold mode
}

// GeneratePlacementTiersConfig generates placement tiers based on tournament configuration.
// It handles different formats appropriately:
// - Single/Double Elimination: Power-of-2 tiers (1st, 2nd, 3rd-4th, 5th-8th, etc.)
// - Swiss Standard: Same as elimination (grouped tiers)
// - Swiss Threshold: Split into Advanced and Eliminated groups with sub-tiers
// - Multi-stage: Tiers for final stage participants, then tiers for group stage eliminated
func GeneratePlacementTiersConfig(config TierGenerationConfig) []PlacementTier {
	if config.ParticipantCount < 2 {
		return []PlacementTier{}
	}

	// Multi-stage takes precedence
	if len(config.Stages) > 0 {
		return generateMultiStageTiers(config.Stages, config.ParticipantCount)
	}

	// Swiss threshold mode
	if config.Format == FormatSwiss && config.WinsToAdvance != nil && config.LossesToEliminate != nil {
		return generateSwissThresholdTiers(config.ParticipantCount, *config.WinsToAdvance, *config.LossesToEliminate)
	}

	// Standard formats (single elim, double elim, swiss standard)
	// All use power-of-2 boundaries
	return GeneratePlacementTiers(config.ParticipantCount)
}

// generateSwissThresholdTiers creates tiers for Swiss tournaments with early advance/eliminate thresholds.
// Tiers are grouped by final record (e.g., 3-0, 3-1, 3-2 for advanced; 2-3, 1-3, 0-3 for eliminated).
// This uses Swiss bracket mathematics to calculate the expected player count at each record.
func generateSwissThresholdTiers(participantCount, winsToAdvance, lossesToEliminate int) []PlacementTier {
	if participantCount < 2 {
		return []PlacementTier{}
	}

	// Calculate distribution of players at each record
	distribution := calculateSwissDistribution(participantCount, winsToAdvance, lossesToEliminate)

	var tiers []PlacementTier
	currentPosition := 1

	for _, record := range distribution {
		if record.Count == 0 {
			continue
		}

		endPosition := currentPosition + record.Count - 1

		// Create tier for this record group
		tier := PlacementTier{
			Placement: formatPlacementRange(currentPosition, endPosition),
			Low:       currentPosition,
			High:      endPosition,
		}
		tiers = append(tiers, tier)

		currentPosition = endPosition + 1
	}

	return tiers
}

// generateSwissEliminatedTiers creates tiers for eliminated players in a Swiss stage.
// Only includes records that result in elimination (reaching lossesToEliminate before winsToAdvance).
func generateSwissEliminatedTiers(startPos, participants, winsToAdvance, lossesToEliminate int) []PlacementTier {
	// Get the full distribution
	distribution := calculateSwissDistribution(participants, winsToAdvance, lossesToEliminate)

	var tiers []PlacementTier
	currentPos := startPos

	// Only process eliminated records (IsAdvanced == false)
	for _, record := range distribution {
		if record.IsAdvanced || record.Count == 0 {
			continue
		}

		endPos := currentPos + record.Count - 1
		tier := PlacementTier{
			Placement: formatPlacementRange(currentPos, endPos),
			Low:       currentPos,
			High:      endPos,
		}
		tiers = append(tiers, tier)
		currentPos = endPos + 1
	}

	return tiers
}

// generateDoubleElimEliminatedTiers creates tiers for eliminated players in a double elimination stage.
// Groups players by their final W-L record (all eliminated players have 2 losses).
func generateDoubleElimEliminatedTiers(startPos, participants, advancing int) []PlacementTier {
	distribution := calculateDoubleElimDistribution(participants, advancing)

	var tiers []PlacementTier
	currentPos := startPos

	for _, record := range distribution {
		if record.IsAdvanced || record.Count == 0 {
			continue
		}

		endPos := currentPos + record.Count - 1
		tier := PlacementTier{
			Placement: formatPlacementRange(currentPos, endPos),
			Low:       currentPos,
			High:      endPos,
		}
		tiers = append(tiers, tier)
		currentPos = endPos + 1
	}

	return tiers
}

// estimateSwissAdvancing estimates how many participants will advance in a Swiss threshold system.
// Uses the ratio of losses_to_eliminate / (wins_to_advance + losses_to_eliminate) as a probability estimate.
// Rounds to nearest power of 2 for cleaner tier boundaries.
func estimateSwissAdvancing(n, winsToAdvance, lossesToEliminate int) int {
	if winsToAdvance <= 0 || lossesToEliminate <= 0 {
		return n / 2 // Default to half
	}

	// Probability of advancing ≈ L / (W + L)
	// For W=3, L=3: 50% advance
	// For W=2, L=3: 60% advance (easier to advance)
	// For W=3, L=2: 40% advance (harder to advance)
	advanceRatio := float64(lossesToEliminate) / float64(winsToAdvance+lossesToEliminate)
	estimated := int(float64(n) * advanceRatio)

	// Clamp to reasonable bounds
	if estimated < 1 {
		estimated = 1
	}
	if estimated > n-1 {
		estimated = n - 1
	}

	// Round to nearest power of 2 for cleaner tier boundaries
	return nearestPowerOf2(estimated)
}

// nearestPowerOf2 returns the power of 2 closest to n
func nearestPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	// Find lower and upper power of 2 bounds
	lower := 1
	for lower*2 <= n {
		lower *= 2
	}
	upper := lower * 2

	// Return whichever is closer
	if n-lower < upper-n {
		return lower
	}
	return upper
}

// generateMultiStageTiers creates tiers for multi-stage tournaments.
// Final stage participants get top placements, group stage eliminated get lower placements.
func generateMultiStageTiers(stages []StageTierConfig, totalParticipants int) []PlacementTier {
	if len(stages) == 0 {
		return GeneratePlacementTiers(totalParticipants)
	}

	// Sort stages: final (order=0) first for top placements, then group stages in descending order
	// This way: finals get 1st-Nth, last group stage eliminated get next tier, etc.
	sortedStages := sortStagesForTiers(stages)

	var allTiers []PlacementTier
	currentPosition := 1

	for _, stage := range sortedStages {
		if stage.StageType == StageTypeFinal {
			// Final stage: standard tier generation for finalists
			finalTiers := generateTiersForStageFormat(
				stage.Format,
				currentPosition,
				stage.Participants,
				stage.WinsToAdvance,
				stage.LossesToEliminate,
			)
			allTiers = append(allTiers, finalTiers...)
			currentPosition += stage.Participants
		} else {
			// Group stage: tiers for eliminated participants
			eliminated := stage.Participants - stage.Advancing
			if eliminated > 0 {
				var eliminatedTiers []PlacementTier

				// Use record-based tiers for Swiss stages with threshold mode
				if stage.Format == FormatSwiss && stage.WinsToAdvance != nil && stage.LossesToEliminate != nil {
					eliminatedTiers = generateSwissEliminatedTiers(
						currentPosition,
						stage.Participants,
						*stage.WinsToAdvance,
						*stage.LossesToEliminate,
					)
				} else if stage.Format == FormatDoubleElimination {
					// Double elimination: group by W-L record
					eliminatedTiers = generateDoubleElimEliminatedTiers(
						currentPosition,
						stage.Participants,
						stage.Advancing,
					)
				} else {
					// Default to power-of-2 tiers for other formats (single elim, standard swiss)
					eliminatedTiers = generatePowerOf2TiersRange(currentPosition, currentPosition+eliminated-1)
				}

				allTiers = append(allTiers, eliminatedTiers...)
				currentPosition += eliminated
			}
		}
	}

	return allTiers
}

// sortStagesForTiers sorts stages for tier generation:
// Final stage first (order=0), then group stages in descending order
// This ensures top placements go to finalists, then to later group stage eliminated, etc.
func sortStagesForTiers(stages []StageTierConfig) []StageTierConfig {
	sorted := make([]StageTierConfig, len(stages))
	copy(sorted, stages)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			// Final stage (order=0, type=final) should come first
			// Then group stages in descending order (higher order = later stage = better placement)
			shouldSwap := false

			if sorted[j].StageType == StageTypeFinal && sorted[j+1].StageType == StageTypeFinal {
				shouldSwap = false // both finals, keep order
			} else if sorted[j].StageType == StageTypeFinal {
				shouldSwap = false // j is final, should stay first
			} else if sorted[j+1].StageType == StageTypeFinal {
				shouldSwap = true // j+1 is final, should come first
			} else {
				// Both are group stages - higher order = better placements = should come first
				shouldSwap = sorted[j].StageOrder < sorted[j+1].StageOrder
			}

			if shouldSwap {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// generateTiersForStageFormat generates tiers for a specific stage format
func generateTiersForStageFormat(format TournamentFormat, startPos, count int, winsToAdvance, lossesToEliminate *int) []PlacementTier {
	if format == FormatSwiss && winsToAdvance != nil && lossesToEliminate != nil {
		// Swiss threshold within stage - use record-based tiers
		distribution := calculateSwissDistribution(count, *winsToAdvance, *lossesToEliminate)

		var tiers []PlacementTier
		currentPos := startPos

		for _, record := range distribution {
			if record.Count == 0 {
				continue
			}

			endPos := currentPos + record.Count - 1
			tier := PlacementTier{
				Placement: formatPlacementRange(currentPos, endPos),
				Low:       currentPos,
				High:      endPos,
			}
			tiers = append(tiers, tier)
			currentPos = endPos + 1
		}

		return tiers
	}

	if format == FormatDoubleElimination {
		// Double elimination finals: group by W-L record
		// For finals, all participants finish (advancing=0)
		distribution := calculateDoubleElimDistribution(count, 0)

		var tiers []PlacementTier
		currentPos := startPos

		// First, add champion and finalist tiers (they don't have 2 losses)
		// Champion: 1 player
		tiers = append(tiers, PlacementTier{
			Placement: ordinal(currentPos),
			Low:       currentPos,
			High:      currentPos,
		})
		currentPos++

		// Runner-up: 1 player
		if count >= 2 {
			tiers = append(tiers, PlacementTier{
				Placement: ordinal(currentPos),
				Low:       currentPos,
				High:      currentPos,
			})
			currentPos++
		}

		// Eliminated players grouped by record
		for _, record := range distribution {
			if record.Count == 0 {
				continue
			}

			endPos := currentPos + record.Count - 1
			tier := PlacementTier{
				Placement: formatPlacementRange(currentPos, endPos),
				Low:       currentPos,
				High:      endPos,
			}
			tiers = append(tiers, tier)
			currentPos = endPos + 1
		}

		return tiers
	}

	// Standard power-of-2 tiers (single elimination, standard swiss)
	return generatePowerOf2TiersRange(startPos, startPos+count-1)
}

// generatePowerOf2TiersRange generates power-of-2 based tiers for positions startPos through endPos.
// For example, startPos=17, endPos=32 would generate tiers like 17th-24th, 25th-32nd.
func generatePowerOf2TiersRange(startPos, endPos int) []PlacementTier {
	if startPos > endPos || startPos < 1 {
		return []PlacementTier{}
	}

	count := endPos - startPos + 1

	// Handle small ranges directly
	if count == 1 {
		return []PlacementTier{{
			Placement: ordinal(startPos),
			Low:       startPos,
			High:      startPos,
		}}
	}

	if count == 2 {
		return []PlacementTier{
			{Placement: ordinal(startPos), Low: startPos, High: startPos},
			{Placement: ordinal(endPos), Low: endPos, High: endPos},
		}
	}

	// For larger ranges, use power-of-2 groupings
	// Generate base tiers as if starting from 1
	baseTiers := GeneratePlacementTiers(count)

	// Shift positions to start from startPos
	for i := range baseTiers {
		baseTiers[i].Low += startPos - 1
		baseTiers[i].High += startPos - 1
		baseTiers[i].Placement = formatPlacementRange(baseTiers[i].Low, baseTiers[i].High)
	}

	return baseTiers
}

// formatPlacementRange creates a placement string like "1st", "3rd-4th", "17th-24th"
func formatPlacementRange(low, high int) string {
	if low == high {
		return ordinal(low)
	}
	return fmt.Sprintf("%s-%s", ordinal(low), ordinal(high))
}

// GeneratePlacementTiers creates standard placement tiers for a given participant count.
// The tiers follow standard elimination bracket patterns:
// - 8 participants: 1st, 2nd, 3rd-4th, 5th-8th
// - 16 participants: 1st, 2nd, 3rd-4th, 5th-8th, 9th-16th
// - etc.
func GeneratePlacementTiers(participantCount int) []PlacementTier {
	if participantCount < 2 {
		return []PlacementTier{}
	}

	tiers := []PlacementTier{
		{Placement: "1st", Low: 1, High: 1},
	}

	if participantCount >= 2 {
		tiers = append(tiers, PlacementTier{Placement: "2nd", Low: 2, High: 2})
	}

	// Generate remaining tiers based on elimination rounds
	// Pattern: 3-4, 5-8, 9-16, 17-32, etc.
	low := 3
	for low <= participantCount {
		// Calculate the upper bound (next power of 2)
		high := nextPowerOf2(low)
		if high > participantCount {
			high = participantCount
		}

		tier := PlacementTier{Low: low, High: high}
		if low == high {
			tier.Placement = ordinal(low)
		} else {
			tier.Placement = fmt.Sprintf("%s-%s", ordinal(low), ordinal(high))
		}
		tiers = append(tiers, tier)
		low = high + 1
	}

	return tiers
}

// nextPowerOf2 returns the smallest power of 2 that is >= n*2
// For n=3, returns 8 (so high becomes 7, giving range 3-4 would be wrong)
// Actually we want: for low=3, high=4; for low=5, high=8; for low=9, high=16
func nextPowerOf2(n int) int {
	// Find the power of 2 that this range ends at
	// 3-4: ends at 4 (2^2)
	// 5-8: ends at 8 (2^3)
	// 9-16: ends at 16 (2^4)
	power := 2
	for power < n {
		power *= 2
	}
	return power
}

// ordinal converts an integer to its ordinal string representation
// 1 -> "1st", 2 -> "2nd", 3 -> "3rd", 4 -> "4th", etc.
func ordinal(n int) string {
	suffix := "th"
	switch n % 10 {
	case 1:
		if n%100 != 11 {
			suffix = "st"
		}
	case 2:
		if n%100 != 12 {
			suffix = "nd"
		}
	case 3:
		if n%100 != 13 {
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}
