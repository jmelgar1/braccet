package engine

import (
	"testing"
)

func TestCalculateBracketSize(t *testing.T) {
	tests := []struct {
		participants int
		want         int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{6, 8},
		{7, 8},
		{8, 8},
		{9, 16},
		{16, 16},
		{17, 32},
		{32, 32},
	}

	for _, tt := range tests {
		got := CalculateBracketSize(tt.participants)
		if got != tt.want {
			t.Errorf("CalculateBracketSize(%d) = %d, want %d", tt.participants, got, tt.want)
		}
	}
}

func TestTotalRounds(t *testing.T) {
	tests := []struct {
		bracketSize int
		want        int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{4, 2},
		{8, 3},
		{16, 4},
		{32, 5},
	}

	for _, tt := range tests {
		got := TotalRounds(tt.bracketSize)
		if got != tt.want {
			t.Errorf("TotalRounds(%d) = %d, want %d", tt.bracketSize, got, tt.want)
		}
	}
}

func TestMatchesInRound(t *testing.T) {
	tests := []struct {
		bracketSize int
		round       int
		want        int
	}{
		{8, 1, 4},
		{8, 2, 2},
		{8, 3, 1},
		{16, 1, 8},
		{16, 2, 4},
		{16, 3, 2},
		{16, 4, 1},
		{4, 1, 2},
		{4, 2, 1},
		{8, 0, 0}, // invalid round
		{0, 1, 0}, // invalid bracket size
	}

	for _, tt := range tests {
		got := MatchesInRound(tt.bracketSize, tt.round)
		if got != tt.want {
			t.Errorf("MatchesInRound(%d, %d) = %d, want %d", tt.bracketSize, tt.round, got, tt.want)
		}
	}
}

func TestGenerateSeedPairings(t *testing.T) {
	tests := []struct {
		bracketSize int
		want        [][2]int
	}{
		{
			bracketSize: 2,
			want:        [][2]int{{1, 2}},
		},
		{
			bracketSize: 4,
			want:        [][2]int{{1, 4}, {2, 3}},
		},
		{
			bracketSize: 8,
			want:        [][2]int{{1, 8}, {4, 5}, {2, 7}, {3, 6}},
		},
		{
			bracketSize: 16,
			want: [][2]int{
				{1, 16}, {8, 9}, {4, 13}, {5, 12},
				{2, 15}, {7, 10}, {3, 14}, {6, 11},
			},
		},
	}

	for _, tt := range tests {
		got := GenerateSeedPairings(tt.bracketSize)
		if len(got) != len(tt.want) {
			t.Errorf("GenerateSeedPairings(%d): got %d pairings, want %d", tt.bracketSize, len(got), len(tt.want))
			continue
		}
		for i, pair := range got {
			if pair != tt.want[i] {
				t.Errorf("GenerateSeedPairings(%d)[%d] = %v, want %v", tt.bracketSize, i, pair, tt.want[i])
			}
		}
	}
}

func TestSeedPairings_TopSeedsMeetLater(t *testing.T) {
	// Verify that top seeds can only meet in later rounds
	pairings := GenerateSeedPairings(8)

	// In an 8-player bracket:
	// Round 1 winners: 1, 4, 2, 3 (assuming top seeds win)
	// Round 2: 1 vs 4, 2 vs 3
	// Finals: 1 vs 2

	// Check seed 1 and 2 are in different halves (can only meet in finals)
	seed1Pos := -1
	seed2Pos := -1
	for i, pair := range pairings {
		if pair[0] == 1 || pair[1] == 1 {
			seed1Pos = i
		}
		if pair[0] == 2 || pair[1] == 2 {
			seed2Pos = i
		}
	}

	// Seeds 1 and 2 should be in different halves (positions 0-1 vs 2-3)
	seed1Half := seed1Pos / 2
	seed2Half := seed2Pos / 2
	if seed1Half == seed2Half {
		t.Errorf("Seeds 1 and 2 are in the same half of bracket (positions %d and %d)", seed1Pos, seed2Pos)
	}
}

func TestSeedPairings_SumEqualsSize(t *testing.T) {
	// Each pairing should sum to bracketSize + 1
	sizes := []int{4, 8, 16, 32}

	for _, size := range sizes {
		pairings := GenerateSeedPairings(size)
		for _, pair := range pairings {
			sum := pair[0] + pair[1]
			if sum != size+1 {
				t.Errorf("Pairing %v in bracket size %d sums to %d, want %d", pair, size, sum, size+1)
			}
		}
	}
}
