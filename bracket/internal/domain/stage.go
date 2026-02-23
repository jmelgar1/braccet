package domain

import (
	"fmt"
	"time"
)

type BracketStage struct {
	ID           uint64
	TournamentID uint64
	BracketType  BracketType
	Round        int
	StageName    *string
	BestOf       int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DefaultStageName returns the default name for a round based on its position relative to the final
func DefaultStageName(round, totalRounds int) string {
	if round == totalRounds {
		return "Final"
	}
	if round == totalRounds-1 {
		return "Semifinals"
	}
	if round == totalRounds-2 {
		return "Quarterfinals"
	}
	return fmt.Sprintf("Round %d", round)
}

// GetDisplayName returns the stage name to display (custom name if set, otherwise default)
func (s *BracketStage) GetDisplayName(totalRounds int) string {
	if s.StageName != nil && *s.StageName != "" {
		return *s.StageName
	}
	return DefaultStageName(s.Round, totalRounds)
}
