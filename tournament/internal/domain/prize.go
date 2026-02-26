package domain

import "fmt"

// GeneratePlacementTiers creates standard placement tiers for a given participant count.
// The tiers follow standard elimination bracket patterns:
// - 8 participants: 1st, 2nd, 3rd-4th, 5th-8th
// - 16 participants: 1st, 2nd, 3rd-4th, 5th-8th, 9th-16th
// - etc.
func GeneratePlacementTiers(participantCount int) []PlacementTier {
	if participantCount < 2 {
		return nil
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
		// Calculate the upper bound (next power of 2 minus 1)
		high := nextPowerOf2(low) - 1
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
