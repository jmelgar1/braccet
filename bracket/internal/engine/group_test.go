package engine

import (
	"testing"

	"github.com/braccet/bracket/internal/domain"
)

func TestApplyCrossGroupSeeding_TwoGroups(t *testing.T) {
	// Test case: 2 groups with 3 advancing each
	// Expected: Cross-seeding ensures participants from same group are distributed
	// across bracket halves (seeds 1,4,5 top half; seeds 2,3,6 bottom half)

	standings := []*domain.StageStanding{
		// Group A participants (GroupID=1)
		{ParticipantID: 1, GroupID: 1, GroupRank: 1}, // A1 - 1st from Group A
		{ParticipantID: 2, GroupID: 1, GroupRank: 2}, // A2 - 2nd from Group A
		{ParticipantID: 3, GroupID: 1, GroupRank: 3}, // A3 - 3rd from Group A
		// Group B participants (GroupID=2)
		{ParticipantID: 4, GroupID: 2, GroupRank: 1}, // B1 - 1st from Group B
		{ParticipantID: 5, GroupID: 2, GroupRank: 2}, // B2 - 2nd from Group B
		{ParticipantID: 6, GroupID: 2, GroupRank: 3}, // B3 - 3rd from Group B
	}

	applyCrossGroupSeeding(standings)

	// Verify stage ranks were assigned
	for i, s := range standings {
		if s.StageRank == nil {
			t.Errorf("StageRank not assigned for standings[%d]", i)
			continue
		}
	}

	// Build a map of stage rank to participant/group for easier verification
	rankToParticipant := make(map[int]uint64)
	rankToGroup := make(map[int]uint64)
	for _, s := range standings {
		rankToParticipant[*s.StageRank] = s.ParticipantID
		rankToGroup[*s.StageRank] = s.GroupID
	}

	// Expected ordering with serpentine by super-tiers:
	// Tier 0 (1st places): A1, B1 -> ranks 1, 2 (normal order)
	// Tier 1 (2nd places): A2, B2 -> ranks 3, 4 (normal order, still super-tier 0)
	// Tier 2 (3rd places): B3, A3 -> ranks 5, 6 (reversed, super-tier 1)

	// Top half (seeds 1, 4, 5): should have participants from both groups
	topHalfSeeds := []int{1, 4, 5}
	topHalfGroups := make(map[uint64]bool)
	for _, seed := range topHalfSeeds {
		if group, ok := rankToGroup[seed]; ok {
			topHalfGroups[group] = true
		}
	}
	if len(topHalfGroups) < 2 {
		t.Errorf("Top half should have participants from both groups, got groups: %v", topHalfGroups)
	}

	// Bottom half (seeds 2, 3, 6): should have participants from both groups
	bottomHalfSeeds := []int{2, 3, 6}
	bottomHalfGroups := make(map[uint64]bool)
	for _, seed := range bottomHalfSeeds {
		if group, ok := rankToGroup[seed]; ok {
			bottomHalfGroups[group] = true
		}
	}
	if len(bottomHalfGroups) < 2 {
		t.Errorf("Bottom half should have participants from both groups, got groups: %v", bottomHalfGroups)
	}

	// Verify specific expected order:
	// A1 (rank 1), B1 (rank 2), A2 (rank 3), B2 (rank 4), B3 (rank 5), A3 (rank 6)
	expectedOrder := []struct {
		rank        int
		participant uint64
		group       uint64
	}{
		{1, 1, 1}, // A1
		{2, 4, 2}, // B1
		{3, 2, 1}, // A2
		{4, 5, 2}, // B2
		{5, 6, 2}, // B3 (reversed for super-tier 1)
		{6, 3, 1}, // A3 (reversed for super-tier 1)
	}

	for _, exp := range expectedOrder {
		if got := rankToParticipant[exp.rank]; got != exp.participant {
			t.Errorf("Rank %d: expected participant %d, got %d", exp.rank, exp.participant, got)
		}
		if got := rankToGroup[exp.rank]; got != exp.group {
			t.Errorf("Rank %d: expected group %d, got %d", exp.rank, exp.group, got)
		}
	}
}

func TestApplyCrossGroupSeeding_ThreeGroups(t *testing.T) {
	// Test case: 3 groups with 2 advancing each
	standings := []*domain.StageStanding{
		{ParticipantID: 1, GroupID: 1, GroupRank: 1}, // A1
		{ParticipantID: 2, GroupID: 1, GroupRank: 2}, // A2
		{ParticipantID: 3, GroupID: 2, GroupRank: 1}, // B1
		{ParticipantID: 4, GroupID: 2, GroupRank: 2}, // B2
		{ParticipantID: 5, GroupID: 3, GroupRank: 1}, // C1
		{ParticipantID: 6, GroupID: 3, GroupRank: 2}, // C2
	}

	applyCrossGroupSeeding(standings)

	// Verify all stage ranks assigned
	for i, s := range standings {
		if s.StageRank == nil {
			t.Errorf("StageRank not assigned for standings[%d]", i)
		}
	}

	// Build rank to group map
	rankToGroup := make(map[int]uint64)
	for _, s := range standings {
		rankToGroup[*s.StageRank] = s.GroupID
	}

	// Expected for 3 groups:
	// Tier 0 (1st places): A1, B1, C1 -> ranks 1, 2, 3 (normal)
	// Tier 1 (2nd places): A2, B2, C2 -> ranks 4, 5, 6 (normal, still super-tier 0)
	// If we had tier 2-3, they would be reversed (super-tier 1)

	// Verify tier 0 order: groups 1, 2, 3
	if rankToGroup[1] != 1 || rankToGroup[2] != 2 || rankToGroup[3] != 3 {
		t.Errorf("Tier 0 order incorrect: got groups %d, %d, %d; expected 1, 2, 3",
			rankToGroup[1], rankToGroup[2], rankToGroup[3])
	}

	// Verify tier 1 order: groups 1, 2, 3 (same as tier 0 since both in super-tier 0)
	if rankToGroup[4] != 1 || rankToGroup[5] != 2 || rankToGroup[6] != 3 {
		t.Errorf("Tier 1 order incorrect: got groups %d, %d, %d; expected 1, 2, 3",
			rankToGroup[4], rankToGroup[5], rankToGroup[6])
	}
}

func TestApplyCrossGroupSeeding_SingleGroup(t *testing.T) {
	// Test case: Single group (regression test)
	standings := []*domain.StageStanding{
		{ParticipantID: 1, GroupID: 1, GroupRank: 1},
		{ParticipantID: 2, GroupID: 1, GroupRank: 2},
		{ParticipantID: 3, GroupID: 1, GroupRank: 3},
	}

	applyCrossGroupSeeding(standings)

	// Verify ranks assigned sequentially
	for i, s := range standings {
		if s.StageRank == nil {
			t.Errorf("StageRank not assigned for standings[%d]", i)
			continue
		}
		expectedRank := i + 1
		if *s.StageRank != expectedRank {
			t.Errorf("standings[%d]: expected rank %d, got %d", i, expectedRank, *s.StageRank)
		}
	}
}

func TestApplyCrossGroupSeeding_EmptyStandings(t *testing.T) {
	standings := []*domain.StageStanding{}
	applyCrossGroupSeeding(standings)
	// Should not panic with empty slice
}

func TestApplyCrossGroupSeeding_FourAdvancingEach(t *testing.T) {
	// Test case: 2 groups with 4 advancing each (tests more super-tiers)
	standings := []*domain.StageStanding{
		// Group A
		{ParticipantID: 1, GroupID: 1, GroupRank: 1},
		{ParticipantID: 2, GroupID: 1, GroupRank: 2},
		{ParticipantID: 3, GroupID: 1, GroupRank: 3},
		{ParticipantID: 4, GroupID: 1, GroupRank: 4},
		// Group B
		{ParticipantID: 5, GroupID: 2, GroupRank: 1},
		{ParticipantID: 6, GroupID: 2, GroupRank: 2},
		{ParticipantID: 7, GroupID: 2, GroupRank: 3},
		{ParticipantID: 8, GroupID: 2, GroupRank: 4},
	}

	applyCrossGroupSeeding(standings)

	rankToGroup := make(map[int]uint64)
	for _, s := range standings {
		rankToGroup[*s.StageRank] = s.GroupID
	}

	// Expected order:
	// Tier 0 (rank 1): A, B -> ranks 1, 2 (super-tier 0, normal)
	// Tier 1 (rank 2): A, B -> ranks 3, 4 (super-tier 0, normal)
	// Tier 2 (rank 3): B, A -> ranks 5, 6 (super-tier 1, reversed)
	// Tier 3 (rank 4): B, A -> ranks 7, 8 (super-tier 1, reversed)

	expectedGroups := []uint64{1, 2, 1, 2, 2, 1, 2, 1}
	for rank := 1; rank <= 8; rank++ {
		expected := expectedGroups[rank-1]
		if rankToGroup[rank] != expected {
			t.Errorf("Rank %d: expected group %d, got %d", rank, expected, rankToGroup[rank])
		}
	}
}

func TestAggregateStageStandings_CrossGroupSeeding(t *testing.T) {
	// Integration test: verify AggregateStageStandings applies cross-group seeding
	groupStandings := map[uint64][]*domain.GroupStanding{
		1: {
			{ParticipantID: 1, GroupRank: intPtr(1), MatchWins: 2},
			{ParticipantID: 2, GroupRank: intPtr(2), MatchWins: 1},
			{ParticipantID: 3, GroupRank: intPtr(3), MatchWins: 0},
		},
		2: {
			{ParticipantID: 4, GroupRank: intPtr(1), MatchWins: 2},
			{ParticipantID: 5, GroupRank: intPtr(2), MatchWins: 1},
			{ParticipantID: 6, GroupRank: intPtr(3), MatchWins: 0},
		},
	}

	result := AggregateStageStandings(groupStandings, 3, nil)

	// Verify cross-seeding was applied
	rankToGroup := make(map[int]uint64)
	for _, s := range result {
		rankToGroup[*s.StageRank] = s.GroupID
	}

	// Top half (seeds 1, 4, 5) should have both groups
	topHalfGroups := make(map[uint64]bool)
	for _, seed := range []int{1, 4, 5} {
		if group, ok := rankToGroup[seed]; ok {
			topHalfGroups[group] = true
		}
	}

	if len(topHalfGroups) < 2 {
		t.Errorf("Top half should have participants from both groups after AggregateStageStandings")
	}
}

func intPtr(i int) *int {
	return &i
}
