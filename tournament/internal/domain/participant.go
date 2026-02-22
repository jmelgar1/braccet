package domain

import "time"

type ParticipantStatus string

const (
	ParticipantRegistered   ParticipantStatus = "registered"
	ParticipantCheckedIn    ParticipantStatus = "checked_in"
	ParticipantActive       ParticipantStatus = "active"
	ParticipantEliminated   ParticipantStatus = "eliminated"
	ParticipantDisqualified ParticipantStatus = "disqualified"
	ParticipantWithdrawn    ParticipantStatus = "withdrawn"
)

type Participant struct {
	ID                uint64
	TournamentID      uint64
	UserID            *uint64 // Pointer to allow nil for display-name-only participants
	CommunityMemberID *uint64 // Links to community member for reuse across tournaments
	DisplayName       string
	Seed              *uint
	Status            ParticipantStatus
	CheckedInAt       *time.Time
	CreatedAt         time.Time
}
