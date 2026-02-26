package engine

import (
	"testing"

	"github.com/braccet/bracket/internal/domain"
)

// TestGenerateSwissPairings_ScoreGroupsRespected tests that pairings respect score groups.
// This was a bug where players with different records (e.g., 2-1 vs 1-2) were being paired
// when they shouldn't be.
func TestGenerateSwissPairings_ScoreGroupsRespected(t *testing.T) {
	// Simulate 12 active players after 3 rounds:
	// 6 players at 2-1: MOUZ, CPH, fnatic, NAVI, Dignitas, Liquid
	// 6 players at 1-2: Envy, CLG, HEROIC, G2, Renegades, NiP
	standings := []*domain.SwissStanding{
		{ParticipantID: 2134, ParticipantName: "MOUZ", Seed: 14, Wins: 2, Losses: 1, GameWins: 158, GameLosses: 132},
		{ParticipantID: 2123, ParticipantName: "CPH", Seed: 3, Wins: 2, Losses: 1, GameWins: 123, GameLosses: 97},
		{ParticipantID: 2126, ParticipantName: "fnatic", Seed: 6, Wins: 2, Losses: 1, GameWins: 125, GameLosses: 116},
		{ParticipantID: 2124, ParticipantName: "NAVI", Seed: 4, Wins: 2, Losses: 1, GameWins: 102, GameLosses: 96},
		{ParticipantID: 2135, ParticipantName: "Dignitas", Seed: 15, Wins: 2, Losses: 1, GameWins: 157, GameLosses: 162},
		{ParticipantID: 2132, ParticipantName: "Liquid", Seed: 12, Wins: 2, Losses: 1, GameWins: 143, GameLosses: 157},
		{ParticipantID: 2122, ParticipantName: "Envy", Seed: 2, Wins: 1, Losses: 2, GameWins: 142, GameLosses: 130},
		{ParticipantID: 2137, ParticipantName: "CLG", Seed: 16, Wins: 1, Losses: 2, GameWins: 129, GameLosses: 129},
		{ParticipantID: 2129, ParticipantName: "HEROIC", Seed: 9, Wins: 1, Losses: 2, GameWins: 165, GameLosses: 167},
		{ParticipantID: 2131, ParticipantName: "G2", Seed: 11, Wins: 1, Losses: 2, GameWins: 147, GameLosses: 151},
		{ParticipantID: 2128, ParticipantName: "Renegades", Seed: 8, Wins: 1, Losses: 2, GameWins: 95, GameLosses: 105},
		{ParticipantID: 2130, ParticipantName: "NiP", Seed: 10, Wins: 1, Losses: 2, GameWins: 73, GameLosses: 104},
	}

	// Previous pairings (R1-R3) - the specific case where the bug occurred
	playedPairs := BuildPlayedPairsMap([]*domain.SwissPairingHistory{
		// Round 1
		{Participant1ID: 2123, Participant2ID: 2131}, // CPH vs G2
		{Participant1ID: 2124, Participant2ID: 2132}, // NAVI vs Liquid
		{Participant1ID: 2126, Participant2ID: 2134}, // fnatic vs MOUZ
		{Participant1ID: 2127, Participant2ID: 2135}, // Titan vs Dignitas (Titan eliminated)
		{Participant1ID: 2128, Participant2ID: 2137}, // Renegades vs CLG
		// Round 2
		{Participant1ID: 2123, Participant2ID: 2132}, // CPH vs Liquid
		{Participant1ID: 2124, Participant2ID: 2131}, // NAVI vs G2
		{Participant1ID: 2129, Participant2ID: 2134}, // HEROIC vs MOUZ
		{Participant1ID: 2135, Participant2ID: 2137}, // Dignitas vs CLG
		// Round 3
		{Participant1ID: 2122, Participant2ID: 2134}, // Envy vs MOUZ
		{Participant1ID: 2124, Participant2ID: 2128}, // NAVI vs Renegades
		{Participant1ID: 2126, Participant2ID: 2130}, // fnatic vs NiP
		{Participant1ID: 2132, Participant2ID: 2137}, // Liquid vs CLG
	})

	pairings, err := GenerateSwissPairings(standings, playedPairs, 4)
	if err != nil {
		t.Fatalf("GenerateSwissPairings failed: %v", err)
	}

	// Verify all pairings are same-record (no 2-1 vs 1-2)
	for _, p := range pairings {
		if p.IsBye {
			continue
		}

		// Find the standings for both participants
		var p1Standing, p2Standing *domain.SwissStanding
		for _, s := range standings {
			if s.ParticipantID == p.Participant1ID {
				p1Standing = s
			}
			if s.ParticipantID == p.Participant2ID {
				p2Standing = s
			}
		}

		if p1Standing == nil || p2Standing == nil {
			t.Errorf("Could not find standings for pairing %d vs %d", p.Participant1ID, p.Participant2ID)
			continue
		}

		if p1Standing.Wins != p2Standing.Wins {
			t.Errorf("Cross-record pairing: %s (%d-%d) vs %s (%d-%d)",
				p1Standing.ParticipantName, p1Standing.Wins, p1Standing.Losses,
				p2Standing.ParticipantName, p2Standing.Wins, p2Standing.Losses)
		}
	}

	// Should have 6 pairings (12 players / 2)
	if len(pairings) != 6 {
		t.Errorf("Expected 6 pairings, got %d", len(pairings))
	}
}

// TestGenerateSwissPairings_OddScoreGroup tests that a player floats down when their
// score group has an odd number.
func TestGenerateSwissPairings_OddScoreGroup(t *testing.T) {
	// 5 players: 3 at 1-0, 2 at 0-1
	standings := []*domain.SwissStanding{
		{ParticipantID: 1, ParticipantName: "A", Seed: 1, Wins: 1, Losses: 0},
		{ParticipantID: 2, ParticipantName: "B", Seed: 2, Wins: 1, Losses: 0},
		{ParticipantID: 3, ParticipantName: "C", Seed: 3, Wins: 1, Losses: 0},
		{ParticipantID: 4, ParticipantName: "D", Seed: 4, Wins: 0, Losses: 1},
		{ParticipantID: 5, ParticipantName: "E", Seed: 5, Wins: 0, Losses: 1},
	}

	// Round 1: 1v4 (1 wins), 2v5 (2 wins), 3 gets BYE
	playedPairs := BuildPlayedPairsMap([]*domain.SwissPairingHistory{
		{Participant1ID: 1, Participant2ID: 4},
		{Participant1ID: 2, Participant2ID: 5},
	})

	// Mark player 3 as having had a BYE
	standings[2].HasBye = true

	pairings, err := GenerateSwissPairings(standings, playedPairs, 2)
	if err != nil {
		t.Fatalf("GenerateSwissPairings failed: %v", err)
	}

	// Should have 2 match pairings + 1 BYE (5 players, odd)
	byeCount := 0
	matchCount := 0
	for _, p := range pairings {
		if p.IsBye {
			byeCount++
		} else {
			matchCount++
		}
	}

	if byeCount != 1 {
		t.Errorf("Expected 1 BYE, got %d", byeCount)
	}
	if matchCount != 2 {
		t.Errorf("Expected 2 matches, got %d", matchCount)
	}

	// One cross-record pairing is expected since we have 3 players at 1-0 (odd number)
	// Two 1-0 players pair together, one 1-0 floats down to pair with a 0-1
	crossRecordCount := 0
	for _, p := range pairings {
		if p.IsBye {
			continue
		}
		var p1Wins, p2Wins int
		for _, s := range standings {
			if s.ParticipantID == p.Participant1ID {
				p1Wins = s.Wins
			}
			if s.ParticipantID == p.Participant2ID {
				p2Wins = s.Wins
			}
		}
		if p1Wins != p2Wins {
			crossRecordCount++
		}
	}

	// Exactly 1 cross-record pairing is expected (the floater)
	if crossRecordCount != 1 {
		t.Errorf("Expected exactly 1 cross-record pairing (floater), got %d", crossRecordCount)
	}
}

// TestGenerateSwissPairings_TopVsBottom tests that within a score group,
// the highest-ranked player is paired with the lowest-ranked player.
// Example: In 1-0 pool with [Reason, Tempo Storm, Cloud9, LDLC] (ranked),
// pairings should be: Reason vs LDLC, Tempo Storm vs Cloud9
func TestGenerateSwissPairings_TopVsBottom(t *testing.T) {
	// 8 players after round 1: 4 at 1-0, 4 at 0-1
	// Within 1-0: Reason(1), Tempo Storm(2), Cloud9(3), LDLC(4)
	standings := []*domain.SwissStanding{
		{ParticipantID: 1, ParticipantName: "Reason", Seed: 1, Wins: 1, Losses: 0},
		{ParticipantID: 2, ParticipantName: "Tempo Storm", Seed: 2, Wins: 1, Losses: 0},
		{ParticipantID: 3, ParticipantName: "Cloud9", Seed: 3, Wins: 1, Losses: 0},
		{ParticipantID: 4, ParticipantName: "LDLC", Seed: 4, Wins: 1, Losses: 0},
		{ParticipantID: 5, ParticipantName: "NiP", Seed: 5, Wins: 0, Losses: 1},
		{ParticipantID: 6, ParticipantName: "Fnatic", Seed: 6, Wins: 0, Losses: 1},
		{ParticipantID: 7, ParticipantName: "G2", Seed: 7, Wins: 0, Losses: 1},
		{ParticipantID: 8, ParticipantName: "Astralis", Seed: 8, Wins: 0, Losses: 1},
	}

	// Round 1: 1v5, 2v6, 3v7, 4v8
	playedPairs := BuildPlayedPairsMap([]*domain.SwissPairingHistory{
		{Participant1ID: 1, Participant2ID: 5},
		{Participant1ID: 2, Participant2ID: 6},
		{Participant1ID: 3, Participant2ID: 7},
		{Participant1ID: 4, Participant2ID: 8},
	})

	pairings, err := GenerateSwissPairings(standings, playedPairs, 2)
	if err != nil {
		t.Fatalf("GenerateSwissPairings failed: %v", err)
	}

	// Should have 4 pairings
	if len(pairings) != 4 {
		t.Errorf("Expected 4 pairings, got %d", len(pairings))
	}

	// Verify top-vs-bottom within each score group
	// 1-0 pool: Reason(1) vs LDLC(4), Tempo Storm(2) vs Cloud9(3)
	expectedPairings := map[uint64]uint64{
		1: 4, // Reason vs LDLC
		2: 3, // Tempo Storm vs Cloud9
		5: 8, // NiP vs Astralis
		6: 7, // Fnatic vs G2
	}

	for _, p := range pairings {
		expected, ok := expectedPairings[p.Participant1ID]
		if !ok {
			expected, ok = expectedPairings[p.Participant2ID]
			if !ok {
				t.Errorf("Unexpected pairing: %d vs %d", p.Participant1ID, p.Participant2ID)
				continue
			}
			// Swap for comparison
			if expected != p.Participant1ID {
				t.Errorf("Expected top-vs-bottom pairing, got %d vs %d (expected %d vs %d)",
					p.Participant1ID, p.Participant2ID, p.Participant2ID, expected)
			}
		} else {
			if expected != p.Participant2ID {
				t.Errorf("Expected top-vs-bottom pairing, got %d vs %d (expected %d vs %d)",
					p.Participant1ID, p.Participant2ID, p.Participant1ID, expected)
			}
		}
	}
}

// TestSortStandings_ThresholdModeRanking tests that standings are sorted correctly
// in threshold mode: 3-0 > 3-1 > 3-2 for advanced, 0-3 < 1-3 < 2-3 for eliminated.
func TestSortStandings_ThresholdModeRanking(t *testing.T) {
	standings := []*domain.SwissStanding{
		// Advanced participants (different loss counts)
		{ParticipantID: 1, ParticipantName: "A", Seed: 1, Wins: 3, Losses: 2, Status: domain.SwissStatusAdvanced}, // 3-2
		{ParticipantID: 2, ParticipantName: "B", Seed: 2, Wins: 3, Losses: 0, Status: domain.SwissStatusAdvanced}, // 3-0
		{ParticipantID: 3, ParticipantName: "C", Seed: 3, Wins: 3, Losses: 1, Status: domain.SwissStatusAdvanced}, // 3-1
		// Active participants
		{ParticipantID: 4, ParticipantName: "D", Seed: 4, Wins: 2, Losses: 1, Status: domain.SwissStatusActive}, // 2-1
		{ParticipantID: 5, ParticipantName: "E", Seed: 5, Wins: 1, Losses: 2, Status: domain.SwissStatusActive}, // 1-2
		// Eliminated participants (different win counts)
		{ParticipantID: 6, ParticipantName: "F", Seed: 6, Wins: 0, Losses: 3, Status: domain.SwissStatusEliminated}, // 0-3
		{ParticipantID: 7, ParticipantName: "G", Seed: 7, Wins: 2, Losses: 3, Status: domain.SwissStatusEliminated}, // 2-3
		{ParticipantID: 8, ParticipantName: "H", Seed: 8, Wins: 1, Losses: 3, Status: domain.SwissStatusEliminated}, // 1-3
	}

	sortStandings(standings)

	// Expected order:
	// 1. Advanced: 3-0 (B), 3-1 (C), 3-2 (A)
	// 2. Active: 2-1 (D), 1-2 (E)
	// 3. Eliminated: 2-3 (G), 1-3 (H), 0-3 (F)
	expectedOrder := []uint64{2, 3, 1, 4, 5, 7, 8, 6}

	for i, expected := range expectedOrder {
		if standings[i].ParticipantID != expected {
			t.Errorf("Position %d: expected participant %d, got %d (%s, %d-%d, status=%s)",
				i, expected, standings[i].ParticipantID, standings[i].ParticipantName,
				standings[i].Wins, standings[i].Losses, standings[i].Status)
		}
	}
}

// TestSortStandings_AdvancedTiebreakers tests that tiebreakers work within advanced status
// when loss counts are equal.
func TestSortStandings_AdvancedTiebreakers(t *testing.T) {
	standings := []*domain.SwissStanding{
		// Both 3-1, but different game differentials
		{ParticipantID: 1, ParticipantName: "A", Seed: 2, Wins: 3, Losses: 1, GameWins: 10, GameLosses: 8, Status: domain.SwissStatusAdvanced},  // +2
		{ParticipantID: 2, ParticipantName: "B", Seed: 1, Wins: 3, Losses: 1, GameWins: 12, GameLosses: 6, Status: domain.SwissStatusAdvanced},  // +6
		{ParticipantID: 3, ParticipantName: "C", Seed: 3, Wins: 3, Losses: 1, GameWins: 10, GameLosses: 10, Status: domain.SwissStatusAdvanced}, // 0
	}

	sortStandings(standings)

	// Expected order: B (+6), A (+2), C (0)
	expectedOrder := []uint64{2, 1, 3}

	for i, expected := range expectedOrder {
		if standings[i].ParticipantID != expected {
			t.Errorf("Position %d: expected participant %d, got %d", i, expected, standings[i].ParticipantID)
		}
	}
}

// TestSortStandings_EliminatedTiebreakers tests that tiebreakers work within eliminated status
// when win counts are equal.
func TestSortStandings_EliminatedTiebreakers(t *testing.T) {
	standings := []*domain.SwissStanding{
		// Both 1-3, but different game differentials
		{ParticipantID: 1, ParticipantName: "A", Seed: 2, Wins: 1, Losses: 3, GameWins: 6, GameLosses: 10, Status: domain.SwissStatusEliminated},  // -4
		{ParticipantID: 2, ParticipantName: "B", Seed: 1, Wins: 1, Losses: 3, GameWins: 8, GameLosses: 10, Status: domain.SwissStatusEliminated},  // -2
		{ParticipantID: 3, ParticipantName: "C", Seed: 3, Wins: 1, Losses: 3, GameWins: 4, GameLosses: 12, Status: domain.SwissStatusEliminated}, // -8
	}

	sortStandings(standings)

	// Expected order: B (-2), A (-4), C (-8)
	expectedOrder := []uint64{2, 1, 3}

	for i, expected := range expectedOrder {
		if standings[i].ParticipantID != expected {
			t.Errorf("Position %d: expected participant %d, got %d", i, expected, standings[i].ParticipantID)
		}
	}
}

// TestGenerateSwissPairings_AvoidRematches tests that rematches are avoided within score groups.
func TestGenerateSwissPairings_AvoidRematches(t *testing.T) {
	// 4 players at same record (1-0)
	standings := []*domain.SwissStanding{
		{ParticipantID: 1, ParticipantName: "A", Seed: 1, Wins: 1, Losses: 0},
		{ParticipantID: 2, ParticipantName: "B", Seed: 2, Wins: 1, Losses: 0},
		{ParticipantID: 3, ParticipantName: "C", Seed: 3, Wins: 1, Losses: 0},
		{ParticipantID: 4, ParticipantName: "D", Seed: 4, Wins: 1, Losses: 0},
	}

	// 1 played 2, 3 played 4 in round 1
	playedPairs := BuildPlayedPairsMap([]*domain.SwissPairingHistory{
		{Participant1ID: 1, Participant2ID: 2},
		{Participant1ID: 3, Participant2ID: 4},
	})

	pairings, err := GenerateSwissPairings(standings, playedPairs, 2)
	if err != nil {
		t.Fatalf("GenerateSwissPairings failed: %v", err)
	}

	// Should have 2 pairings
	if len(pairings) != 2 {
		t.Errorf("Expected 2 pairings, got %d", len(pairings))
	}

	// Check no rematches
	for _, p := range pairings {
		if havePlayed(playedPairs, p.Participant1ID, p.Participant2ID) {
			t.Errorf("Rematch detected: %d vs %d", p.Participant1ID, p.Participant2ID)
		}
	}
}
