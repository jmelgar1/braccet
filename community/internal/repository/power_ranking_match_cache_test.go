//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/braccet/community/internal/domain"
)

func TestPowerRankingMatchCacheRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	cacheRepo := NewPowerRankingMatchCacheRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	memberID := createTestMember(t, db, communityID, 1, "Player One")
	opponentID := createTestMember(t, db, communityID, 2, "Opponent")

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
		name    string
		cache   *domain.PowerRankingMatchCache
		wantErr bool
	}{
		{
			name: "create valid cache entry - winner",
			cache: &domain.PowerRankingMatchCache{
				MemberID:         memberID,
				PRSystemID:       system.ID,
				MatchID:          1000,
				TournamentID:     100,
				IsWinner:         true,
				SetsWon:          2,
				SetsLost:         1,
				OpponentMemberID: &opponentID,
				OpponentRating:   ptrInt(1500),
				IsLAN:            true,
				CompletedAt:      time.Now().Add(-24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "create valid cache entry - loser",
			cache: &domain.PowerRankingMatchCache{
				MemberID:         memberID,
				PRSystemID:       system.ID,
				MatchID:          1001,
				TournamentID:     100,
				IsWinner:         false,
				SetsWon:          0,
				SetsLost:         2,
				OpponentMemberID: &opponentID,
				OpponentRating:   ptrInt(1600),
				IsLAN:            false,
				CompletedAt:      time.Now().Add(-48 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "create cache entry without opponent",
			cache: &domain.PowerRankingMatchCache{
				MemberID:     memberID,
				PRSystemID:   system.ID,
				MatchID:      1002,
				TournamentID: 101,
				IsWinner:     true,
				SetsWon:      2,
				SetsLost:     0,
				IsLAN:        false,
				CompletedAt:  time.Now().Add(-72 * time.Hour),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cacheRepo.Create(ctx, tt.cache)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.cache.ID == 0 {
					t.Error("Create() should set ID")
				}
				if tt.cache.CreatedAt.IsZero() {
					t.Error("Create() should set CreatedAt")
				}
			}
		})
	}
}

func TestPowerRankingMatchCacheRepository_GetByMemberSince(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	cacheRepo := NewPowerRankingMatchCacheRepository(db)
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

	// Create cache entries at different times
	entries := []struct {
		matchID     uint64
		completedAt time.Time
	}{
		{1000, now.Add(-1 * 24 * time.Hour)},   // 1 day ago
		{1001, now.Add(-7 * 24 * time.Hour)},   // 7 days ago
		{1002, now.Add(-30 * 24 * time.Hour)},  // 30 days ago
		{1003, now.Add(-60 * 24 * time.Hour)},  // 60 days ago
		{1004, now.Add(-120 * 24 * time.Hour)}, // 120 days ago
	}

	for _, e := range entries {
		cache := &domain.PowerRankingMatchCache{
			MemberID:     memberID,
			PRSystemID:   system.ID,
			MatchID:      e.matchID,
			TournamentID: 100,
			IsWinner:     true,
			SetsWon:      2,
			SetsLost:     0,
			IsLAN:        false,
			CompletedAt:  e.completedAt,
		}
		if err := cacheRepo.Create(ctx, cache); err != nil {
			t.Fatalf("failed to create cache entry: %v", err)
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
			wantCount: 2,
		},
		{
			name:      "last 30 days",
			since:     now.Add(-30 * 24 * time.Hour),
			wantCount: 3,
		},
		{
			name:      "last 90 days (form window)",
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
			got, err := cacheRepo.GetByMemberSince(ctx, memberID, system.ID, tt.since)
			if err != nil {
				t.Fatalf("GetByMemberSince() error = %v", err)
			}

			if len(got) != tt.wantCount {
				t.Errorf("GetByMemberSince() returned %d entries, want %d", len(got), tt.wantCount)
			}
		})
	}

	// Verify ordering (should be DESC by completed_at)
	got, _ := cacheRepo.GetByMemberSince(ctx, memberID, system.ID, now.Add(-365*24*time.Hour))
	if len(got) >= 2 && got[0].CompletedAt.Before(got[1].CompletedAt) {
		t.Error("GetByMemberSince() should order by completed_at DESC")
	}
}

func TestPowerRankingMatchCacheRepository_GetByTournament(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	cacheRepo := NewPowerRankingMatchCacheRepository(db)
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

	// Create cache entries for tournament 1
	for i, memberID := range []uint64{member1ID, member2ID} {
		cache := &domain.PowerRankingMatchCache{
			MemberID:     memberID,
			PRSystemID:   system.ID,
			MatchID:      uint64(1000 + i),
			TournamentID: tournament1ID,
			IsWinner:     i == 0,
			SetsWon:      2 - i,
			SetsLost:     i,
			IsLAN:        true,
			CompletedAt:  now,
		}
		if err := cacheRepo.Create(ctx, cache); err != nil {
			t.Fatalf("failed to create cache entry: %v", err)
		}
	}

	// Create cache entry for tournament 2
	cache := &domain.PowerRankingMatchCache{
		MemberID:     member1ID,
		PRSystemID:   system.ID,
		MatchID:      2000,
		TournamentID: tournament2ID,
		IsWinner:     true,
		SetsWon:      2,
		SetsLost:     0,
		IsLAN:        false,
		CompletedAt:  now,
	}
	if err := cacheRepo.Create(ctx, cache); err != nil {
		t.Fatalf("failed to create cache entry: %v", err)
	}

	// Get by tournament 1
	got1, err := cacheRepo.GetByTournament(ctx, tournament1ID)
	if err != nil {
		t.Fatalf("GetByTournament() error = %v", err)
	}

	if len(got1) != 2 {
		t.Errorf("GetByTournament() returned %d entries for tournament 1, want 2", len(got1))
	}

	// Get by tournament 2
	got2, err := cacheRepo.GetByTournament(ctx, tournament2ID)
	if err != nil {
		t.Fatalf("GetByTournament() error = %v", err)
	}

	if len(got2) != 1 {
		t.Errorf("GetByTournament() returned %d entries for tournament 2, want 1", len(got2))
	}

	// Non-existent tournament
	got3, err := cacheRepo.GetByTournament(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByTournament() error = %v", err)
	}

	if len(got3) != 0 {
		t.Errorf("GetByTournament() returned %d entries for non-existent tournament, want 0", len(got3))
	}
}

func TestPowerRankingMatchCacheRepository_Upsert(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	cacheRepo := NewPowerRankingMatchCacheRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)
	memberID := createTestMember(t, db, communityID, 1, "Player One")
	opponentID := createTestMember(t, db, communityID, 2, "Opponent")

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
	matchID := uint64(1000)

	// Initial cache entry
	cache := &domain.PowerRankingMatchCache{
		MemberID:         memberID,
		PRSystemID:       system.ID,
		MatchID:          matchID,
		TournamentID:     100,
		IsWinner:         false, // Initially lost
		SetsWon:          1,
		SetsLost:         2,
		OpponentMemberID: &opponentID,
		OpponentRating:   ptrInt(1500),
		IsLAN:            true,
		CompletedAt:      now,
	}

	// Insert via Upsert
	if err := cacheRepo.Upsert(ctx, cache); err != nil {
		t.Fatalf("Upsert() insert error = %v", err)
	}

	originalID := cache.ID
	if originalID == 0 {
		t.Error("Upsert() should set ID on insert")
	}

	// Update via Upsert (result correction - actually won)
	cache.IsWinner = true
	cache.SetsWon = 2
	cache.SetsLost = 1
	cache.OpponentRating = ptrInt(1600) // Corrected rating

	if err := cacheRepo.Upsert(ctx, cache); err != nil {
		t.Fatalf("Upsert() update error = %v", err)
	}

	// ID should remain the same
	if cache.ID != originalID {
		t.Errorf("Upsert() ID changed from %d to %d, should remain same", originalID, cache.ID)
	}

	// Verify update
	entries, err := cacheRepo.GetByTournament(ctx, 100)
	if err != nil {
		t.Fatalf("GetByTournament() error = %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Upsert() created duplicate, got %d entries, want 1", len(entries))
		return
	}

	got := entries[0]
	if !got.IsWinner {
		t.Error("Upsert() IsWinner = false, want true")
	}
	if got.SetsWon != 2 {
		t.Errorf("Upsert() SetsWon = %d, want 2", got.SetsWon)
	}
	if got.SetsLost != 1 {
		t.Errorf("Upsert() SetsLost = %d, want 1", got.SetsLost)
	}
	if got.OpponentRating == nil || *got.OpponentRating != 1600 {
		t.Errorf("Upsert() OpponentRating = %v, want 1600", got.OpponentRating)
	}
}

func TestPowerRankingMatchCacheRepository_DeleteByTournament(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	cacheRepo := NewPowerRankingMatchCacheRepository(db)
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

	tournament1ID := uint64(100)
	tournament2ID := uint64(200)
	now := time.Now()

	// Create cache entries for tournament 1
	for i := 0; i < 3; i++ {
		cache := &domain.PowerRankingMatchCache{
			MemberID:     memberID,
			PRSystemID:   system.ID,
			MatchID:      uint64(1000 + i),
			TournamentID: tournament1ID,
			IsWinner:     true,
			SetsWon:      2,
			SetsLost:     0,
			IsLAN:        false,
			CompletedAt:  now,
		}
		if err := cacheRepo.Create(ctx, cache); err != nil {
			t.Fatalf("failed to create cache entry: %v", err)
		}
	}

	// Create cache entry for tournament 2
	cache := &domain.PowerRankingMatchCache{
		MemberID:     memberID,
		PRSystemID:   system.ID,
		MatchID:      2000,
		TournamentID: tournament2ID,
		IsWinner:     true,
		SetsWon:      2,
		SetsLost:     0,
		IsLAN:        false,
		CompletedAt:  now,
	}
	if err := cacheRepo.Create(ctx, cache); err != nil {
		t.Fatalf("failed to create cache entry: %v", err)
	}

	// Delete tournament 1 cache
	if err := cacheRepo.DeleteByTournament(ctx, tournament1ID); err != nil {
		t.Fatalf("DeleteByTournament() error = %v", err)
	}

	// Verify tournament 1 cache is gone
	got1, _ := cacheRepo.GetByTournament(ctx, tournament1ID)
	if len(got1) != 0 {
		t.Errorf("DeleteByTournament() left %d entries, want 0", len(got1))
	}

	// Verify tournament 2 cache still exists
	got2, _ := cacheRepo.GetByTournament(ctx, tournament2ID)
	if len(got2) != 1 {
		t.Errorf("DeleteByTournament() affected other tournament, got %d entries, want 1", len(got2))
	}
}

func TestPowerRankingMatchCacheRepository_DeleteByMatch(t *testing.T) {
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	cacheRepo := NewPowerRankingMatchCacheRepository(db)
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

	now := time.Now()
	matchID := uint64(1000)
	otherMatchID := uint64(1001)

	// Create cache entries for both players in the same match
	for _, memberID := range []uint64{member1ID, member2ID} {
		cache := &domain.PowerRankingMatchCache{
			MemberID:     memberID,
			PRSystemID:   system.ID,
			MatchID:      matchID,
			TournamentID: 100,
			IsWinner:     memberID == member1ID,
			SetsWon:      2,
			SetsLost:     0,
			IsLAN:        false,
			CompletedAt:  now,
		}
		if err := cacheRepo.Create(ctx, cache); err != nil {
			t.Fatalf("failed to create cache entry: %v", err)
		}
	}

	// Create cache entry for different match
	cache := &domain.PowerRankingMatchCache{
		MemberID:     member1ID,
		PRSystemID:   system.ID,
		MatchID:      otherMatchID,
		TournamentID: 100,
		IsWinner:     true,
		SetsWon:      2,
		SetsLost:     0,
		IsLAN:        false,
		CompletedAt:  now,
	}
	if err := cacheRepo.Create(ctx, cache); err != nil {
		t.Fatalf("failed to create cache entry: %v", err)
	}

	// Delete by match (should delete both player entries)
	if err := cacheRepo.DeleteByMatch(ctx, matchID); err != nil {
		t.Fatalf("DeleteByMatch() error = %v", err)
	}

	// Verify match entries are gone
	got, _ := cacheRepo.GetByTournament(ctx, 100)

	// Should only have the other match entry left
	if len(got) != 1 {
		t.Errorf("DeleteByMatch() left %d entries, want 1 (other match only)", len(got))
	}

	if len(got) > 0 && got[0].MatchID != otherMatchID {
		t.Error("DeleteByMatch() deleted wrong match")
	}
}

func TestPowerRankingMatchCacheRepository_FormCalculationScenario(t *testing.T) {
	// Test a realistic form calculation scenario
	db := setupTestDB(t)
	prRepo := NewPowerRankingSystemRepository(db)
	cacheRepo := NewPowerRankingMatchCacheRepository(db)
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
	formWindow := now.Add(-90 * 24 * time.Hour) // 3 months ago

	// Create realistic match history
	matches := []struct {
		matchID        uint64
		isWinner       bool
		setsWon        int
		setsLost       int
		opponentRating int
		daysAgo        int
	}{
		{1000, true, 2, 0, 1800, 5},   // Recent big win
		{1001, true, 2, 1, 1500, 10},  // Win
		{1002, false, 1, 2, 1600, 20}, // Loss
		{1003, true, 2, 0, 1400, 30},  // Win
		{1004, true, 2, 1, 1550, 60},  // Win
		{1005, false, 0, 2, 1700, 80}, // Loss (still in form window)
		{1006, true, 2, 0, 1450, 100}, // Outside form window
	}

	for _, m := range matches {
		opponentRating := m.opponentRating
		cache := &domain.PowerRankingMatchCache{
			MemberID:       memberID,
			PRSystemID:     system.ID,
			MatchID:        m.matchID,
			TournamentID:   100,
			IsWinner:       m.isWinner,
			SetsWon:        m.setsWon,
			SetsLost:       m.setsLost,
			OpponentRating: &opponentRating,
			IsLAN:          false,
			CompletedAt:    now.Add(-time.Duration(m.daysAgo) * 24 * time.Hour),
		}
		if err := cacheRepo.Create(ctx, cache); err != nil {
			t.Fatalf("failed to create cache entry: %v", err)
		}
	}

	// Get form window matches
	got, err := cacheRepo.GetByMemberSince(ctx, memberID, system.ID, formWindow)
	if err != nil {
		t.Fatalf("GetByMemberSince() error = %v", err)
	}

	// Should have 6 matches in form window (excluding 100 days ago)
	if len(got) != 6 {
		t.Errorf("GetByMemberSince() returned %d matches in form window, want 6", len(got))
	}

	// Calculate form metrics from results
	var wins, losses, setWins, setLosses, totalOpponentRating, bestWinRating int
	for _, m := range got {
		if m.IsWinner {
			wins++
			if m.OpponentRating != nil && *m.OpponentRating > bestWinRating {
				bestWinRating = *m.OpponentRating
			}
		} else {
			losses++
		}
		setWins += m.SetsWon
		setLosses += m.SetsLost
		if m.OpponentRating != nil {
			totalOpponentRating += *m.OpponentRating
		}
	}

	if wins != 4 {
		t.Errorf("Form window wins = %d, want 4", wins)
	}
	if losses != 2 {
		t.Errorf("Form window losses = %d, want 2", losses)
	}
	if setWins != 9 {
		t.Errorf("Form window set wins = %d, want 9", setWins)
	}
	if setLosses != 6 {
		t.Errorf("Form window set losses = %d, want 6", setLosses)
	}
	if bestWinRating != 1800 {
		t.Errorf("Form window best win rating = %d, want 1800", bestWinRating)
	}
}
