package service

import (
	"context"

	"github.com/braccet/bracket/internal/client"
	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/engine"
	"github.com/braccet/bracket/internal/repository"
)

type BracketService interface {
	GenerateSingleElimination(ctx context.Context, tournamentID uint64, participants []domain.Participant) (*BracketState, error)
	GenerateDoubleElimination(ctx context.Context, tournamentID uint64, participants []domain.Participant) (*BracketState, error)
	GenerateGroupBracket(ctx context.Context, params engine.GroupBracketParams) (*BracketState, error)
	DeleteBracket(ctx context.Context, tournamentID uint64) error
}

type bracketService struct {
	repo            repository.MatchRepository
	stageRepo       repository.StageRepository
	swissRepo       repository.SwissRepository
	groupRepo       repository.GroupRepository
	communityClient client.CommunityClient
}

func NewBracketService(repo repository.MatchRepository, stageRepo repository.StageRepository) BracketService {
	return &bracketService{repo: repo, stageRepo: stageRepo}
}

// NewBracketServiceWithCommunity creates a bracket service with community client for ELO operations
func NewBracketServiceWithCommunity(repo repository.MatchRepository, stageRepo repository.StageRepository, communityClient client.CommunityClient) BracketService {
	return &bracketService{repo: repo, stageRepo: stageRepo, communityClient: communityClient}
}

// NewBracketServiceWithSwiss creates a bracket service with Swiss support
func NewBracketServiceWithSwiss(repo repository.MatchRepository, stageRepo repository.StageRepository, swissRepo repository.SwissRepository, communityClient client.CommunityClient) BracketService {
	return &bracketService{repo: repo, stageRepo: stageRepo, swissRepo: swissRepo, communityClient: communityClient}
}

// NewBracketServiceFull creates a bracket service with all repositories
func NewBracketServiceFull(repo repository.MatchRepository, stageRepo repository.StageRepository, swissRepo repository.SwissRepository, groupRepo repository.GroupRepository, communityClient client.CommunityClient) BracketService {
	return &bracketService{repo: repo, stageRepo: stageRepo, swissRepo: swissRepo, groupRepo: groupRepo, communityClient: communityClient}
}

// GenerateSingleElimination creates a single elimination bracket and persists it.
func (s *bracketService) GenerateSingleElimination(ctx context.Context, tournamentID uint64, participants []domain.Participant) (*BracketState, error) {
	// Generate matches in memory
	matches, err := engine.SingleElimination(tournamentID, participants)
	if err != nil {
		return nil, err
	}

	// Save matches to DB (assigns IDs)
	if err := s.repo.CreateBatch(ctx, matches); err != nil {
		return nil, err
	}

	// Link matches now that we have IDs
	engine.LinkMatches(matches)

	// Persist the links
	if err := s.repo.UpdateNextMatchLinks(ctx, matches); err != nil {
		return nil, err
	}

	// Advance bye winners through the bracket
	if err := s.advanceByeWinners(ctx, matches); err != nil {
		return nil, err
	}

	// Create default stages for the bracket (single elimination uses simple names)
	state := buildBracketState(tournamentID, matches)
	if s.stageRepo != nil && state.TotalRounds > 0 {
		if err := s.stageRepo.CreateDefaultStages(ctx, tournamentID, domain.BracketWinners, state.TotalRounds, false); err != nil {
			return nil, err
		}
	}

	// Reload matches to get final state
	matches, err = s.repo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	return buildBracketState(tournamentID, matches), nil
}

// GenerateDoubleElimination creates a double elimination bracket and persists it.
func (s *bracketService) GenerateDoubleElimination(ctx context.Context, tournamentID uint64, participants []domain.Participant) (*BracketState, error) {
	// Generate all matches in memory (winners, losers, grand final)
	matches, err := engine.DoubleElimination(tournamentID, participants)
	if err != nil {
		return nil, err
	}

	// Save matches to DB (assigns IDs)
	if err := s.repo.CreateBatch(ctx, matches); err != nil {
		return nil, err
	}

	// Link matches now that we have IDs (sets NextMatchID and LoserMatchID)
	engine.LinkDoubleElimMatches(matches)

	// Persist all the links
	if err := s.repo.UpdateNextMatchLinks(ctx, matches); err != nil {
		return nil, err
	}

	// Advance bye winners through the winners bracket only
	// Note: byes don't advance to losers bracket (they never lost)
	if err := s.advanceByeWinnersDoubleElim(ctx, matches); err != nil {
		return nil, err
	}

	// Create default stages for all bracket types (double elimination uses prefixed names)
	if s.stageRepo != nil {
		bracketSize := engine.CalculateBracketSize(len(participants))
		winnersRounds := engine.TotalRounds(bracketSize)
		losersRounds := engine.LosersRounds(winnersRounds)

		// Winners bracket stages
		if winnersRounds > 0 {
			if err := s.stageRepo.CreateDefaultStages(ctx, tournamentID, domain.BracketWinners, winnersRounds, true); err != nil {
				return nil, err
			}
		}

		// Losers bracket stages
		if losersRounds > 0 {
			if err := s.stageRepo.CreateDefaultStages(ctx, tournamentID, domain.BracketLosers, losersRounds, true); err != nil {
				return nil, err
			}
		}

		// Grand final stage
		if err := s.stageRepo.CreateDefaultStages(ctx, tournamentID, domain.BracketGrandFinal, 1, true); err != nil {
			return nil, err
		}
	}

	// Reload matches to get final state
	matches, err = s.repo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	return buildDoubleElimBracketState(tournamentID, matches), nil
}

// advanceByeWinnersDoubleElim propagates winners from bye matches in double elimination.
// Important: bye winners don't go to losers bracket (they never played and lost).
func (s *bracketService) advanceByeWinnersDoubleElim(ctx context.Context, matches []*domain.Match) error {
	// Only process winners bracket for initial bye advancement
	winnersMatches := make([]*domain.Match, 0)
	for _, m := range matches {
		if m.BracketType == domain.BracketWinners {
			winnersMatches = append(winnersMatches, m)
		}
	}

	// Find all matches in a map for quick lookup
	matchMap := make(map[uint64]*domain.Match)
	for _, m := range matches {
		matchMap[m.ID] = m
	}

	// Process completed bye matches and advance their winners
	for _, match := range winnersMatches {
		if match.Status == domain.MatchCompleted && match.WinnerID != nil && match.NextMatchID != nil {
			nextMatch := matchMap[*match.NextMatchID]
			if nextMatch == nil {
				continue
			}

			// Get winner info
			winnerName, winnerIconURL, winnerSeed := getParticipantInfo(match, *match.WinnerID)

			// Determine slot based on position
			slot := 1
			if match.Position%2 == 0 {
				slot = 2
			}

			if err := s.repo.SetParticipant(ctx, nextMatch.ID, slot, *match.WinnerID, winnerName, winnerIconURL, winnerSeed); err != nil {
				return err
			}

			// Update in-memory for subsequent iterations
			if slot == 1 {
				nextMatch.Participant1ID = match.WinnerID
				nextMatch.Participant1Name = &winnerName
				if winnerIconURL != "" {
					nextMatch.Participant1IconURL = &winnerIconURL
				}
				nextMatch.Seed1 = &winnerSeed
			} else {
				nextMatch.Participant2ID = match.WinnerID
				nextMatch.Participant2Name = &winnerName
				if winnerIconURL != "" {
					nextMatch.Participant2IconURL = &winnerIconURL
				}
				nextMatch.Seed2 = &winnerSeed
			}
		}
	}

	// Check if any next-round matches are now ready
	for _, match := range winnersMatches {
		if match.Status == domain.MatchPending && match.Participant1ID != nil && match.Participant2ID != nil {
			if err := s.repo.UpdateStatus(ctx, match.ID, domain.MatchReady); err != nil {
				return err
			}
			match.Status = domain.MatchReady
		}
	}

	// Handle cascading byes
	for _, match := range winnersMatches {
		if match.Status == domain.MatchReady {
			// Check if this is a bye match
			if match.Participant1ID == nil || match.Participant2ID == nil {
				var winnerID *uint64
				if match.Participant1ID != nil {
					winnerID = match.Participant1ID
				} else {
					winnerID = match.Participant2ID
				}

				if winnerID != nil {
					if err := s.repo.UpdateResult(ctx, match.ID, *winnerID); err != nil {
						return err
					}
					match.Status = domain.MatchCompleted
					match.WinnerID = winnerID

					if match.NextMatchID != nil {
						return s.advanceByeWinnersDoubleElim(ctx, matches)
					}
				}
			}
		}
	}

	return nil
}

// getParticipantInfo extracts name, icon URL, and seed for a participant
func getParticipantInfo(match *domain.Match, participantID uint64) (name string, iconURL string, seed int) {
	if match.Participant1ID != nil && *match.Participant1ID == participantID {
		if match.Participant1Name != nil {
			name = *match.Participant1Name
		}
		if match.Participant1IconURL != nil {
			iconURL = *match.Participant1IconURL
		}
		if match.Seed1 != nil {
			seed = *match.Seed1
		}
	} else {
		if match.Participant2Name != nil {
			name = *match.Participant2Name
		}
		if match.Participant2IconURL != nil {
			iconURL = *match.Participant2IconURL
		}
		if match.Seed2 != nil {
			seed = *match.Seed2
		}
	}
	return
}

func buildDoubleElimBracketState(tournamentID uint64, matches []*domain.Match) *BracketState {
	if len(matches) == 0 {
		return &BracketState{TournamentID: tournamentID}
	}

	state := &BracketState{
		TournamentID: tournamentID,
		Matches:      matches,
	}

	// Find total rounds from winners bracket
	for _, m := range matches {
		if m.BracketType == domain.BracketWinners && m.Round > state.TotalRounds {
			state.TotalRounds = m.Round
		}
	}

	// Current round is the lowest round with non-completed winners bracket matches
	state.CurrentRound = state.TotalRounds
	for _, m := range matches {
		if m.BracketType == domain.BracketWinners && m.Status != domain.MatchCompleted && m.Round < state.CurrentRound {
			state.CurrentRound = m.Round
		}
	}

	// Check if complete (grand final has winner)
	for _, m := range matches {
		if m.BracketType == domain.BracketGrandFinal && m.WinnerID != nil {
			state.IsComplete = true
			state.ChampionID = m.WinnerID
			break
		}
	}

	return state
}

// advanceByeWinners propagates winners from bye matches to subsequent rounds.
func (s *bracketService) advanceByeWinners(ctx context.Context, matches []*domain.Match) error {
	// Process completed matches (byes) and advance their winners
	for _, match := range matches {
		if match.Status == domain.MatchCompleted && match.WinnerID != nil && match.NextMatchID != nil {
			// Find the next match
			var nextMatch *domain.Match
			for _, m := range matches {
				if m.ID == *match.NextMatchID {
					nextMatch = m
					break
				}
			}

			if nextMatch == nil {
				continue
			}

			// Determine winner's name, icon URL, and seed
			winnerName := ""
			winnerIconURL := ""
			winnerSeed := 0
			if match.Participant1ID != nil && *match.Participant1ID == *match.WinnerID {
				if match.Participant1Name != nil {
					winnerName = *match.Participant1Name
				}
				if match.Participant1IconURL != nil {
					winnerIconURL = *match.Participant1IconURL
				}
				if match.Seed1 != nil {
					winnerSeed = *match.Seed1
				}
			} else {
				if match.Participant2Name != nil {
					winnerName = *match.Participant2Name
				}
				if match.Participant2IconURL != nil {
					winnerIconURL = *match.Participant2IconURL
				}
				if match.Seed2 != nil {
					winnerSeed = *match.Seed2
				}
			}

			// Determine slot based on position
			slot := 1
			if match.Position%2 == 0 {
				slot = 2
			}

			if err := s.repo.SetParticipant(ctx, nextMatch.ID, slot, *match.WinnerID, winnerName, winnerIconURL, winnerSeed); err != nil {
				return err
			}

			// Update in-memory for subsequent iterations
			if slot == 1 {
				nextMatch.Participant1ID = match.WinnerID
				nextMatch.Participant1Name = &winnerName
				nextMatch.Participant1IconURL = &winnerIconURL
				nextMatch.Seed1 = &winnerSeed
			} else {
				nextMatch.Participant2ID = match.WinnerID
				nextMatch.Participant2Name = &winnerName
				nextMatch.Participant2IconURL = &winnerIconURL
				nextMatch.Seed2 = &winnerSeed
			}
		}
	}

	// Check if any next-round matches are now ready (both participants set)
	for _, match := range matches {
		if match.Status == domain.MatchPending && match.Participant1ID != nil && match.Participant2ID != nil {
			if err := s.repo.UpdateStatus(ctx, match.ID, domain.MatchReady); err != nil {
				return err
			}
			match.Status = domain.MatchReady
		}
	}

	// Recursively handle any matches that became completed due to both participants being byes
	// (This handles cases like only 2 participants in an 8-bracket where multiple rounds auto-complete)
	for _, match := range matches {
		if match.Status == domain.MatchReady {
			// Check if this is a bye match (one participant nil)
			if match.Participant1ID == nil || match.Participant2ID == nil {
				// Auto-complete the bye
				var winnerID *uint64
				if match.Participant1ID != nil {
					winnerID = match.Participant1ID
				} else {
					winnerID = match.Participant2ID
				}

				if winnerID != nil {
					// For bye matches, just update the winner directly (no sets needed)
					if err := s.repo.UpdateResult(ctx, match.ID, *winnerID); err != nil {
						return err
					}
					match.Status = domain.MatchCompleted
					match.WinnerID = winnerID

					// Recurse to handle this completed match
					if match.NextMatchID != nil {
						return s.advanceByeWinners(ctx, matches)
					}
				}
			}
		}
	}

	return nil
}

func buildBracketState(tournamentID uint64, matches []*domain.Match) *BracketState {
	if len(matches) == 0 {
		return &BracketState{TournamentID: tournamentID}
	}

	state := &BracketState{
		TournamentID: tournamentID,
		Matches:      matches,
	}

	for _, m := range matches {
		if m.Round > state.TotalRounds {
			state.TotalRounds = m.Round
		}
	}

	state.CurrentRound = state.TotalRounds
	for _, m := range matches {
		if m.Status != domain.MatchCompleted && m.Round < state.CurrentRound {
			state.CurrentRound = m.Round
		}
	}

	for _, m := range matches {
		if m.Round == state.TotalRounds && m.WinnerID != nil {
			state.IsComplete = true
			state.ChampionID = m.WinnerID
			break
		}
	}

	return state
}

// GenerateGroupBracket creates a bracket for a specific group within a stage.
// It generates matches tagged with the stage and group IDs for proper filtering.
func (s *bracketService) GenerateGroupBracket(ctx context.Context, params engine.GroupBracketParams) (*BracketState, error) {
	// For Swiss format with threshold options, use the Swiss service
	if params.Format == "swiss" && s.swissRepo != nil && (params.WinsToAdvance != nil || params.LossesToEliminate != nil || params.SwissRounds != nil) {
		return s.generateSwissGroupBracket(ctx, params)
	}

	// Generate matches in memory
	matches, err := engine.GenerateGroupBracket(params)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return &BracketState{TournamentID: params.TournamentID}, nil
	}

	// Save matches to DB (assigns IDs)
	if err := s.repo.CreateBatch(ctx, matches); err != nil {
		return nil, err
	}

	// Link matches now that we have IDs
	if params.Format == "double_elimination" {
		engine.LinkDoubleElimMatches(matches)
	} else {
		engine.LinkMatches(matches)
	}

	// Persist the links
	if err := s.repo.UpdateNextMatchLinks(ctx, matches); err != nil {
		return nil, err
	}

	// Advance bye winners through the bracket
	if params.Format == "double_elimination" {
		if err := s.advanceByeWinnersDoubleElim(ctx, matches); err != nil {
			return nil, err
		}
	} else if params.Format != "swiss" {
		if err := s.advanceByeWinners(ctx, matches); err != nil {
			return nil, err
		}
	}

	// Reload matches to get final state
	matches, err = s.repo.GetByTournamentStageGroup(ctx, params.TournamentID, params.StageID, params.GroupID)
	if err != nil {
		return nil, err
	}

	// Build state to determine total rounds
	state := buildBracketState(params.TournamentID, matches)

	// Create default stages for the bracket (if not already created for this tournament)
	if s.stageRepo != nil && state.TotalRounds > 0 {
		isDoubleElim := params.Format == "double_elimination"

		if isDoubleElim {
			bracketSize := engine.CalculateBracketSize(len(params.Participants))
			winnersRounds := engine.TotalRounds(bracketSize)
			losersRounds := engine.LosersRounds(winnersRounds)

			// Winners bracket stages
			if winnersRounds > 0 {
				if err := s.stageRepo.CreateDefaultStages(ctx, params.TournamentID, domain.BracketWinners, winnersRounds, true); err != nil {
					return nil, err
				}
			}
			// Losers bracket stages
			if losersRounds > 0 {
				if err := s.stageRepo.CreateDefaultStages(ctx, params.TournamentID, domain.BracketLosers, losersRounds, true); err != nil {
					return nil, err
				}
			}
			// Grand final stage (only if skip_finals is false)
			if !params.SkipFinals {
				if err := s.stageRepo.CreateDefaultStages(ctx, params.TournamentID, domain.BracketGrandFinal, 1, true); err != nil {
					return nil, err
				}
			}
		} else if params.Format == "swiss" {
			if err := s.stageRepo.CreateDefaultStages(ctx, params.TournamentID, domain.BracketSwiss, state.TotalRounds, false); err != nil {
				return nil, err
			}
		} else {
			// Single elimination
			if err := s.stageRepo.CreateDefaultStages(ctx, params.TournamentID, domain.BracketWinners, state.TotalRounds, false); err != nil {
				return nil, err
			}
		}
	}

	return state, nil
}

// generateSwissGroupBracket creates a Swiss bracket with threshold options
func (s *bracketService) generateSwissGroupBracket(ctx context.Context, params engine.GroupBracketParams) (*BracketState, error) {
	// Calculate rounds - use 0 for threshold mode (dynamic rounds)
	var totalRounds int
	if params.WinsToAdvance != nil || params.LossesToEliminate != nil {
		totalRounds = 0
	} else {
		totalRounds = engine.CalculateSwissRounds(len(params.Participants))
		if params.SwissRounds != nil && *params.SwissRounds > 0 {
			totalRounds = *params.SwissRounds
		}
	}

	// Create config
	config := &domain.SwissConfig{
		TournamentID:      params.TournamentID,
		StageID:           params.StageID,
		GroupID:           params.GroupID,
		TotalRounds:       totalRounds,
		CurrentRound:      1,
		IsComplete:        false,
		WinsToAdvance:     params.WinsToAdvance,
		LossesToEliminate: params.LossesToEliminate,
	}
	if err := s.swissRepo.CreateConfig(ctx, config); err != nil {
		return nil, err
	}

	// Create standings for each participant
	standings := make([]*domain.SwissStanding, len(params.Participants))
	for i, p := range params.Participants {
		iconURL := ""
		if p.IconURL != "" {
			iconURL = p.IconURL
		}
		standings[i] = &domain.SwissStanding{
			TournamentID:       params.TournamentID,
			StageID:            params.StageID,
			GroupID:            params.GroupID,
			ParticipantID:      p.ID,
			ParticipantName:    p.Name,
			ParticipantIconURL: &iconURL,
			Seed:               p.Seed,
			Wins:               0,
			Losses:             0,
			MatchWins:          0,
			GameWins:           0,
			GameLosses:         0,
			OpponentWins:       0,
			HasBye:             false,
			Status:             domain.SwissStatusActive,
		}
	}
	if err := s.swissRepo.CreateStandings(ctx, standings); err != nil {
		return nil, err
	}

	// Generate round 1 pairings
	pairings := engine.GenerateRound1Pairings(standings)

	// Create pairing history for non-BYE matches
	var historyRecords []*domain.SwissPairingHistory
	for _, p := range pairings {
		if !p.IsBye {
			historyRecords = append(historyRecords, &domain.SwissPairingHistory{
				TournamentID:   params.TournamentID,
				StageID:        params.StageID,
				GroupID:        params.GroupID,
				Participant1ID: p.Participant1ID,
				Participant2ID: p.Participant2ID,
				Round:          1,
			})
		}
	}
	if err := s.swissRepo.CreatePairings(ctx, historyRecords); err != nil {
		return nil, err
	}

	// Convert pairings to matches
	matches := engine.PairingsToMatches(params.TournamentID, 1, pairings)

	// Tag matches with stage and group IDs
	for _, m := range matches {
		m.StageID = &params.StageID
		m.GroupID = &params.GroupID
	}

	// Save matches
	if err := s.repo.CreateBatch(ctx, matches); err != nil {
		return nil, err
	}

	// Update standings for BYE matches (BYEs increment Wins but NOT MatchWins)
	for _, p := range pairings {
		if p.IsBye {
			standing, err := s.swissRepo.GetStandingByParticipant(ctx, params.TournamentID, p.Participant1ID)
			if err != nil {
				return nil, err
			}
			standing.Wins = 1
			standing.HasBye = true
			// Note: MatchWins stays at 0 - BYEs don't count toward advancement
			if err := s.swissRepo.UpdateStanding(ctx, standing); err != nil {
				return nil, err
			}
		}
	}

	// Create stages for this group's Swiss bracket
	if s.stageRepo != nil {
		if totalRounds > 0 {
			// Fixed rounds mode: create all stages upfront
			if err := s.stageRepo.CreateDefaultStagesForGroup(ctx, params.TournamentID, params.StageID, params.GroupID, domain.BracketSwiss, totalRounds); err != nil {
				return nil, err
			}
		} else {
			// Threshold mode: create only Round 1 stage (others created dynamically in AdvanceGroupRound)
			stage := &domain.BracketStage{
				TournamentID: params.TournamentID,
				StageID:      &params.StageID,
				GroupID:      &params.GroupID,
				BracketType:  domain.BracketSwiss,
				Round:        1,
				BestOf:       1,
			}
			stageName := domain.DefaultSwissStageName(1, 1)
			stage.StageName = &stageName
			if err := s.stageRepo.UpsertForGroup(ctx, stage); err != nil {
				return nil, err
			}
		}
	}

	// Build state
	state := &BracketState{
		TournamentID: params.TournamentID,
		TotalRounds:  totalRounds,
		CurrentRound: 1,
		IsComplete:   false,
	}

	return state, nil
}

// DeleteBracket deletes all bracket data for a tournament and reverts ELO changes.
// This is used when resetting a tournament back to registration phase.
func (s *bracketService) DeleteBracket(ctx context.Context, tournamentID uint64) error {
	// First, revert ELO changes if community client is configured
	if s.communityClient != nil {
		if err := s.communityClient.RevertTournamentElo(ctx, tournamentID); err != nil {
			// Log but continue - ELO might not be configured for this tournament
			// The community service will handle "no history" gracefully
		}
	}

	// Delete all matches (match_sets are deleted via ON DELETE CASCADE)
	if err := s.repo.DeleteByTournament(ctx, tournamentID); err != nil {
		return err
	}

	// Delete all bracket stages
	if s.stageRepo != nil {
		if err := s.stageRepo.DeleteByTournament(ctx, tournamentID); err != nil {
			return err
		}
	}

	// Delete Swiss data if Swiss repo is configured
	if s.swissRepo != nil {
		if err := s.swissRepo.DeleteByTournament(ctx, tournamentID); err != nil {
			return err
		}
	}

	// Delete group standings and stage standings (for multi-stage tournaments)
	if s.groupRepo != nil {
		if err := s.groupRepo.DeleteGroupStandingsByTournament(ctx, tournamentID); err != nil {
			return err
		}
		if err := s.groupRepo.DeleteStageStandingsByTournament(ctx, tournamentID); err != nil {
			return err
		}
	}

	return nil
}
