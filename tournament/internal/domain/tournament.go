package domain

import (
	"encoding/json"
	"time"
)

type TournamentFormat string

const (
	FormatSingleElimination TournamentFormat = "single_elimination"
	FormatDoubleElimination TournamentFormat = "double_elimination"
)

type TournamentStatus string

const (
	StatusRegistration TournamentStatus = "registration"
	StatusInProgress   TournamentStatus = "in_progress"
	StatusCompleted    TournamentStatus = "completed"
	StatusCancelled    TournamentStatus = "cancelled"
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
	RegistrationOpen bool
	Settings         json.RawMessage
	StartsAt         *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
