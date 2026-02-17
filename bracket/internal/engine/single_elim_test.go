package engine

import (
	"testing"

	"github.com/braccet/bracket/internal/domain"
)

func makeParticipants(n int) []domain.Participant {
	participants := make([]domain.Participant, n)
	for i := range n {
		participants[i] = domain.Participant{
			ID:   uint64(i + 1),
			Name: string(rune('A' + i)),
			Seed: i + 1,
		}
	}
	return participants
}

func TestSingleElimination_MinimumParticipants(t *testing.T) {
	_, err := SingleElimination(1, []domain.Participant{})
	if err == nil {
		t.Error("expected error for 0 participants")
	}

	_, err = SingleElimination(1, makeParticipants(1))
	if err == nil {
		t.Error("expected error for 1 participant")
	}
}

func TestSingleElimination_TwoParticipants(t *testing.T) {
	matches, err := SingleElimination(1, makeParticipants(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	m := matches[0]
	if m.Round != 1 {
		t.Errorf("expected round 1, got %d", m.Round)
	}
	if m.Status != domain.MatchReady {
		t.Errorf("expected status Ready, got %s", m.Status)
	}
	if m.Participant1ID == nil || m.Participant2ID == nil {
		t.Error("both participants should be set")
	}
}

func TestSingleElimination_FourParticipants(t *testing.T) {
	matches, err := SingleElimination(1, makeParticipants(4))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 4 participants: 2 round1 matches + 1 final = 3 matches
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(matches))
	}

	// Count matches per round
	round1 := 0
	round2 := 0
	for _, m := range matches {
		if m.Round == 1 {
			round1++
		} else if m.Round == 2 {
			round2++
		}
	}

	if round1 != 2 {
		t.Errorf("expected 2 round 1 matches, got %d", round1)
	}
	if round2 != 1 {
		t.Errorf("expected 1 round 2 match, got %d", round2)
	}

	// Round 1 matches should be ready
	for _, m := range matches {
		if m.Round == 1 && m.Status != domain.MatchReady {
			t.Errorf("round 1 match should be Ready, got %s", m.Status)
		}
	}

	// Round 2 match should be pending
	for _, m := range matches {
		if m.Round == 2 && m.Status != domain.MatchPending {
			t.Errorf("round 2 match should be Pending, got %s", m.Status)
		}
	}
}

func TestSingleElimination_EightParticipants(t *testing.T) {
	matches, err := SingleElimination(1, makeParticipants(8))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 8 participants: 4 + 2 + 1 = 7 matches
	if len(matches) != 7 {
		t.Fatalf("expected 7 matches, got %d", len(matches))
	}

	roundCounts := make(map[int]int)
	for _, m := range matches {
		roundCounts[m.Round]++
	}

	if roundCounts[1] != 4 {
		t.Errorf("expected 4 round 1 matches, got %d", roundCounts[1])
	}
	if roundCounts[2] != 2 {
		t.Errorf("expected 2 round 2 matches, got %d", roundCounts[2])
	}
	if roundCounts[3] != 1 {
		t.Errorf("expected 1 round 3 match, got %d", roundCounts[3])
	}
}

func TestSingleElimination_WithByes(t *testing.T) {
	// 5 participants -> 8 bracket size -> 3 byes
	matches, err := SingleElimination(1, makeParticipants(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 7 matches (same as 8-player bracket)
	if len(matches) != 7 {
		t.Fatalf("expected 7 matches, got %d", len(matches))
	}

	// Count completed matches (byes)
	completed := 0
	ready := 0
	for _, m := range matches {
		if m.Round == 1 {
			if m.Status == domain.MatchCompleted {
				completed++
				// Completed match should have a winner
				if m.WinnerID == nil {
					t.Error("bye match should have winner set")
				}
			} else if m.Status == domain.MatchReady {
				ready++
			}
		}
	}

	// 3 byes = 3 completed matches in round 1
	// 5 participants in 4 matches: at least one match has both participants
	if completed != 3 {
		t.Errorf("expected 3 bye matches (completed), got %d", completed)
	}
	if ready != 1 {
		t.Errorf("expected 1 ready match in round 1, got %d", ready)
	}
}

func TestSingleElimination_SixteenParticipants(t *testing.T) {
	matches, err := SingleElimination(1, makeParticipants(16))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 16 participants: 8 + 4 + 2 + 1 = 15 matches
	if len(matches) != 15 {
		t.Fatalf("expected 15 matches, got %d", len(matches))
	}

	roundCounts := make(map[int]int)
	for _, m := range matches {
		roundCounts[m.Round]++
	}

	if roundCounts[1] != 8 {
		t.Errorf("expected 8 round 1 matches, got %d", roundCounts[1])
	}
	if roundCounts[2] != 4 {
		t.Errorf("expected 4 round 2 matches, got %d", roundCounts[2])
	}
	if roundCounts[3] != 2 {
		t.Errorf("expected 2 round 3 matches, got %d", roundCounts[3])
	}
	if roundCounts[4] != 1 {
		t.Errorf("expected 1 round 4 match, got %d", roundCounts[4])
	}
}

func TestSingleElimination_ThirtyTwoParticipants(t *testing.T) {
	matches, err := SingleElimination(1, makeParticipants(32))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 32 participants: 16 + 8 + 4 + 2 + 1 = 31 matches
	if len(matches) != 31 {
		t.Fatalf("expected 31 matches, got %d", len(matches))
	}
}

func TestSingleElimination_Seeding(t *testing.T) {
	matches, err := SingleElimination(1, makeParticipants(8))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find round 1 matches and verify seeding
	// Expected: 1v8, 4v5, 2v7, 3v6
	expectedPairings := map[uint64]uint64{
		1: 8,
		4: 5,
		2: 7,
		3: 6,
	}

	for _, m := range matches {
		if m.Round != 1 {
			continue
		}
		if m.Participant1ID == nil || m.Participant2ID == nil {
			continue
		}
		p1 := *m.Participant1ID
		p2 := *m.Participant2ID

		// Check if either direction matches expected
		expected, ok := expectedPairings[p1]
		if !ok {
			expected, ok = expectedPairings[p2]
			if !ok || expected != p1 {
				t.Errorf("unexpected pairing: %d vs %d", p1, p2)
			}
		} else if expected != p2 {
			t.Errorf("seed %d should face seed %d, got %d", p1, expected, p2)
		}
	}
}

func TestSingleElimination_TournamentID(t *testing.T) {
	tournamentID := uint64(42)
	matches, err := SingleElimination(tournamentID, makeParticipants(4))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, m := range matches {
		if m.TournamentID != tournamentID {
			t.Errorf("expected tournament ID %d, got %d", tournamentID, m.TournamentID)
		}
	}
}

func TestSingleElimination_BracketType(t *testing.T) {
	matches, err := SingleElimination(1, makeParticipants(8))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, m := range matches {
		if m.BracketType != domain.BracketWinners {
			t.Errorf("expected bracket type 'winners', got '%s'", m.BracketType)
		}
	}
}

func TestGetBracketState(t *testing.T) {
	matches, _ := SingleElimination(1, makeParticipants(8))

	state := GetBracketState(1, matches)

	if state.TournamentID != 1 {
		t.Errorf("expected tournament ID 1, got %d", state.TournamentID)
	}
	if state.Format != FormatSingleElim {
		t.Errorf("expected format single_elimination, got %s", state.Format)
	}
	if state.TotalRounds != 3 {
		t.Errorf("expected 3 total rounds, got %d", state.TotalRounds)
	}
	if state.IsComplete {
		t.Error("bracket should not be complete")
	}
	if len(state.Matches) != 7 {
		t.Errorf("expected 7 matches, got %d", len(state.Matches))
	}
}
