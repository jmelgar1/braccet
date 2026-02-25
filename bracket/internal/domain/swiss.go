package domain

import "time"

// SwissParticipantStatus represents the status of a participant in threshold-based Swiss
type SwissParticipantStatus string

const (
	SwissStatusActive     SwissParticipantStatus = "active"     // Still playing
	SwissStatusAdvanced   SwissParticipantStatus = "advanced"   // Reached wins_to_advance threshold
	SwissStatusEliminated SwissParticipantStatus = "eliminated" // Reached losses_to_eliminate threshold
)

// SwissStanding tracks a participant's cumulative results in a Swiss tournament
type SwissStanding struct {
	ID                 uint64
	TournamentID       uint64
	StageID            uint64
	GroupID            uint64
	ParticipantID      uint64
	ParticipantName    string
	ParticipantIconURL *string
	Seed               int
	Wins               int
	Losses             int
	MatchWins          int                    // Real wins (excluding BYEs) - used for advancement threshold
	GameWins           int                    // Total games/sets won (tiebreaker)
	GameLosses         int                    // Total games/sets lost
	OpponentWins       int                    // Buchholz tiebreaker: sum of opponents' wins
	HasBye             bool
	Status             SwissParticipantStatus // active/advanced/eliminated for threshold mode
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SwissConfig stores tournament-level Swiss configuration
type SwissConfig struct {
	TournamentID        uint64
	StageID             uint64
	GroupID             uint64
	TotalRounds         int  // 0 when using threshold mode (dynamic rounds)
	CurrentRound        int
	IsComplete          bool
	HasTiebreakers      bool // True if tiebreaker matches are needed
	TiebreakersComplete bool // True when all tiebreakers resolved
	WinsToAdvance       *int // Number of real wins (excluding BYEs) to advance
	LossesToEliminate   *int // Number of losses to be eliminated
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// IsThresholdMode returns true if the Swiss bracket uses advancement/elimination thresholds
func (c *SwissConfig) IsThresholdMode() bool {
	return c.WinsToAdvance != nil || c.LossesToEliminate != nil
}

// SwissPairingHistory records which participants have played each other
type SwissPairingHistory struct {
	ID             uint64
	TournamentID   uint64
	StageID        uint64
	GroupID        uint64
	Participant1ID uint64
	Participant2ID uint64
	Round          int
	CreatedAt      time.Time
}

// SwissBracketState represents the full state of a Swiss bracket
type SwissBracketState struct {
	TournamentID uint64
	Config       *SwissConfig
	Standings    []*SwissStanding
	Matches      []*Match // All matches across all rounds
	Stages       []*BracketStage
	Tiebreakers  []*TiebreakerBracket // Tiebreaker brackets if any
}

// Pairing represents two participants matched for a Swiss round
type Pairing struct {
	Participant1ID      uint64
	Participant1Name    string
	Participant1IconURL *string
	Participant1Seed    int
	Participant2ID      uint64
	Participant2Name    string
	Participant2IconURL *string
	Participant2Seed    int
	IsBye               bool // true if Participant2 is a BYE
}
