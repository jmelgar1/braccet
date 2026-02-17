package engine

// CalculateBracketSize returns the smallest power of 2 >= participants.
func CalculateBracketSize(participants int) int {
	if participants <= 0 {
		return 0
	}
	size := 1
	for size < participants {
		size *= 2
	}
	return size
}

// GenerateSeedPairings returns seed pairings for a bracket.
// Uses standard tournament seeding where top seeds meet in later rounds.
// For an 8-player bracket: [[1,8], [4,5], [2,7], [3,6]]
// This ensures 1 vs 2 can only happen in the finals.
func GenerateSeedPairings(bracketSize int) [][2]int {
	if bracketSize < 2 {
		return nil
	}

	// Build pairings recursively using the standard algorithm:
	// Start with [1, 2] for a 2-player bracket
	// For each doubling, mirror and interleave with new seeds
	return buildPairings(bracketSize)
}

func buildPairings(size int) [][2]int {
	if size == 2 {
		return [][2]int{{1, 2}}
	}

	// Get pairings for half-size bracket
	smaller := buildPairings(size / 2)

	// For each match in smaller bracket, create two matches
	// Top half keeps the original seed, pairs with (size+1 - original opponent)
	// This maintains proper seeding where seed N faces seed (size+1-N)
	result := make([][2]int, len(smaller)*2)
	for i, pair := range smaller {
		// First match: original seed vs its proper opponent for this bracket size
		result[i*2] = [2]int{pair[0], size + 1 - pair[0]}
		// Second match: original opponent vs its proper opponent
		result[i*2+1] = [2]int{pair[1], size + 1 - pair[1]}
	}

	return result
}

// TotalRounds returns the number of rounds needed for a bracket of given size.
func TotalRounds(bracketSize int) int {
	if bracketSize <= 1 {
		return 0
	}
	rounds := 0
	size := bracketSize
	for size > 1 {
		rounds++
		size /= 2
	}
	return rounds
}

// MatchesInRound returns the number of matches in a given round (1-indexed).
// Round 1 has bracketSize/2 matches, each subsequent round has half.
func MatchesInRound(bracketSize, round int) int {
	if round < 1 || bracketSize < 2 {
		return 0
	}
	matches := bracketSize / 2
	for i := 1; i < round; i++ {
		matches /= 2
	}
	return matches
}
