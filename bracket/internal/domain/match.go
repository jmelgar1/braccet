package domain

import "time"

type BracketType string

const (
	BracketWinners    BracketType = "winners"
	BracketLosers     BracketType = "losers"
	BracketGrandFinal BracketType = "grand_final"
)

type MatchStatus string

const (
	MatchPending    MatchStatus = "pending"
	MatchReady      MatchStatus = "ready"
	MatchInProgress MatchStatus = "in_progress"
	MatchCompleted  MatchStatus = "completed"
)

type Match struct {
	ID               uint64
	TournamentID     uint64
	BracketType      BracketType
	Round            int
	Position         int
	Participant1ID   *uint64
	Participant2ID   *uint64
	Participant1Name *string
	Participant2Name *string
	WinnerID         *uint64
	Sets             []Set // Set-based scoring (replaces simple scores)
	Status           MatchStatus
	ScheduledAt      *time.Time
	CompletedAt      *time.Time
	NextMatchID      *uint64
	LoserMatchID     *uint64
	ForfeitWinnerID  *uint64 // Non-nil if match was won by forfeit (opponent withdrew)
	CreatedAt        time.Time
	UpdatedAt        time.Time
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
