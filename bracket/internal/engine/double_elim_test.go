package engine

import (
	"testing"

	"github.com/braccet/bracket/internal/domain"
)

// createTestParticipants creates n test participants with sequential seeds
func createTestParticipants(n int) []domain.Participant {
	participants := make([]domain.Participant, n)
	for i := 0; i < n; i++ {
		participants[i] = domain.Participant{
			ID:   uint64(i + 1),
			Name: "Player " + string(rune('A'+i)),
			Seed: i + 1,
		}
	}
	return participants
}

// countMatchesByBracketAndRound counts matches for a specific bracket type and round
func countMatchesByBracketAndRound(matches []*domain.Match, bracketType domain.BracketType, round int) int {
	count := 0
	for _, m := range matches {
		if m.BracketType == bracketType && m.Round == round {
			count++
		}
	}
	return count
}

// getMatchesByBracketAndRound returns matches for a specific bracket type and round
func getMatchesByBracketAndRoundTest(matches []*domain.Match, bracketType domain.BracketType, round int) []*domain.Match {
	var result []*domain.Match
	for _, m := range matches {
		if m.BracketType == bracketType && m.Round == round {
			result = append(result, m)
		}
	}
	return result
}

func TestDoubleElimination_5Participants_SkipsUnreachableLR1(t *testing.T) {
	participants := createTestParticipants(5)

	matches, err := DoubleElimination(1, participants)
	if err != nil {
		t.Fatalf("DoubleElimination failed: %v", err)
	}

	// For 5 participants (8-bracket):
	// - W-R1: 4 matches (1 real: seed 4v5, 3 BYEs)
	// - L-R1: Should have 1 match (not 2) because L-R1-M2 would receive from two BYE matches

	wr1Count := countMatchesByBracketAndRound(matches, domain.BracketWinners, 1)
	if wr1Count != 4 {
		t.Errorf("Expected 4 W-R1 matches, got %d", wr1Count)
	}

	lr1Count := countMatchesByBracketAndRound(matches, domain.BracketLosers, 1)
	if lr1Count != 1 {
		t.Errorf("Expected 1 L-R1 match (unreachable one should be skipped), got %d", lr1Count)
	}

	// The single L-R1 match should be at position 1
	lr1Matches := getMatchesByBracketAndRoundTest(matches, domain.BracketLosers, 1)
	if len(lr1Matches) > 0 && lr1Matches[0].Position != 1 {
		t.Errorf("Expected L-R1 match at position 1, got position %d", lr1Matches[0].Position)
	}
}

func TestDoubleElimination_6Participants_BothLR1MatchesExist(t *testing.T) {
	participants := createTestParticipants(6)

	matches, err := DoubleElimination(1, participants)
	if err != nil {
		t.Fatalf("DoubleElimination failed: %v", err)
	}

	// For 6 participants (8-bracket):
	// - W-R1: 4 matches (2 real: seed 4v5 and 3v6, 2 BYEs)
	// - L-R1: Should have 2 matches (each has 1 real + 1 BYE source)

	wr1Count := countMatchesByBracketAndRound(matches, domain.BracketWinners, 1)
	if wr1Count != 4 {
		t.Errorf("Expected 4 W-R1 matches, got %d", wr1Count)
	}

	lr1Count := countMatchesByBracketAndRound(matches, domain.BracketLosers, 1)
	if lr1Count != 2 {
		t.Errorf("Expected 2 L-R1 matches, got %d", lr1Count)
	}
}

func TestDoubleElimination_7Participants_BothLR1MatchesExist(t *testing.T) {
	participants := createTestParticipants(7)

	matches, err := DoubleElimination(1, participants)
	if err != nil {
		t.Fatalf("DoubleElimination failed: %v", err)
	}

	// For 7 participants (8-bracket):
	// - W-R1: 4 matches (3 real: seed 4v5, 3v6, 2v7, 1 BYE)
	// - L-R1: Should have 2 matches (both have at least 1 real source)

	lr1Count := countMatchesByBracketAndRound(matches, domain.BracketLosers, 1)
	if lr1Count != 2 {
		t.Errorf("Expected 2 L-R1 matches, got %d", lr1Count)
	}
}

func TestDoubleElimination_8Participants_AllLR1MatchesExist(t *testing.T) {
	participants := createTestParticipants(8)

	matches, err := DoubleElimination(1, participants)
	if err != nil {
		t.Fatalf("DoubleElimination failed: %v", err)
	}

	// For 8 participants (8-bracket):
	// - W-R1: 4 matches (all real, no BYEs)
	// - L-R1: Should have 2 matches

	lr1Count := countMatchesByBracketAndRound(matches, domain.BracketLosers, 1)
	if lr1Count != 2 {
		t.Errorf("Expected 2 L-R1 matches, got %d", lr1Count)
	}
}

func TestDoubleElimination_9Participants_ReducedLR1(t *testing.T) {
	participants := createTestParticipants(9)

	matches, err := DoubleElimination(1, participants)
	if err != nil {
		t.Fatalf("DoubleElimination failed: %v", err)
	}

	// For 9 participants (16-bracket):
	// - W-R1: 8 matches (1 real: seed 8v9, 7 BYEs)
	// - L-R1: Should have 1 match (not 4) because 3 L-R1 positions have double-BYE sources

	wr1Count := countMatchesByBracketAndRound(matches, domain.BracketWinners, 1)
	if wr1Count != 8 {
		t.Errorf("Expected 8 W-R1 matches, got %d", wr1Count)
	}

	lr1Count := countMatchesByBracketAndRound(matches, domain.BracketLosers, 1)
	if lr1Count != 1 {
		t.Errorf("Expected 1 L-R1 match (3 unreachable should be skipped), got %d", lr1Count)
	}
}

func TestDoubleElimination_5Participants_LR2HasByeSlot(t *testing.T) {
	participants := createTestParticipants(5)

	matches, err := DoubleElimination(1, participants)
	if err != nil {
		t.Fatalf("DoubleElimination failed: %v", err)
	}

	// For 5 participants:
	// - L-R1 has 1 match (position 1)
	// - L-R2 should have 2 matches
	// - L-R2-M2 should have slot 2 as BYE (source L-R1-M2 doesn't exist)

	lr2Matches := getMatchesByBracketAndRoundTest(matches, domain.BracketLosers, 2)
	if len(lr2Matches) != 2 {
		t.Fatalf("Expected 2 L-R2 matches, got %d", len(lr2Matches))
	}

	// Find L-R2-M2 (position 2)
	var lr2m2 *domain.Match
	for _, m := range lr2Matches {
		if m.Position == 2 {
			lr2m2 = m
			break
		}
	}

	if lr2m2 == nil {
		t.Fatal("L-R2-M2 not found")
	}

	// L-R2-M2 should have slot 2 marked as BYE
	if lr2m2.Participant2Name == nil || *lr2m2.Participant2Name != "BYE" {
		t.Errorf("Expected L-R2-M2 slot 2 to be BYE, got %v", lr2m2.Participant2Name)
	}
}

func TestDoubleElimination_5Participants_LR1HasSingleBye(t *testing.T) {
	participants := createTestParticipants(5)

	matches, err := DoubleElimination(1, participants)
	if err != nil {
		t.Fatalf("DoubleElimination failed: %v", err)
	}

	// For 5 participants:
	// - L-R1-M1 receives from W-R1-M1 (BYE) and W-R1-M2 (real: 4v5)
	// - L-R1-M1 should have slot 2 as BYE

	lr1Matches := getMatchesByBracketAndRoundTest(matches, domain.BracketLosers, 1)
	if len(lr1Matches) != 1 {
		t.Fatalf("Expected 1 L-R1 match, got %d", len(lr1Matches))
	}

	lr1m1 := lr1Matches[0]
	if lr1m1.Participant2Name == nil || *lr1m1.Participant2Name != "BYE" {
		t.Errorf("Expected L-R1-M1 slot 2 to be BYE, got %v", lr1m1.Participant2Name)
	}
}
