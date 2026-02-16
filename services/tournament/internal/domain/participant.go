package domain

import "time"

type ParticipantStatus string

const (
	ParticipantRegistered   ParticipantStatus = "registered"
	ParticipantCheckedIn    ParticipantStatus = "checked_in"
	ParticipantActive       ParticipantStatus = "active"
	ParticipantEliminated   ParticipantStatus = "eliminated"
	ParticipantDisqualified ParticipantStatus = "disqualified"
)

type Participant struct {
	ID           uint64
	TournamentID uint64
	UserID       uint64
	DisplayName  string
	Seed         *uint
	Status       ParticipantStatus
	CheckedInAt  *time.Time
	CreatedAt    time.Time
}
