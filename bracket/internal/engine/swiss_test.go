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
