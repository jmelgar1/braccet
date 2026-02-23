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

// DefaultStageName returns the default name for a round based on bracket type and position relative to the final
func DefaultStageName(bracketType BracketType, round, totalRounds int) string {
	switch bracketType {
	case BracketGrandFinal:
		return "Grand Final"

	case BracketLosers:
		if round == totalRounds {
			return "Losers Final"
		}
		if round == totalRounds-1 {
			return "Losers Semis"
		}
		return fmt.Sprintf("Losers Round %d", round)

	case BracketWinners:
		// Check if this is double elimination (has losers bracket context)
		// For single elimination, totalRounds represents the whole bracket
		// For double elimination winners, use "Winners" prefix
		if round == totalRounds {
			return "Winners Final"
		}
		if round == totalRounds-1 {
			return "Winners Semis"
		}
		if round == totalRounds-2 {
			return "Winners Quarters"
		}
		return fmt.Sprintf("Winners Round %d", round)

	default:
		// Single elimination or unknown type - use simple names
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
}

// GetDisplayName returns the stage name to display (custom name if set, otherwise default)
func (s *BracketStage) GetDisplayName(totalRounds int) string {
	if s.StageName != nil && *s.StageName != "" {
		return *s.StageName
	}
	return DefaultStageName(s.BracketType, s.Round, totalRounds)
}
