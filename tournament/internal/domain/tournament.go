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
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
