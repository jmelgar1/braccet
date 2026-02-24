//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/braccet/community/internal/domain"
)

func TestTournamentPlacementRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	placementRepo := NewTournamentPlacementRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	memberID := createTestMember(t, db, communityID, 1, "Player One")

	// Create a power ranking system
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

	tests := []struct {
		name      string
		placement *domain.TournamentPlacement
		wantErr   bool
	}{
		{
			name: "create valid placement",
			placement: &domain.TournamentPlacement{
				MemberID:          memberID,
				PRSystemID:        system.ID,
				TournamentID:      100,
				Placement:         1,
				ParticipantCount:  32,
				TournamentClass:   "major",
				PrizePoolUSD:      ptrFloat64(50000),
				IsBo3Playoffs:     true,
				AvgOpponentRating: ptrInt(1500),
				IsLAN:             true,
				RawPoints:         1000,
				CompletedAt:       time.Now().Add(-24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "create placement with minimal fields",
			placement: &domain.TournamentPlacement{
				MemberID:         memberID,
				PRSystemID:       system.ID,
				TournamentID:     101,
				Placement:        5,
				ParticipantCount: 16,
				TournamentClass:  "online",
				IsLAN:            false,
				RawPoints:        300,
				CompletedAt:      time.Now().Add(-48 * time.Hour),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := placementRepo.Create(ctx, tt.placement)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.placement.ID == 0 {
					t.Error("Create() should set ID")
				}
				if tt.placement.CreatedAt.IsZero() {
					t.Error("Create() should set CreatedAt")
				}
			}
		})
	}
}

func TestTournamentPlacementRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	placementRepo := NewTournamentPlacementRepository(db)
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

	placement := &domain.TournamentPlacement{
		MemberID:          memberID,
		PRSystemID:        system.ID,
		TournamentID:      100,
		Placement:         1,
		ParticipantCount:  32,
		TournamentClass:   "major",
		PrizePoolUSD:      ptrFloat64(50000),
		IsBo3Playoffs:     true,
		AvgOpponentRating: ptrInt(1500),
		IsLAN:             true,
		RawPoints:         1000,
		CompletedAt:       time.Now().Add(-24 * time.Hour),
	}
	if err := placementRepo.Create(ctx, placement); err != nil {
		t.Fatalf("failed to create placement: %v", err)
	}

	tests := []struct {
		name    string
		id      uint64
		wantErr error
	}{
		{
			name:    "get existing placement",
			id:      placement.ID,
			wantErr: nil,
		},
		{
			name:    "get non-existent placement",
			id:      99999,
			wantErr: ErrPlacementNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := placementRepo.GetByID(ctx, tt.id)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetByID() unexpected error = %v", err)
				return
			}

			if got.ID != placement.ID {
				t.Errorf("GetByID() ID = %v, want %v", got.ID, placement.ID)
			}
			if got.Placement != placement.Placement {
				t.Errorf("GetByID() Placement = %v, want %v", got.Placement, placement.Placement)
			}
			if got.TournamentClass != placement.TournamentClass {
				t.Errorf("GetByID() TournamentClass = %v, want %v", got.TournamentClass, placement.TournamentClass)
			}
		})
	}
}

func TestTournamentPlacementRepository_GetByMember(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	placementRepo := NewTournamentPlacementRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	member1ID := createTestMember(t, db, communityID, 1, "Player One")
	member2ID := createTestMember(t, db, communityID, 2, "Player Two")

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

	// Create placements for member1
	for i, tournament := range []uint64{100, 101, 102} {
		placement := &domain.TournamentPlacement{
			MemberID:         member1ID,
			PRSystemID:       system.ID,
			TournamentID:     tournament,
			Placement:        i + 1,
			ParticipantCount: 32,
			TournamentClass:  "regional",
			IsLAN:            false,
			RawPoints:        100 * (3 - i),
			CompletedAt:      time.Now().Add(-time.Duration(i*24) * time.Hour),
		}
		if err := placementRepo.Create(ctx, placement); err != nil {
			t.Fatalf("failed to create placement: %v", err)
		}
	}

	// Create placement for member2
	placement := &domain.TournamentPlacement{
		MemberID:         member2ID,
		PRSystemID:       system.ID,
		TournamentID:     200,
		Placement:        1,
		ParticipantCount: 16,
		TournamentClass:  "online",
		IsLAN:            false,
		RawPoints:        500,
		CompletedAt:      time.Now().Add(-24 * time.Hour),
	}
	if err := placementRepo.Create(ctx, placement); err != nil {
		t.Fatalf("failed to create placement: %v", err)
	}

	// Get member1's placements
	got, err := placementRepo.GetByMember(ctx, member1ID, system.ID)
	if err != nil {
		t.Fatalf("GetByMember() error = %v", err)
	}

	if len(got) != 3 {
		t.Errorf("GetByMember() returned %d placements, want 3", len(got))
	}

	// Should be ordered by completed_at DESC (most recent first)
	if len(got) >= 2 && got[0].CompletedAt.Before(got[1].CompletedAt) {
		t.Error("GetByMember() should order by completed_at DESC")
	}

	// Get member2's placements
	got2, err := placementRepo.GetByMember(ctx, member2ID, system.ID)
	if err != nil {
		t.Fatalf("GetByMember() error = %v", err)
	}

	if len(got2) != 1 {
		t.Errorf("GetByMember() returned %d placements for member2, want 1", len(got2))
	}

	// Get non-existent member
	got3, err := placementRepo.GetByMember(ctx, 99999, system.ID)
	if err != nil {
		t.Fatalf("GetByMember() error = %v", err)
	}
	if len(got3) != 0 {
		t.Errorf("GetByMember() returned %d placements for non-existent member, want 0", len(got3))
	}
}

func TestTournamentPlacementRepository_GetByMemberSince(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	placementRepo := NewTournamentPlacementRepository(db)
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

	// Create placements at different times
	placements := []struct {
		tournamentID uint64
		completedAt  time.Time
	}{
		{100, now.Add(-1 * 24 * time.Hour)},   // 1 day ago
		{101, now.Add(-7 * 24 * time.Hour)},   // 7 days ago
		{102, now.Add(-30 * 24 * time.Hour)},  // 30 days ago
		{103, now.Add(-90 * 24 * time.Hour)},  // 90 days ago
		{104, now.Add(-180 * 24 * time.Hour)}, // 180 days ago
	}

	for _, p := range placements {
		placement := &domain.TournamentPlacement{
			MemberID:         memberID,
			PRSystemID:       system.ID,
			TournamentID:     p.tournamentID,
			Placement:        1,
			ParticipantCount: 16,
			TournamentClass:  "regional",
			IsLAN:            false,
			RawPoints:        500,
			CompletedAt:      p.completedAt,
		}
		if err := placementRepo.Create(ctx, placement); err != nil {
			t.Fatalf("failed to create placement: %v", err)
		}
	}

	tests := []struct {
		name      string
		since     time.Time
		wantCount int
	}{
		{
			name:      "last 7 days",
			since:     now.Add(-7 * 24 * time.Hour),
			wantCount: 2, // 1 day and 7 days ago
		},
		{
			name:      "last 30 days",
			since:     now.Add(-30 * 24 * time.Hour),
			wantCount: 3,
		},
		{
			name:      "last 90 days",
			since:     now.Add(-90 * 24 * time.Hour),
			wantCount: 4,
		},
		{
			name:      "all time (1 year)",
			since:     now.Add(-365 * 24 * time.Hour),
			wantCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := placementRepo.GetByMemberSince(ctx, memberID, system.ID, tt.since)
			if err != nil {
				t.Fatalf("GetByMemberSince() error = %v", err)
			}

			if len(got) != tt.wantCount {
				t.Errorf("GetByMemberSince() returned %d placements, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestTournamentPlacementRepository_GetLANByMember(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	placementRepo := NewTournamentPlacementRepository(db)
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

	// Create mix of LAN and online placements
	placements := []struct {
		tournamentID uint64
		isLAN        bool
		completedAt  time.Time
	}{
		{100, true, now.Add(-1 * 24 * time.Hour)},
		{101, false, now.Add(-2 * 24 * time.Hour)},
		{102, true, now.Add(-3 * 24 * time.Hour)},
		{103, true, now.Add(-4 * 24 * time.Hour)},
		{104, false, now.Add(-5 * 24 * time.Hour)},
		{105, true, now.Add(-6 * 24 * time.Hour)},
	}

	for _, p := range placements {
		placement := &domain.TournamentPlacement{
			MemberID:         memberID,
			PRSystemID:       system.ID,
			TournamentID:     p.tournamentID,
			Placement:        1,
			ParticipantCount: 16,
			TournamentClass:  "world_lan",
			IsLAN:            p.isLAN,
			RawPoints:        500,
			CompletedAt:      p.completedAt,
		}
		if err := placementRepo.Create(ctx, placement); err != nil {
			t.Fatalf("failed to create placement: %v", err)
		}
	}

	// Get last 10 LAN placements (should return 4)
	got, err := placementRepo.GetLANByMember(ctx, memberID, system.ID, 10)
	if err != nil {
		t.Fatalf("GetLANByMember() error = %v", err)
	}

	if len(got) != 4 {
		t.Errorf("GetLANByMember() returned %d placements, want 4", len(got))
	}

	// All should be LAN
	for _, p := range got {
		if !p.IsLAN {
			t.Error("GetLANByMember() returned non-LAN placement")
		}
	}

	// Get last 2 LAN placements
	got2, err := placementRepo.GetLANByMember(ctx, memberID, system.ID, 2)
	if err != nil {
		t.Fatalf("GetLANByMember() error = %v", err)
	}

	if len(got2) != 2 {
		t.Errorf("GetLANByMember() with limit 2 returned %d placements, want 2", len(got2))
	}
}

func TestTournamentPlacementRepository_GetByTournament(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	placementRepo := NewTournamentPlacementRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	member1ID := createTestMember(t, db, communityID, 1, "Player One")
	member2ID := createTestMember(t, db, communityID, 2, "Player Two")
	member3ID := createTestMember(t, db, communityID, 3, "Player Three")

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

	tournamentID := uint64(100)
	now := time.Now()

	// Create placements for tournament
	members := []struct {
		memberID  uint64
		placement int
	}{
		{member1ID, 1},
		{member2ID, 2},
		{member3ID, 3},
	}

	for _, m := range members {
		placement := &domain.TournamentPlacement{
			MemberID:         m.memberID,
			PRSystemID:       system.ID,
			TournamentID:     tournamentID,
			Placement:        m.placement,
			ParticipantCount: 32,
			TournamentClass:  "major",
			IsLAN:            true,
			RawPoints:        1000 / m.placement,
			CompletedAt:      now,
		}
		if err := placementRepo.Create(ctx, placement); err != nil {
			t.Fatalf("failed to create placement: %v", err)
		}
	}

	// Get placements for tournament
	got, err := placementRepo.GetByTournament(ctx, tournamentID)
	if err != nil {
		t.Fatalf("GetByTournament() error = %v", err)
	}

	if len(got) != 3 {
		t.Errorf("GetByTournament() returned %d placements, want 3", len(got))
	}

	// Should be ordered by placement ASC
	for i := 0; i < len(got)-1; i++ {
		if got[i].Placement > got[i+1].Placement {
			t.Error("GetByTournament() should order by placement ASC")
		}
	}
}

func TestTournamentPlacementRepository_Upsert(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	placementRepo := NewTournamentPlacementRepository(db)
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

	tournamentID := uint64(100)
	now := time.Now()

	// Initial placement
	placement := &domain.TournamentPlacement{
		MemberID:         memberID,
		PRSystemID:       system.ID,
		TournamentID:     tournamentID,
		Placement:        3,
		ParticipantCount: 32,
		TournamentClass:  "major",
		IsLAN:            true,
		RawPoints:        500,
		CompletedAt:      now,
	}

	// Insert
	if err := placementRepo.Upsert(ctx, placement); err != nil {
		t.Fatalf("Upsert() insert error = %v", err)
	}

	originalID := placement.ID
	if originalID == 0 {
		t.Error("Upsert() should set ID on insert")
	}

	// Update (same member + tournament)
	placement.Placement = 1 // Better placement
	placement.RawPoints = 1000

	if err := placementRepo.Upsert(ctx, placement); err != nil {
		t.Fatalf("Upsert() update error = %v", err)
	}

	// ID should remain the same (upsert updates existing row)
	if placement.ID != originalID {
		t.Errorf("Upsert() ID changed from %d to %d, should remain same", originalID, placement.ID)
	}

	// Verify update
	got, err := placementRepo.GetByID(ctx, placement.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Placement != 1 {
		t.Errorf("Upsert() Placement = %d, want 1", got.Placement)
	}
	if got.RawPoints != 1000 {
		t.Errorf("Upsert() RawPoints = %d, want 1000", got.RawPoints)
	}

	// Should still only have one placement for this member/tournament
	all, _ := placementRepo.GetByTournament(ctx, tournamentID)
	if len(all) != 1 {
		t.Errorf("Upsert() created duplicate, got %d placements, want 1", len(all))
	}
}

func TestTournamentPlacementRepository_DeleteByTournament(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	placementRepo := NewTournamentPlacementRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	member1ID := createTestMember(t, db, communityID, 1, "Player One")
	member2ID := createTestMember(t, db, communityID, 2, "Player Two")

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

	tournament1ID := uint64(100)
	tournament2ID := uint64(200)
	now := time.Now()

	// Create placements for tournament 1
	for _, memberID := range []uint64{member1ID, member2ID} {
		placement := &domain.TournamentPlacement{
			MemberID:         memberID,
			PRSystemID:       system.ID,
			TournamentID:     tournament1ID,
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
	}

	// Create placement for tournament 2
	placement := &domain.TournamentPlacement{
		MemberID:         member1ID,
		PRSystemID:       system.ID,
		TournamentID:     tournament2ID,
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

	// Delete tournament 1 placements
	if err := placementRepo.DeleteByTournament(ctx, tournament1ID); err != nil {
		t.Fatalf("DeleteByTournament() error = %v", err)
	}

	// Verify tournament 1 placements are gone
	got1, _ := placementRepo.GetByTournament(ctx, tournament1ID)
	if len(got1) != 0 {
		t.Errorf("DeleteByTournament() left %d placements, want 0", len(got1))
	}

	// Verify tournament 2 placement still exists
	got2, _ := placementRepo.GetByTournament(ctx, tournament2ID)
	if len(got2) != 1 {
		t.Errorf("DeleteByTournament() affected other tournament, got %d placements, want 1", len(got2))
	}
}
