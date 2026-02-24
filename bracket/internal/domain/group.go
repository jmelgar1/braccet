package domain

import "time"

// GroupStanding represents a participant's stats within a single group
type GroupStanding struct {
	ID                 uint64
	TournamentID       uint64
	StageID            uint64
	GroupID            uint64
	ParticipantID      uint64
	ParticipantName    string
	ParticipantIconURL *string
	Seed               int

	// Match stats
	MatchWins   int
	MatchLosses int

	// Set/game stats
	SetWins   int
	SetLosses int

	// Point stats (aggregate of all set scores)
	PointsScored  int
	PointsAgainst int

	// Tiebreaker stats
	OpponentMatchWins int // Buchholz-style tiebreaker: sum of opponents' wins

	// Final ranking within group (computed after group completes)
	GroupRank *int

	CreatedAt time.Time
	UpdatedAt time.Time
}

// StageStanding represents a participant's overall standing in a stage (cross-group)
type StageStanding struct {
	ID            uint64
	TournamentID  uint64
	StageID       uint64
	ParticipantID uint64
	GroupID       uint64
	GroupRank     int // Rank within their group

	// Aggregated stats for cross-group comparison
	MatchWins   int
	MatchLosses int
	SetWins     int
	SetLosses   int
	PointsScored  int
	PointsAgainst int

	// Final stage rank (computed after all groups complete)
	StageRank *int
	Advances  bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

// GroupBracketState represents the state of a group bracket
type GroupBracketState struct {
	TournamentID uint64
	StageID      uint64
	GroupID      uint64
	GroupName    string
	Format       string // single_elimination, double_elimination, swiss
	TotalRounds  int
	CurrentRound int
	IsComplete   bool
	WinnerID     *uint64
	Matches      []*Match
	Standings    []*GroupStanding
}

// SetDifferential returns set wins minus set losses
func (s *GroupStanding) SetDifferential() int {
	return s.SetWins - s.SetLosses
}

// PointDifferential returns points scored minus points against
func (s *GroupStanding) PointDifferential() int {
	return s.PointsScored - s.PointsAgainst
}

// SetWinPercentage returns set win percentage (0-100)
func (s *GroupStanding) SetWinPercentage() float64 {
	total := s.SetWins + s.SetLosses
	if total == 0 {
		return 0
	}
	return float64(s.SetWins) / float64(total) * 100
}

// RankingCriterion defines criteria for ranking participants
type RankingCriterion string

const (
	CriterionMatchWins          RankingCriterion = "match_wins"
	CriterionSetWins            RankingCriterion = "set_wins"
	CriterionSetWinPct          RankingCriterion = "set_win_pct"
	CriterionSetDifferential    RankingCriterion = "set_differential"
	CriterionPointsScored       RankingCriterion = "points_scored"
	CriterionPointsDifferential RankingCriterion = "points_differential"
)

// CompareStandings compares two standings by a given criterion
// Returns negative if a < b, 0 if equal, positive if a > b
func CompareStandings(a, b *GroupStanding, criterion RankingCriterion) int {
	switch criterion {
	case CriterionMatchWins:
		return a.MatchWins - b.MatchWins
	case CriterionSetWins:
		return a.SetWins - b.SetWins
	case CriterionSetWinPct:
		aPct := a.SetWinPercentage()
		bPct := b.SetWinPercentage()
		if aPct > bPct {
			return 1
		} else if aPct < bPct {
			return -1
		}
		return 0
	case CriterionSetDifferential:
		return a.SetDifferential() - b.SetDifferential()
	case CriterionPointsScored:
		return a.PointsScored - b.PointsScored
	case CriterionPointsDifferential:
		return a.PointDifferential() - b.PointDifferential()
	default:
		return 0
	}
}
