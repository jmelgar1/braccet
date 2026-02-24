package domain

import "time"

// TournamentClass represents the classification level for power ranking weighting
type TournamentClass string

const (
	TournamentClassMajor       TournamentClass = "major"
	TournamentClassWorldLAN    TournamentClass = "world_lan"
	TournamentClassContinental TournamentClass = "continental"
	TournamentClassRegional    TournamentClass = "regional"
	TournamentClassOnline      TournamentClass = "online"
)

// PowerRankingSystem represents a configurable power ranking system for a community
type PowerRankingSystem struct {
	ID          uint64
	CommunityID uint64
	Name        string
	Description *string

	// Component weights (must sum to 100)
	AchievementsWeight int
	FormWeight         int
	LANWeight          int

	// Achievements configuration
	AchievementDecayMonths  int // Months until ~50% decay
	AchievementWindowMonths int // Rolling window

	// Form configuration
	FormWindowMonths int

	// LAN configuration
	LANResultsCount int

	IsDefault bool
	IsActive  bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// TournamentPlacement represents a participant's placement in a completed tournament
type TournamentPlacement struct {
	ID           uint64
	MemberID     uint64
	PRSystemID   uint64
	TournamentID uint64

	Placement        int
	ParticipantCount int

	// Tournament metadata (denormalized for calculation efficiency)
	TournamentClass   TournamentClass
	PrizePoolUSD      *float64
	IsBo3Playoffs     bool
	AvgOpponentRating *int
	IsLAN             bool

	RawPoints   int
	CompletedAt time.Time
	CreatedAt   time.Time
}

// MemberPowerRanking represents a member's current power ranking
type MemberPowerRanking struct {
	ID         uint64
	MemberID   uint64
	PRSystemID uint64

	TotalPoints float64
	Rank        *int

	// Component scores
	AchievementsScore float64
	FormScore         float64
	LANScore          float64

	// Form metrics (cached for display)
	FormWins              int
	FormSetWins           int
	FormSetLosses         int
	FormAvgOpponentRating *int
	FormBestWinRating     *int

	// LAN metrics
	LANPlacementsCount int

	LastCalculatedAt *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time

	// Joined fields for display
	MemberDisplayName *string
	MemberRegion      *string
	MemberIconURL     *string
}

// PowerRankingMatchCache stores match results for form calculation
type PowerRankingMatchCache struct {
	ID           uint64
	MemberID     uint64
	PRSystemID   uint64
	MatchID      uint64
	TournamentID uint64

	IsWinner         bool
	SetsWon          int
	SetsLost         int
	OpponentMemberID *uint64
	OpponentRating   *int
	IsLAN            bool

	CompletedAt time.Time
	CreatedAt   time.Time
}

// CalculationResult holds the output of a power ranking calculation
type CalculationResult struct {
	TotalPoints       float64
	AchievementsScore float64
	FormScore         float64
	LANScore          float64
	FormMetrics       FormMetrics
	LANPlacementsCount int
}

// FormMetrics contains the detailed form metrics
type FormMetrics struct {
	Wins              int
	SetWins           int
	SetLosses         int
	AvgOpponentRating *int
	BestWinRating     *int
	Score             float64
}

// ProcessPRMatchRequest is the request for processing a match result
type ProcessPRMatchRequest struct {
	PRSystemID      uint64
	MatchID         uint64
	TournamentID    uint64
	WinnerMemberID  uint64
	LoserMemberID   uint64
	WinnerSetsWon   int
	LoserSetsWon    int
	WinnerRating    int // Current ELO
	LoserRating     int
	IsLAN           bool
	CompletedAt     time.Time
}

// ProcessPlacementRequest is the request for recording a tournament placement
type ProcessPlacementRequest struct {
	PRSystemID        uint64
	TournamentID      uint64
	MemberID          uint64
	Placement         int
	ParticipantCount  int
	TournamentClass   TournamentClass
	PrizePoolUSD      *float64
	IsBo3Playoffs     bool
	AvgOpponentRating *int
	IsLAN             bool
	CompletedAt       time.Time
}
