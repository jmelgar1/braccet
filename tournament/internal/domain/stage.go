package domain

import "time"

// StageType represents whether a stage is a group stage or the final stage
type StageType string

const (
	StageTypeGroup StageType = "group"
	StageTypeFinal StageType = "final"
)

// VenueType represents the venue type for power ranking LAN tracking
type VenueType string

const (
	VenueTypeLAN    VenueType = "lan"
	VenueTypeOnline VenueType = "online"
	VenueTypeHybrid VenueType = "hybrid"
)

// RankingCriterion represents a criterion used for ranking participants in a stage
type RankingCriterion string

const (
	CriterionMatchWins          RankingCriterion = "match_wins"
	CriterionSetWins            RankingCriterion = "set_wins"
	CriterionSetWinPct          RankingCriterion = "set_win_pct"
	CriterionSetDifferential    RankingCriterion = "set_differential"
	CriterionPointsScored       RankingCriterion = "points_scored"
	CriterionPointsDifferential RankingCriterion = "points_differential"
	CriterionSeed               RankingCriterion = "seed"
)

// TournamentStage represents a stage in a multi-stage tournament
type TournamentStage struct {
	ID                   uint64
	TournamentID         uint64
	StageOrder           int // 1, 2, 3 for group stages; 0 for final stage
	StageType            StageType
	Format               TournamentFormat
	ParticipantsPerGroup *int // NULL for final stage
	AdvancingPerGroup    *int // NULL for final stage
	SwissRounds          *int // Optional override for Swiss format
	WinsToAdvance        *int // For Swiss: wins needed to advance (excluding BYEs)
	LossesToEliminate    *int // For Swiss: losses that eliminate a participant
	VenueType            VenueType // Venue type for power ranking LAN tracking
	SkipFinals           bool      // When true, skip final match(es) and determine standings by bracket position
	IsActive             bool
	IsComplete           bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// StageGroup represents a group within a stage (e.g., "Group A", "Group B")
type StageGroup struct {
	ID           uint64
	TournamentID uint64
	StageID      uint64
	GroupName    string
	GroupOrder   int // For display ordering (0 = Group A, 1 = Group B, etc.)
	CreatedAt    time.Time
}

// GroupAssignment tracks which participant is assigned to which group
type GroupAssignment struct {
	ID            uint64
	TournamentID  uint64
	StageID       uint64
	GroupID       uint64
	ParticipantID uint64
	SeedInGroup   *int // Seed within the group (1-based)
	CreatedAt     time.Time
}

// StageRankingCriteria represents a ranking criterion with its priority for a stage
type StageRankingCriteria struct {
	ID        uint64
	StageID   uint64
	Criterion RankingCriterion
	Priority  int // Lower = higher priority (1 is primary, 2 is secondary, etc.)
}

// PlacementMatchConfig configures placement matches for tiebreaking
type PlacementMatchConfig struct {
	ID      uint64
	StageID uint64
	Enabled bool
	Depth   int // How many positions to resolve via matches
}

// StageSeedAssignment represents a manual seed assignment before a stage starts
// These are used to influence serpentine distribution when StartStage is called
type StageSeedAssignment struct {
	ID               uint64
	TournamentID     uint64
	StageID          uint64
	ParticipantID    uint64
	TargetGroupOrder int // 0 = Group A, 1 = Group B, etc.
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// StageParticipantPool tracks which stage each participant starts in for multi-stage tournaments.
// Participants assigned to later stages (e.g., Finals) wait idle until their stage becomes active.
// This allows top seeds to skip early stages and start directly in later rounds.
type StageParticipantPool struct {
	ID            uint64
	TournamentID  uint64
	StageID       uint64
	ParticipantID uint64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// MultiStageConfig represents the complete configuration for a multi-stage tournament
type MultiStageConfig struct {
	Stages          []TournamentStage
	Groups          map[uint64][]StageGroup          // stageID -> groups
	Assignments     map[uint64][]GroupAssignment     // groupID -> assignments
	RankingCriteria map[uint64][]StageRankingCriteria // stageID -> criteria
	PlacementConfig map[uint64]*PlacementMatchConfig  // stageID -> config
}

// GroupNameFromOrder returns a group name like "Group A", "Group B" from an order index
func GroupNameFromOrder(order int) string {
	if order < 0 || order > 25 {
		return "Group ?"
	}
	return "Group " + string(rune('A'+order))
}

// ValidRankingCriteria returns all valid ranking criteria
func ValidRankingCriteria() []RankingCriterion {
	return []RankingCriterion{
		CriterionMatchWins,
		CriterionSetWins,
		CriterionSetWinPct,
		CriterionSetDifferential,
		CriterionPointsScored,
		CriterionPointsDifferential,
		CriterionSeed,
	}
}

// IsValidRankingCriterion checks if a criterion is valid
func IsValidRankingCriterion(c RankingCriterion) bool {
	for _, valid := range ValidRankingCriteria() {
		if c == valid {
			return true
		}
	}
	return false
}
