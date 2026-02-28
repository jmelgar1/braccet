package domain

import (
	"testing"
)

func TestCalculateSwissDistribution(t *testing.T) {
	// 16-player Swiss with 3W/3L
	dist := calculateSwissDistribution(16, 3, 3)

	t.Log("Distribution for 16-player Swiss 3W/3L:")
	total := 0
	for _, d := range dist {
		t.Logf("  %d-%d: %d players (advanced=%v)", d.Wins, d.Losses, d.Count, d.IsAdvanced)
		total += d.Count
	}
	t.Logf("  Total: %d", total)

	// Verify total equals participant count
	if total != 16 {
		t.Errorf("Total players %d != 16", total)
	}

	// Expected distribution:
	// 3-0: 2, 3-1: 3, 3-2: 3 (advanced)
	// 2-3: 3, 1-3: 3, 0-3: 2 (eliminated)
	expected := []struct {
		wins, losses, count int
		advanced            bool
	}{
		{3, 0, 2, true},
		{3, 1, 3, true},
		{3, 2, 3, true},
		{2, 3, 3, false},
		{1, 3, 3, false},
		{0, 3, 2, false},
	}

	if len(dist) != len(expected) {
		t.Errorf("Expected %d records, got %d", len(expected), len(dist))
	}

	for i, exp := range expected {
		if i >= len(dist) {
			break
		}
		d := dist[i]
		if d.Wins != exp.wins || d.Losses != exp.losses {
			t.Errorf("Record %d: expected %d-%d, got %d-%d", i, exp.wins, exp.losses, d.Wins, d.Losses)
		}
		if d.Count != exp.count {
			t.Errorf("Record %d-%d: expected %d players, got %d", d.Wins, d.Losses, exp.count, d.Count)
		}
		if d.IsAdvanced != exp.advanced {
			t.Errorf("Record %d-%d: expected advanced=%v, got %v", d.Wins, d.Losses, exp.advanced, d.IsAdvanced)
		}
	}
}

func TestGenerateSwissThresholdTiers(t *testing.T) {
	// 16-player Swiss with 3W/3L
	tiers := generateSwissThresholdTiers(16, 3, 3)

	t.Log("Tiers for 16-player Swiss 3W/3L:")
	for _, tier := range tiers {
		t.Logf("  %s (low=%d, high=%d)", tier.Placement, tier.Low, tier.High)
	}

	// Expected tiers:
	// 1st-2nd (3-0), 3rd-5th (3-1), 6th-8th (3-2)
	// 9th-11th (2-3), 12th-14th (1-3), 15th-16th (0-3)
	expected := []struct {
		placement string
		low, high int
	}{
		{"1st-2nd", 1, 2},
		{"3rd-5th", 3, 5},
		{"6th-8th", 6, 8},
		{"9th-11th", 9, 11},
		{"12th-14th", 12, 14},
		{"15th-16th", 15, 16},
	}

	if len(tiers) != len(expected) {
		t.Errorf("Expected %d tiers, got %d", len(expected), len(tiers))
	}

	for i, exp := range expected {
		if i >= len(tiers) {
			break
		}
		tier := tiers[i]
		if tier.Placement != exp.placement {
			t.Errorf("Tier %d: expected placement %q, got %q", i, exp.placement, tier.Placement)
		}
		if tier.Low != exp.low || tier.High != exp.high {
			t.Errorf("Tier %d: expected range %d-%d, got %d-%d", i, exp.low, exp.high, tier.Low, tier.High)
		}
	}
}

func TestBinomialCoeff(t *testing.T) {
	tests := []struct {
		n, k, expected int
	}{
		{0, 0, 1},
		{5, 0, 1},
		{5, 5, 1},
		{5, 2, 10},
		{6, 3, 20},
		{10, 5, 252},
	}

	for _, tc := range tests {
		result := binomialCoeff(tc.n, tc.k)
		if result != tc.expected {
			t.Errorf("binomialCoeff(%d, %d) = %d, expected %d", tc.n, tc.k, result, tc.expected)
		}
	}
}

func TestMultiStageSwissTiers(t *testing.T) {
	// Simulate tournament xdzo3nay structure:
	// Stage 1 (order=1): Swiss, 16 participants, 3W/3L
	// Stage 2 (order=2): Swiss, 16 participants, 3W/3L
	// Final (order=0): Single elim, 8 participants
	winsToAdvance := 3
	lossesToEliminate := 3

	stages := []StageTierConfig{
		{
			StageOrder:        2,
			StageType:         StageTypeGroup,
			Format:            FormatSwiss,
			Participants:      16,
			Advancing:         8,
			WinsToAdvance:     &winsToAdvance,
			LossesToEliminate: &lossesToEliminate,
		},
		{
			StageOrder:        1,
			StageType:         StageTypeGroup,
			Format:            FormatSwiss,
			Participants:      16,
			Advancing:         8,
			WinsToAdvance:     &winsToAdvance,
			LossesToEliminate: &lossesToEliminate,
		},
		{
			StageOrder:        0,
			StageType:         StageTypeFinal,
			Format:            FormatSingleElimination,
			Participants:      8,
			Advancing:         0,
			WinsToAdvance:     nil,
			LossesToEliminate: nil,
		},
	}

	tiers := GeneratePlacementTiersConfig(TierGenerationConfig{
		Format:           FormatMultiStage,
		ParticipantCount: 32,
		Stages:           stages,
	})

	t.Log("Multi-stage tiers for tournament with 2 Swiss group stages + Single Elim finals:")
	for _, tier := range tiers {
		t.Logf("  %s (low=%d, high=%d)", tier.Placement, tier.Low, tier.High)
	}

	// Verify we get correct number of tiers
	// Final (8 players): 1st, 2nd, 3rd-4th, 5th-8th = 4 tiers
	// Stage 2 eliminated (8 players from 16-8): should be grouped by record (2-3:3, 1-3:3, 0-3:2)
	// Stage 1 eliminated (8 players from 16-8): should be grouped by record

	// Currently this test just logs output to see what we're getting
	if len(tiers) < 4 {
		t.Errorf("Expected at least 4 tiers for finals, got %d total", len(tiers))
	}
}

func TestCalculateDoubleElimDistribution(t *testing.T) {
	testCases := []struct {
		name         string
		participants int
		advancing    int
		wantTotal    int // Total eliminated should equal participants - advancing
	}{
		{
			name:         "8 player double elim, all finish",
			participants: 8,
			advancing:    0,
			wantTotal:    6, // 8 - 2 (champion + runner-up counted separately)
		},
		{
			name:         "8 player double elim, 4 advance",
			participants: 8,
			advancing:    4,
			wantTotal:    4,
		},
		{
			name:         "12 player double elim (with byes), all finish",
			participants: 12,
			advancing:    0,
			wantTotal:    10, // 12 - 2 (champion + runner-up)
		},
		{
			name:         "4 player double elim, 2 advance",
			participants: 4,
			advancing:    2,
			wantTotal:    2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dist := calculateDoubleElimDistribution(tc.participants, tc.advancing)

			t.Logf("Distribution for %s:", tc.name)
			total := 0
			for _, d := range dist {
				t.Logf("  %d-%d: %d players", d.Wins, d.Losses, d.Count)
				total += d.Count

				// All eliminated players should have 2 losses
				if d.Losses != 2 {
					t.Errorf("Expected 2 losses for eliminated player, got %d", d.Losses)
				}
			}
			t.Logf("  Total eliminated: %d (expected %d)", total, tc.wantTotal)

			// Verify total matches expected
			if total != tc.wantTotal {
				t.Errorf("Total eliminated %d != expected %d", total, tc.wantTotal)
			}
		})
	}
}

func TestGenerateDoubleElimEliminatedTiers(t *testing.T) {
	// 8 players, 4 advance = 4 eliminated
	tiers := generateDoubleElimEliminatedTiers(1, 8, 4)

	t.Log("Eliminated tiers for 8-player double elim (4 advance):")
	for _, tier := range tiers {
		t.Logf("  %s (low=%d, high=%d)", tier.Placement, tier.Low, tier.High)
	}

	// Should have at least 1 tier
	if len(tiers) == 0 {
		t.Error("Expected at least 1 tier")
	}

	// Total coverage should equal 4 eliminated
	totalCovered := 0
	for _, tier := range tiers {
		totalCovered += tier.High - tier.Low + 1
	}
	if totalCovered != 4 {
		t.Errorf("Expected tiers to cover 4 positions, got %d", totalCovered)
	}
}

func TestMultiStageDoubleElimTiers(t *testing.T) {
	// Simulate tournament c5hsbwlp structure:
	// Stage 1 (order=1): Double Elim, 24 participants (2 groups of 12), 8 advance
	// Stage 2 (order=2): Double Elim, 16 participants (4 groups of 4), 8 advance
	// Final (order=0): Single elim, 8 participants

	stages := []StageTierConfig{
		{
			StageOrder:   1,
			StageType:    StageTypeGroup,
			Format:       FormatDoubleElimination,
			Participants: 24,
			Advancing:    8,
		},
		{
			StageOrder:   2,
			StageType:    StageTypeGroup,
			Format:       FormatDoubleElimination,
			Participants: 16,
			Advancing:    8,
		},
		{
			StageOrder:   0,
			StageType:    StageTypeFinal,
			Format:       FormatSingleElimination,
			Participants: 8,
			Advancing:    0,
		},
	}

	tiers := GeneratePlacementTiersConfig(TierGenerationConfig{
		Format:           FormatMultiStage,
		ParticipantCount: 32,
		Stages:           stages,
	})

	t.Log("Multi-stage tiers for tournament with 2 Double Elim group stages + Single Elim finals:")
	for _, tier := range tiers {
		t.Logf("  %s (low=%d, high=%d)", tier.Placement, tier.Low, tier.High)
	}

	// Verify structure:
	// Finals: 1st, 2nd, 3rd-4th, 5th-8th = 4 tiers
	// Stage 2 eliminated (16-8=8): grouped by record
	// Stage 1 eliminated (24-8=16): grouped by record
	if len(tiers) < 4 {
		t.Errorf("Expected at least 4 tiers for finals, got %d total", len(tiers))
	}

	// Verify first tier is 1st place
	if len(tiers) > 0 && tiers[0].Low != 1 {
		t.Errorf("Expected first tier to start at position 1, got %d", tiers[0].Low)
	}

	// Verify total coverage equals 32 participants
	lastHigh := 0
	for _, tier := range tiers {
		if tier.High > lastHigh {
			lastHigh = tier.High
		}
	}
	// Final should cover 8, Stage 2 eliminated should cover 8, Stage 1 eliminated should cover 16
	// Total: 8 + 8 + 16 = 32
	if lastHigh != 32 {
		t.Errorf("Expected tiers to cover up to position 32, got %d", lastHigh)
	}
}

func TestDoubleElimWithByes(t *testing.T) {
	// 5-player double elim (bracket size 8, 3 byes)
	// Verify the distribution accounts for byes
	dist := calculateDoubleElimDistribution(5, 0)

	t.Log("Distribution for 5-player double elim (3 byes):")
	total := 0
	for _, d := range dist {
		t.Logf("  %d-%d: %d players", d.Wins, d.Losses, d.Count)
		total += d.Count
	}
	t.Logf("  Total eliminated: %d (expected 3)", total)

	// 5 players, 2 finalists = 3 eliminated
	if total != 3 {
		t.Errorf("Expected 3 eliminated players, got %d", total)
	}

	// With byes, we should see lower win counts on average
	// because bye recipients advance without gaining a win
}
