//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/braccet/community/internal/domain"
)

func TestMemberPowerRankingRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	rankingRepo := NewMemberPowerRankingRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	memberID := createTestMember(t, db, communityID, 1, "Player One")

	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Test System",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                true,
		IsActive:                 true,
	}
	if err := prRepo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create PR system: %v", err)
	}

	now := time.Now()

	tests := []struct {
		name    string
		ranking *domain.MemberPowerRanking
		wantErr bool
	}{
		{
			name: "create valid ranking",
			ranking: &domain.MemberPowerRanking{
				MemberID:             memberID,
				PRSystemID:           system.ID,
				TotalPoints:          1500.50,
				Rank:                 ptrInt(1),
				AchievementsScore:    750.25,
				FormScore:            450.15,
				LANScore:             300.10,
				FormWins:             15,
				FormSetWins:          35,
				FormSetLosses:        10,
				FormAvgOpponentRating: ptrInt(1400),
				FormBestWinRating:    ptrInt(1800),
				LANPlacementsCount:   5,
				LastCalculatedAt:     &now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rankingRepo.Create(ctx, tt.ranking)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.ranking.ID == 0 {
					t.Error("Create() should set ID")
				}
				if tt.ranking.CreatedAt.IsZero() {
					t.Error("Create() should set CreatedAt")
				}
				if tt.ranking.UpdatedAt.IsZero() {
					t.Error("Create() should set UpdatedAt")
				}
			}
		})
	}
}

func TestMemberPowerRankingRepository_GetByMemberAndSystem(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	rankingRepo := NewMemberPowerRankingRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	memberID := createTestMember(t, db, communityID, 1, "Player One")

	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Test System",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                true,
		IsActive:                 true,
	}
	if err := prRepo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create PR system: %v", err)
	}

	now := time.Now()
	ranking := &domain.MemberPowerRanking{
		MemberID:           memberID,
		PRSystemID:         system.ID,
		TotalPoints:        1500.50,
		Rank:               ptrInt(1),
		AchievementsScore:  750.25,
		FormScore:          450.15,
		LANScore:           300.10,
		FormWins:           15,
		FormSetWins:        35,
		FormSetLosses:      10,
		LANPlacementsCount: 5,
		LastCalculatedAt:   &now,
	}
	if err := rankingRepo.Create(ctx, ranking); err != nil {
		t.Fatalf("failed to create ranking: %v", err)
	}

	tests := []struct {
		name     string
		memberID uint64
		systemID uint64
		wantErr  error
	}{
		{
			name:     "get existing ranking",
			memberID: memberID,
			systemID: system.ID,
			wantErr:  nil,
		},
		{
			name:     "get non-existent member",
			memberID: 99999,
			systemID: system.ID,
			wantErr:  ErrMemberPRNotFound,
		},
		{
			name:     "get non-existent system",
			memberID: memberID,
			systemID: 99999,
			wantErr:  ErrMemberPRNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rankingRepo.GetByMemberAndSystem(ctx, tt.memberID, tt.systemID)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetByMemberAndSystem() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetByMemberAndSystem() unexpected error = %v", err)
				return
			}

			if got.ID != ranking.ID {
				t.Errorf("GetByMemberAndSystem() ID = %v, want %v", got.ID, ranking.ID)
			}
			if got.TotalPoints != ranking.TotalPoints {
				t.Errorf("GetByMemberAndSystem() TotalPoints = %v, want %v", got.TotalPoints, ranking.TotalPoints)
			}
			// Check that member info is joined
			if got.MemberDisplayName == nil {
				t.Error("GetByMemberAndSystem() should join member display name")
			}
		})
	}
}

func TestMemberPowerRankingRepository_GetLeaderboard(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	rankingRepo := NewMemberPowerRankingRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Test System",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                true,
		IsActive:                 true,
	}
	if err := prRepo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create PR system: %v", err)
	}

	// Create members with different rankings
	members := []struct {
		userID      uint64
		displayName string
		totalPoints float64
	}{
		{1, "Top Player", 2000},
		{2, "Second Player", 1500},
		{3, "Third Player", 1000},
		{4, "Fourth Player", 500},
		{5, "Zero Points", 0}, // Should be excluded
	}

	for _, m := range members {
		memberID := createTestMember(t, db, communityID, m.userID, m.displayName)
		ranking := &domain.MemberPowerRanking{
			MemberID:          memberID,
			PRSystemID:        system.ID,
			TotalPoints:       m.totalPoints,
			AchievementsScore: m.totalPoints * 0.5,
			FormScore:         m.totalPoints * 0.3,
			LANScore:          m.totalPoints * 0.2,
		}
		if err := rankingRepo.Create(ctx, ranking); err != nil {
			t.Fatalf("failed to create ranking: %v", err)
		}
	}

	// Get leaderboard with limit
	got, err := rankingRepo.GetLeaderboard(ctx, system.ID, 3)
	if err != nil {
		t.Fatalf("GetLeaderboard() error = %v", err)
	}

	if len(got) != 3 {
		t.Errorf("GetLeaderboard() returned %d entries, want 3", len(got))
	}

	// Should be ordered by total_points DESC
	if len(got) >= 2 && got[0].TotalPoints < got[1].TotalPoints {
		t.Error("GetLeaderboard() should order by total_points DESC")
	}

	// First should have highest points
	if got[0].TotalPoints != 2000 {
		t.Errorf("GetLeaderboard() top entry has %v points, want 2000", got[0].TotalPoints)
	}

	// Member info should be joined
	if got[0].MemberDisplayName == nil || *got[0].MemberDisplayName != "Top Player" {
		t.Error("GetLeaderboard() should join member display name")
	}

	// Get all (should exclude zero points)
	all, err := rankingRepo.GetLeaderboard(ctx, system.ID, 100)
	if err != nil {
		t.Fatalf("GetLeaderboard() error = %v", err)
	}

	if len(all) != 4 {
		t.Errorf("GetLeaderboard() returned %d entries, want 4 (excluding zero points)", len(all))
	}
}

func TestMemberPowerRankingRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	rankingRepo := NewMemberPowerRankingRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	memberID := createTestMember(t, db, communityID, 1, "Player One")

	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Test System",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                true,
		IsActive:                 true,
	}
	if err := prRepo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create PR system: %v", err)
	}

	ranking := &domain.MemberPowerRanking{
		MemberID:           memberID,
		PRSystemID:         system.ID,
		TotalPoints:        1000,
		Rank:               ptrInt(5),
		AchievementsScore:  500,
		FormScore:          300,
		LANScore:           200,
		FormWins:           10,
		FormSetWins:        25,
		FormSetLosses:      15,
		LANPlacementsCount: 3,
	}
	if err := rankingRepo.Create(ctx, ranking); err != nil {
		t.Fatalf("failed to create ranking: %v", err)
	}

	// Update
	now := time.Now()
	ranking.TotalPoints = 1500
	ranking.Rank = ptrInt(3)
	ranking.AchievementsScore = 750
	ranking.FormWins = 15
	ranking.LastCalculatedAt = &now

	if err := rankingRepo.Update(ctx, ranking); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	got, err := rankingRepo.GetByMemberAndSystem(ctx, memberID, system.ID)
	if err != nil {
		t.Fatalf("GetByMemberAndSystem() error = %v", err)
	}

	if got.TotalPoints != 1500 {
		t.Errorf("Update() TotalPoints = %v, want 1500", got.TotalPoints)
	}
	if *got.Rank != 3 {
		t.Errorf("Update() Rank = %v, want 3", *got.Rank)
	}
	if got.FormWins != 15 {
		t.Errorf("Update() FormWins = %v, want 15", got.FormWins)
	}
}

func TestMemberPowerRankingRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	rankingRepo := NewMemberPowerRankingRepository(db)
	ctx := context.Background()

	ranking := &domain.MemberPowerRanking{
		ID:                99999,
		TotalPoints:       1000,
		AchievementsScore: 500,
		FormScore:         300,
		LANScore:          200,
	}

	err := rankingRepo.Update(ctx, ranking)
	if err != ErrMemberPRNotFound {
		t.Errorf("Update() should return ErrMemberPRNotFound for non-existent ID, got %v", err)
	}
}

func TestMemberPowerRankingRepository_Upsert(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	rankingRepo := NewMemberPowerRankingRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	memberID := createTestMember(t, db, communityID, 1, "Player One")

	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Test System",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                true,
		IsActive:                 true,
	}
	if err := prRepo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create PR system: %v", err)
	}

	ranking := &domain.MemberPowerRanking{
		MemberID:           memberID,
		PRSystemID:         system.ID,
		TotalPoints:        1000,
		AchievementsScore:  500,
		FormScore:          300,
		LANScore:           200,
		FormWins:           10,
		FormSetWins:        25,
		FormSetLosses:      15,
		LANPlacementsCount: 3,
	}

	// Insert via Upsert
	if err := rankingRepo.Upsert(ctx, ranking); err != nil {
		t.Fatalf("Upsert() insert error = %v", err)
	}

	originalID := ranking.ID
	if originalID == 0 {
		t.Error("Upsert() should set ID on insert")
	}

	// Update via Upsert (same member + system)
	ranking.TotalPoints = 1500
	ranking.FormWins = 15

	if err := rankingRepo.Upsert(ctx, ranking); err != nil {
		t.Fatalf("Upsert() update error = %v", err)
	}

	// ID should remain the same
	if ranking.ID != originalID {
		t.Errorf("Upsert() ID changed from %d to %d, should remain same", originalID, ranking.ID)
	}

	// Verify update
	got, err := rankingRepo.GetByMemberAndSystem(ctx, memberID, system.ID)
	if err != nil {
		t.Fatalf("GetByMemberAndSystem() error = %v", err)
	}

	if got.TotalPoints != 1500 {
		t.Errorf("Upsert() TotalPoints = %v, want 1500", got.TotalPoints)
	}
	if got.FormWins != 15 {
		t.Errorf("Upsert() FormWins = %v, want 15", got.FormWins)
	}
}

func TestMemberPowerRankingRepository_RecalculateRanks(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	rankingRepo := NewMemberPowerRankingRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Test System",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                true,
		IsActive:                 true,
	}
	if err := prRepo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create PR system: %v", err)
	}

	// Create members with unordered ranks
	members := []struct {
		userID      uint64
		displayName string
		totalPoints float64
	}{
		{1, "Should be #2", 1500},
		{2, "Should be #1", 2000},
		{3, "Should be #3", 1000},
	}

	memberIDs := make([]uint64, len(members))
	for i, m := range members {
		memberIDs[i] = createTestMember(t, db, communityID, m.userID, m.displayName)
		ranking := &domain.MemberPowerRanking{
			MemberID:          memberIDs[i],
			PRSystemID:        system.ID,
			TotalPoints:       m.totalPoints,
			AchievementsScore: m.totalPoints * 0.5,
			FormScore:         m.totalPoints * 0.3,
			LANScore:          m.totalPoints * 0.2,
			Rank:              ptrInt(99), // Wrong rank initially
		}
		if err := rankingRepo.Create(ctx, ranking); err != nil {
			t.Fatalf("failed to create ranking: %v", err)
		}
	}

	// Recalculate ranks
	if err := rankingRepo.RecalculateRanks(ctx, system.ID); err != nil {
		t.Fatalf("RecalculateRanks() error = %v", err)
	}

	// Verify ranks
	// Member with 2000 points should be rank 1
	r1, _ := rankingRepo.GetByMemberAndSystem(ctx, memberIDs[1], system.ID)
	if r1.Rank == nil || *r1.Rank != 1 {
		t.Errorf("RecalculateRanks() member with 2000 points has rank %v, want 1", r1.Rank)
	}

	// Member with 1500 points should be rank 2
	r2, _ := rankingRepo.GetByMemberAndSystem(ctx, memberIDs[0], system.ID)
	if r2.Rank == nil || *r2.Rank != 2 {
		t.Errorf("RecalculateRanks() member with 1500 points has rank %v, want 2", r2.Rank)
	}

	// Member with 1000 points should be rank 3
	r3, _ := rankingRepo.GetByMemberAndSystem(ctx, memberIDs[2], system.ID)
	if r3.Rank == nil || *r3.Rank != 3 {
		t.Errorf("RecalculateRanks() member with 1000 points has rank %v, want 3", r3.Rank)
	}
}

func TestMemberPowerRankingRepository_DeleteBySystem(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	rankingRepo := NewMemberPowerRankingRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	memberID := createTestMember(t, db, communityID, 1, "Player One")

	system1 := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "System 1",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                true,
		IsActive:                 true,
	}
	system2 := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "System 2",
		AchievementsWeight:       40,
		FormWeight:               40,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                false,
		IsActive:                 true,
	}
	if err := prRepo.Create(ctx, system1); err != nil {
		t.Fatalf("failed to create system1: %v", err)
	}
	if err := prRepo.Create(ctx, system2); err != nil {
		t.Fatalf("failed to create system2: %v", err)
	}

	// Create rankings for both systems
	ranking1 := &domain.MemberPowerRanking{
		MemberID:          memberID,
		PRSystemID:        system1.ID,
		TotalPoints:       1000,
		AchievementsScore: 500,
		FormScore:         300,
		LANScore:          200,
	}
	ranking2 := &domain.MemberPowerRanking{
		MemberID:          memberID,
		PRSystemID:        system2.ID,
		TotalPoints:       800,
		AchievementsScore: 400,
		FormScore:         240,
		LANScore:          160,
	}
	if err := rankingRepo.Create(ctx, ranking1); err != nil {
		t.Fatalf("failed to create ranking1: %v", err)
	}
	if err := rankingRepo.Create(ctx, ranking2); err != nil {
		t.Fatalf("failed to create ranking2: %v", err)
	}

	// Delete by system1
	if err := rankingRepo.DeleteBySystem(ctx, system1.ID); err != nil {
		t.Fatalf("DeleteBySystem() error = %v", err)
	}

	// Verify system1 ranking is gone
	_, err := rankingRepo.GetByMemberAndSystem(ctx, memberID, system1.ID)
	if err != ErrMemberPRNotFound {
		t.Error("DeleteBySystem() should have deleted system1 ranking")
	}

	// Verify system2 ranking still exists
	got, err := rankingRepo.GetByMemberAndSystem(ctx, memberID, system2.ID)
	if err != nil {
		t.Errorf("DeleteBySystem() incorrectly deleted system2 ranking: %v", err)
	}
	if got.TotalPoints != 800 {
		t.Errorf("System2 ranking TotalPoints = %v, want 800", got.TotalPoints)
	}
}

func TestMemberPowerRankingRepository_GetMembersWithActivity(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	placementRepo := NewTournamentPlacementRepository(db)
	matchCacheRepo := NewPowerRankingMatchCacheRepository(db)
	rankingRepo := NewMemberPowerRankingRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	member1ID := createTestMember(t, db, communityID, 1, "Player One")
	member2ID := createTestMember(t, db, communityID, 2, "Player Two")
	member3ID := createTestMember(t, db, communityID, 3, "Player Three (No Activity)")

	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Test System",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                true,
		IsActive:                 true,
	}
	if err := prRepo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create PR system: %v", err)
	}

	now := time.Now()

	// Member 1 has a placement
	placement := &domain.TournamentPlacement{
		MemberID:         member1ID,
		PRSystemID:       system.ID,
		TournamentID:     100,
		Placement:        1,
		ParticipantCount: 16,
		TournamentClass:  "regional",
		IsLAN:            false,
		RawPoints:        500,
		CompletedAt:      now,
	}
	if err := placementRepo.Create(ctx, placement); err != nil {
		t.Fatalf("failed to create placement: %v", err)
	}

	// Member 2 has a match cache entry
	matchCache := &domain.PowerRankingMatchCache{
		MemberID:     member2ID,
		PRSystemID:   system.ID,
		MatchID:      1000,
		TournamentID: 200,
		IsWinner:     true,
		SetsWon:      2,
		SetsLost:     0,
		IsLAN:        false,
		CompletedAt:  now,
	}
	if err := matchCacheRepo.Create(ctx, matchCache); err != nil {
		t.Fatalf("failed to create match cache: %v", err)
	}

	// Member 3 has no activity (not used in test data)
	_ = member3ID

	// Get members with activity
	got, err := rankingRepo.GetMembersWithActivity(ctx, system.ID)
	if err != nil {
		t.Fatalf("GetMembersWithActivity() error = %v", err)
	}

	if len(got) != 2 {
		t.Errorf("GetMembersWithActivity() returned %d members, want 2", len(got))
	}

	// Should contain member1 and member2
	hasM1, hasM2 := false, false
	for _, id := range got {
		if id == member1ID {
			hasM1 = true
		}
		if id == member2ID {
			hasM2 = true
		}
	}

	if !hasM1 {
		t.Error("GetMembersWithActivity() should include member1 (has placement)")
	}
	if !hasM2 {
		t.Error("GetMembersWithActivity() should include member2 (has match cache)")
	}
}
