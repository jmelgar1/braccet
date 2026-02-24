package domain

import "time"

type EloSystem struct {
	ID          uint64
	CommunityID uint64
	Name        string
	Description *string

	// Core ELO configuration
	StartingRating int
	KFactor        int
	FloorRating    int

	// Provisional period
	ProvisionalGames   int
	ProvisionalKFactor int

	// Win streak
	WinStreakEnabled   bool
	WinStreakThreshold int
	WinStreakBonus     int

	// Decay
	DecayEnabled bool
	DecayDays    int
	DecayAmount  int
	DecayFloor   int

	IsDefault bool
	IsActive  bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// MatchResult represents a single match result: 'W' for win, 'L' for loss, 'D' for draw
type MatchResult string

const (
	MatchResultWin  MatchResult = "W"
	MatchResultLoss MatchResult = "L"
	MatchResultDraw MatchResult = "D"
)

type MemberEloRating struct {
	ID               uint64
	MemberID         uint64
	EloSystemID      uint64
	Rating           int
	GamesPlayed      int
	GamesWon         int
	CurrentWinStreak int
	HighestRating    int
	LowestRating     int
	LastGameAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Joined fields for display
	MemberDisplayName *string
	MemberRegion      *string
	MemberIconURL     *string

	// Recent match results (last 5), most recent first
	RecentResults []MatchResult
}

type EloChangeType string

const (
	EloChangeMatch      EloChangeType = "match"
	EloChangeDecay      EloChangeType = "decay"
	EloChangeAdjustment EloChangeType = "adjustment"
	EloChangeInitial    EloChangeType = "initial"
)

type EloHistory struct {
	ID          uint64
	MemberID    uint64
	EloSystemID uint64

	ChangeType   EloChangeType
	RatingBefore int
	RatingChange int
	RatingAfter  int

	// Match context
	MatchID              *uint64
	TournamentID         *uint64
	OpponentMemberID     *uint64
	OpponentRatingBefore *int
	IsWinner             *bool

	// Calculation details
	KFactorUsed    *int
	ExpectedScore  *float64
	WinStreakBonus int

	Notes     *string
	CreatedAt time.Time

	// Joined fields for display
	OpponentDisplayName *string
}
