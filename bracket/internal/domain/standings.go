package domain

// Standing represents a participant's standing in any tournament format.
// This is the unified type used for display purposes across Swiss, single,
// and double elimination formats.
type Standing struct {
	Rank            int
	ParticipantID   uint64
	ParticipantName string
	IconURL         *string
	Seed            int

	// Match record
	Wins   int
	Losses int

	// Game/set record (for point differential tiebreaker)
	GameWins   int
	GameLosses int

	// Format-specific fields
	Placement    string      // "Champion", "2nd", "3rd-4th", etc. (elimination)
	OpponentWins int         // Buchholz tiebreaker (Swiss)
	HasBye       bool        // Had a bye round (Swiss)
	FinalRound   int         // Last round played (elimination)
	BracketType  BracketType // Which bracket eliminated from (double elim)
}

// GameDifferential returns game wins minus game losses
func (s *Standing) GameDifferential() int {
	return s.GameWins - s.GameLosses
}

// StandingsCalculator provides game statistics calculation from matches
type StandingsCalculator interface {
	// CalculateGameStats calculates game wins/losses for each participant from match sets
	CalculateGameStats(matches []*Match) map[uint64]*GameStats
}

// GameStats holds calculated game statistics for a participant
type GameStats struct {
	ParticipantID uint64
	GameWins      int
	GameLosses    int
}
