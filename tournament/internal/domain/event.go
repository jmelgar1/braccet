package domain

import "time"

// EventStatus represents the status of a multi-tournament event
type EventStatus string

const (
	EventStatusDraft        EventStatus = "draft"
	EventStatusRegistration EventStatus = "registration"
	EventStatusInProgress   EventStatus = "in_progress"
	EventStatusCompleted    EventStatus = "completed"
	EventStatusCancelled    EventStatus = "cancelled"
)

// EventTournamentRole represents the role of a tournament within an event
type EventTournamentRole string

const (
	EventRoleQualifier EventTournamentRole = "qualifier"
	EventRoleMain      EventTournamentRole = "main"
)

// Event represents a multi-tournament event (e.g., qualifiers + main event)
// Events are community-bound (CommunityID is required)
type Event struct {
	ID          uint64
	Slug        string
	CommunityID uint64 // Required - events are always community-bound
	OrganizerID uint64
	Name        string
	Description *string
	Game        *string
	Status      EventStatus
	StartsAt    *time.Time
	LogoURL     *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EventTournament links a tournament to an event with progression rules
type EventTournament struct {
	ID                 uint64
	EventID            uint64
	TournamentID       uint64
	Role               EventTournamentRole
	DisplayOrder       int
	AdvancingCount     *int    // How many participants advance (NULL for main event)
	TargetTournamentID *uint64 // Where winners go (NULL for main event)
	PositionX          int     // X coordinate in visual DAG editor
	PositionY          int     // Y coordinate in visual DAG editor
	CreatedAt          time.Time
}

// EventAdvancement tracks lineage of advanced participants across tournaments
type EventAdvancement struct {
	ID                  uint64
	EventID             uint64
	SourceTournamentID  uint64
	SourceParticipantID uint64
	TargetTournamentID  uint64
	TargetParticipantID uint64
	FinalPlacement      int // 1 = winner, 2 = runner-up, etc.
	AdvancedAt          time.Time
}
