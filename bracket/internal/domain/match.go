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
	Participant1Score *int
	Participant2Score *int
	Status           MatchStatus
	ScheduledAt      *time.Time
	CompletedAt      *time.Time
	NextMatchID      *uint64
	LoserMatchID     *uint64
	ForfeitWinnerID  *uint64 // Non-nil if match was won by forfeit (opponent withdrew)
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type MatchResult struct {
	WinnerID          uint64
	Participant1Score int
	Participant2Score int
}
