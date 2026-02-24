package domain

import "time"

type BracketType string

const (
	BracketWinners    BracketType = "winners"
	BracketLosers     BracketType = "losers"
	BracketGrandFinal BracketType = "grand_final"
	BracketSwiss      BracketType = "swiss"
)

type MatchStatus string

const (
	MatchPending    MatchStatus = "pending"
	MatchReady      MatchStatus = "ready"
	MatchInProgress MatchStatus = "in_progress"
	MatchCompleted  MatchStatus = "completed"
)

type Match struct {
	ID                  uint64
	TournamentID        uint64
	BracketType         BracketType
	Round               int
	Position            int
	Participant1ID      *uint64
	Participant2ID      *uint64
	Participant1Name    *string
	Participant2Name    *string
	Participant1IconURL *string
	Participant2IconURL *string
	Seed1               *int
	Seed2               *int
	WinnerID            *uint64
	Sets                []Set // Set-based scoring (replaces simple scores)
	Status              MatchStatus
	ScheduledAt         *time.Time
	CompletedAt         *time.Time
	NextMatchID         *uint64
	LoserMatchID        *uint64
	ForfeitWinnerID     *uint64 // Non-nil if match was won by forfeit (opponent withdrew)
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Set represents a single set within a match
type Set struct {
	ID                uint64
	MatchID           uint64
	SetNumber         int
	Participant1Score int
	Participant2Score int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// SetScore represents set scores for API requests (without database fields)
type SetScore struct {
	SetNumber         int `json:"set_number"`
	Participant1Score int `json:"participant1_score"`
	Participant2Score int `json:"participant2_score"`
}

// MatchResult contains the sets data for reporting a match result
// Winner is computed from sets (whoever wins the most sets)
type MatchResult struct {
	Sets []SetScore
}

// EliminationStanding represents a participant's final placement in an elimination bracket
type EliminationStanding struct {
	Rank            int
	ParticipantID   uint64
	ParticipantName string
	IconURL         *string
	Seed            int
	Wins            int
	Losses          int
	GameWins        int
	GameLosses      int
	Placement       string // "Champion", "2nd", "3rd-4th", "5th-8th", etc.
	FinalRound      int    // Last round played
	BracketType     BracketType
}
