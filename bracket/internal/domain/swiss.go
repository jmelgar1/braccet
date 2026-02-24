package domain

import "time"

// SwissStanding tracks a participant's cumulative results in a Swiss tournament
type SwissStanding struct {
	ID                 uint64
	TournamentID       uint64
	ParticipantID      uint64
	ParticipantName    string
	ParticipantIconURL *string
	Seed               int
	Wins               int
	Losses             int
	GameWins           int // Total games/sets won (tiebreaker)
	GameLosses         int // Total games/sets lost
	OpponentWins       int // Buchholz tiebreaker: sum of opponents' wins
	HasBye             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SwissConfig stores tournament-level Swiss configuration
type SwissConfig struct {
	TournamentID uint64
	TotalRounds  int
	CurrentRound int
	IsComplete   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SwissPairingHistory records which participants have played each other
type SwissPairingHistory struct {
	ID             uint64
	TournamentID   uint64
	Participant1ID uint64
	Participant2ID uint64
	Round          int
	CreatedAt      time.Time
}

// SwissBracketState represents the full state of a Swiss bracket
type SwissBracketState struct {
	TournamentID uint64
	Config       *SwissConfig
	Standings    []*SwissStanding
	Matches      []*Match // All matches across all rounds
	Stages       []*BracketStage
}

// Pairing represents two participants matched for a Swiss round
type Pairing struct {
	Participant1ID      uint64
	Participant1Name    string
	Participant1IconURL *string
	Participant1Seed    int
	Participant2ID      uint64
	Participant2Name    string
	Participant2IconURL *string
	Participant2Seed    int
	IsBye               bool // true if Participant2 is a BYE
}
