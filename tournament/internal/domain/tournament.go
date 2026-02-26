package domain

import (
	"encoding/json"
	"time"
)

type TournamentFormat string

const (
	FormatSingleElimination TournamentFormat = "single_elimination"
	FormatDoubleElimination TournamentFormat = "double_elimination"
	FormatSwiss             TournamentFormat = "swiss"
	FormatMultiStage        TournamentFormat = "multi_stage"
)

type TournamentStatus string

const (
	StatusRegistration TournamentStatus = "registration"
	StatusInProgress   TournamentStatus = "in_progress"
	StatusCompleted    TournamentStatus = "completed"
	StatusCancelled    TournamentStatus = "cancelled"
)

// PrizeDistributionMode indicates whether prizes are specified as percentages or fixed amounts
type PrizeDistributionMode string

const (
	PrizeDistModePercentage PrizeDistributionMode = "percentage"
	PrizeDistModeAmount     PrizeDistributionMode = "amount"
)

// PlacementTier represents a prize tier for a placement range
type PlacementTier struct {
	Placement string  `json:"placement"` // e.g., "1st", "3rd-4th"
	Low       int     `json:"low"`       // Lower placement bound (e.g., 3)
	High      int     `json:"high"`      // Upper placement bound (e.g., 4)
	Value     float64 `json:"value"`     // Percentage or dollar amount based on mode
}

// PrizeDistribution defines how the prize pool is distributed across placements
type PrizeDistribution struct {
	Mode  PrizeDistributionMode `json:"mode"`
	Tiers []PlacementTier       `json:"tiers"`
}

// TournamentSettings holds configurable tournament settings stored in JSONB
type TournamentSettings struct {
	PrizeDistribution *PrizeDistribution `json:"prize_distribution,omitempty"`
}

// TournamentClass represents the classification level for power ranking weighting
type TournamentClass string

const (
	TournamentClassMajor       TournamentClass = "major"        // Premier events (e.g., World Championship)
	TournamentClassWorldLAN    TournamentClass = "world_lan"    // World-level LAN events
	TournamentClassContinental TournamentClass = "continental"  // Regional majors (e.g., European Championship)
	TournamentClassRegional    TournamentClass = "regional"     // Local/regional events
	TournamentClassOnline      TournamentClass = "online"       // Online tournaments
)

type Tournament struct {
	ID               uint64
	Slug             string
	OrganizerID      uint64
	CommunityID      *uint64 // Optional - NULL for standalone tournaments
	EloSystemID      *uint64 // Optional - ELO system for rating updates
	PRSystemID       *uint64 // Optional - Power Ranking system for placements
	Name             string
	Description      *string
	Game             *string
	Format           TournamentFormat
	Status           TournamentStatus
	MaxParticipants  *uint
	SwissRounds      *int // Optional - number of rounds for Swiss format (defaults to ceil(log2(n)))
	RegistrationOpen bool
	Settings         json.RawMessage
	StartsAt         *time.Time
	TournamentClass  *TournamentClass // Optional - classification for power ranking weighting
	PrizePoolUSD     *float64         // Optional - prize pool in USD for power ranking calculations
	LogoURL          *string          // Optional - URL of the tournament logo
	EventID          *uint64          // Optional - populated when tournament belongs to an event
	EventRole        *string          // Optional - "qualifier" or "main" when part of an event
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
