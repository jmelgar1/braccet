package domain

import "time"

// TiebreakerBracket represents a mini bracket to resolve tied standings
type TiebreakerBracket struct {
	ID             uint64
	TournamentID   uint64
	TiePosition    int      // The rank position being resolved (e.g., 3 = tied for 3rd)
	ParticipantIDs []uint64 // Participants in this tiebreaker
	IsComplete     bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TiebreakerResult represents a participant's resolved position from a tiebreaker
type TiebreakerResult struct {
	ID            uint64
	TiebreakerID  uint64
	ParticipantID uint64
	FinalPosition int // Resolved rank (absolute position, e.g., 3 if they placed 3rd)
	CreatedAt     time.Time
}

// TieGroup represents a group of participants who are tied on all ranking criteria
type TieGroup struct {
	Position       int      // The starting rank position for this tie
	ParticipantIDs []uint64 // Participants in the tie
}
