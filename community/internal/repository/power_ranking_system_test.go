//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/braccet/community/internal/domain"
)

func TestPowerRankingSystemRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	tests := []struct {
		name    string
		system  *domain.PowerRankingSystem
		wantErr bool
	}{
		{
			name: "create valid system with default weights",
			system: &domain.PowerRankingSystem{
				CommunityID:              communityID,
				Name:                     "Default PR",
				Description:              ptrString("Default power ranking system"),
				AchievementsWeight:       50,
				FormWeight:               30,
				LANWeight:                20,
				AchievementDecayMonths:   4,
				AchievementWindowMonths:  12,
				FormWindowMonths:         3,
				LANResultsCount:          10,
				IsDefault:                true,
				IsActive:                 true,
			},
			wantErr: false,
		},
		{
			name: "create system with custom weights",
			system: &domain.PowerRankingSystem{
				CommunityID:              communityID,
				Name:                     "Custom PR",
				Description:              ptrString("Custom weights"),
				AchievementsWeight:       40,
				FormWeight:               40,
				LANWeight:                20,
				AchievementDecayMonths:   6,
				AchievementWindowMonths:  18,
				FormWindowMonths:         6,
				LANResultsCount:          5,
				IsDefault:                false,
				IsActive:                 true,
			},
			wantErr: false,
		},
		{
			name: "create system with nil description",
			system: &domain.PowerRankingSystem{
				CommunityID:              communityID,
				Name:                     "No Description PR",
				Description:              nil,
				AchievementsWeight:       50,
				FormWeight:               30,
				LANWeight:                20,
				AchievementDecayMonths:   4,
				AchievementWindowMonths:  12,
				FormWindowMonths:         3,
				LANResultsCount:          10,
				IsDefault:                false,
				IsActive:                 true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.system)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.system.ID == 0 {
					t.Error("Create() should set ID")
				}
				if tt.system.CreatedAt.IsZero() {
					t.Error("Create() should set CreatedAt")
				}
				if tt.system.UpdatedAt.IsZero() {
					t.Error("Create() should set UpdatedAt")
				}
			}
		})
	}
}

func TestPowerRankingSystemRepository_Create_WeightsConstraint(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	// Weights don't sum to 100 - should fail
	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Invalid Weights",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                10, // Sum = 90, not 100
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                false,
		IsActive:                 true,
	}

	err := repo.Create(ctx, system)
	if err == nil {
		t.Error("Create() should fail when weights don't sum to 100")
	}
}

func TestPowerRankingSystemRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	// Create a system first
	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Test System",
		Description:              ptrString("Test description"),
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
	if err := repo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create test system: %v", err)
	}

	tests := []struct {
		name    string
		id      uint64
		wantErr error
	}{
		{
			name:    "get existing system",
			id:      system.ID,
			wantErr: nil,
		},
		{
			name:    "get non-existent system",
			id:      99999,
			wantErr: ErrPRSystemNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByID(ctx, tt.id)

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

			if got.ID != system.ID {
				t.Errorf("GetByID() ID = %v, want %v", got.ID, system.ID)
			}
			if got.Name != system.Name {
				t.Errorf("GetByID() Name = %v, want %v", got.Name, system.Name)
			}
			if got.AchievementsWeight != system.AchievementsWeight {
				t.Errorf("GetByID() AchievementsWeight = %v, want %v", got.AchievementsWeight, system.AchievementsWeight)
			}
		})
	}
}

func TestPowerRankingSystemRepository_GetByCommunity(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	// Create multiple systems
	systems := []*domain.PowerRankingSystem{
		{
			CommunityID:              communityID,
			Name:                     "System A",
			AchievementsWeight:       50,
			FormWeight:               30,
			LANWeight:                20,
			AchievementDecayMonths:   4,
			AchievementWindowMonths:  12,
			FormWindowMonths:         3,
			LANResultsCount:          10,
			IsDefault:                true,
			IsActive:                 true,
		},
		{
			CommunityID:              communityID,
			Name:                     "System B",
			AchievementsWeight:       40,
			FormWeight:               40,
			LANWeight:                20,
			AchievementDecayMonths:   4,
			AchievementWindowMonths:  12,
			FormWindowMonths:         3,
			LANResultsCount:          10,
			IsDefault:                false,
			IsActive:                 true,
		},
		{
			CommunityID:              communityID,
			Name:                     "System C (Inactive)",
			AchievementsWeight:       50,
			FormWeight:               30,
			LANWeight:                20,
			AchievementDecayMonths:   4,
			AchievementWindowMonths:  12,
			FormWindowMonths:         3,
			LANResultsCount:          10,
			IsDefault:                false,
			IsActive:                 false, // inactive
		},
	}

	for _, s := range systems {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("failed to create system: %v", err)
		}
	}

	// Get by community - should only return active systems
	got, err := repo.GetByCommunity(ctx, communityID)
	if err != nil {
		t.Fatalf("GetByCommunity() error = %v", err)
	}

	if len(got) != 2 {
		t.Errorf("GetByCommunity() returned %d systems, want 2 (only active)", len(got))
	}

	// First should be default (is_default DESC)
	if !got[0].IsDefault {
		t.Error("GetByCommunity() first result should be the default system")
	}
}

func TestPowerRankingSystemRepository_GetDefaultByCommunity(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	// No default system yet
	_, err := repo.GetDefaultByCommunity(ctx, communityID)
	if err != ErrPRSystemNotFound {
		t.Errorf("GetDefaultByCommunity() should return ErrPRSystemNotFound when no default, got %v", err)
	}

	// Create a default system
	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Default System",
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
	if err := repo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create system: %v", err)
	}

	// Now should find it
	got, err := repo.GetDefaultByCommunity(ctx, communityID)
	if err != nil {
		t.Fatalf("GetDefaultByCommunity() error = %v", err)
	}

	if got.ID != system.ID {
		t.Errorf("GetDefaultByCommunity() ID = %v, want %v", got.ID, system.ID)
	}
	if !got.IsDefault {
		t.Error("GetDefaultByCommunity() should return system with IsDefault = true")
	}
}

func TestPowerRankingSystemRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	// Create a system
	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "Original Name",
		Description:              ptrString("Original description"),
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
	if err := repo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create system: %v", err)
	}

	// Update it
	system.Name = "Updated Name"
	system.Description = ptrString("Updated description")
	system.AchievementsWeight = 40
	system.FormWeight = 40
	// LANWeight stays at 20

	if err := repo.Update(ctx, system); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify update
	got, err := repo.GetByID(ctx, system.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Name != "Updated Name" {
		t.Errorf("Update() Name = %v, want Updated Name", got.Name)
	}
	if *got.Description != "Updated description" {
		t.Errorf("Update() Description = %v, want Updated description", *got.Description)
	}
	if got.AchievementsWeight != 40 {
		t.Errorf("Update() AchievementsWeight = %v, want 40", got.AchievementsWeight)
	}
}

func TestPowerRankingSystemRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	system := &domain.PowerRankingSystem{
		ID:                       99999,
		Name:                     "Ghost System",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsActive:                 true,
	}

	err := repo.Update(ctx, system)
	if err != ErrPRSystemNotFound {
		t.Errorf("Update() should return ErrPRSystemNotFound for non-existent ID, got %v", err)
	}
}

func TestPowerRankingSystemRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	// Create a system
	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "To Delete",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                false,
		IsActive:                 true,
	}
	if err := repo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create system: %v", err)
	}

	// Delete it
	if err := repo.Delete(ctx, system.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	_, err := repo.GetByID(ctx, system.ID)
	if err != ErrPRSystemNotFound {
		t.Errorf("GetByID() after Delete should return ErrPRSystemNotFound, got %v", err)
	}
}

func TestPowerRankingSystemRepository_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err != ErrPRSystemNotFound {
		t.Errorf("Delete() should return ErrPRSystemNotFound for non-existent ID, got %v", err)
	}
}

func TestPowerRankingSystemRepository_SetDefault(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	// Create two systems
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

	if err := repo.Create(ctx, system1); err != nil {
		t.Fatalf("failed to create system1: %v", err)
	}
	if err := repo.Create(ctx, system2); err != nil {
		t.Fatalf("failed to create system2: %v", err)
	}

	// Change default to system2
	if err := repo.SetDefault(ctx, communityID, system2.ID); err != nil {
		t.Fatalf("SetDefault() error = %v", err)
	}

	// Verify system1 is no longer default
	got1, _ := repo.GetByID(ctx, system1.ID)
	if got1.IsDefault {
		t.Error("SetDefault() should clear previous default")
	}

	// Verify system2 is now default
	got2, _ := repo.GetByID(ctx, system2.ID)
	if !got2.IsDefault {
		t.Error("SetDefault() should set new default")
	}

	// GetDefaultByCommunity should return system2
	defaultSys, err := repo.GetDefaultByCommunity(ctx, communityID)
	if err != nil {
		t.Fatalf("GetDefaultByCommunity() error = %v", err)
	}
	if defaultSys.ID != system2.ID {
		t.Errorf("GetDefaultByCommunity() ID = %v, want %v", defaultSys.ID, system2.ID)
	}
}

func TestPowerRankingSystemRepository_SetDefault_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	// Try to set non-existent system as default
	err := repo.SetDefault(ctx, communityID, 99999)
	if err != ErrPRSystemNotFound {
		t.Errorf("SetDefault() should return ErrPRSystemNotFound for non-existent ID, got %v", err)
	}
}

func TestPowerRankingSystemRepository_SetDefault_WrongCommunity(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPowerRankingSystemRepository(db)
	ctx := context.Background()

	communityID := createTestCommunity(t, db)

	// Create a system
	system := &domain.PowerRankingSystem{
		CommunityID:              communityID,
		Name:                     "System",
		AchievementsWeight:       50,
		FormWeight:               30,
		LANWeight:                20,
		AchievementDecayMonths:   4,
		AchievementWindowMonths:  12,
		FormWindowMonths:         3,
		LANResultsCount:          10,
		IsDefault:                false,
		IsActive:                 true,
	}
	if err := repo.Create(ctx, system); err != nil {
		t.Fatalf("failed to create system: %v", err)
	}

	// Try to set default with wrong community ID
	err := repo.SetDefault(ctx, 99999, system.ID)
	if err != ErrPRSystemNotFound {
		t.Errorf("SetDefault() should return ErrPRSystemNotFound when community doesn't match, got %v", err)
	}
}
