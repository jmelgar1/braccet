package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/braccet/bracket/internal/client"
	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/engine"
	"github.com/braccet/bracket/internal/repository"
)

// MultiStageService handles multi-stage tournament operations
type MultiStageService interface {
	// CompleteStage finalizes a stage by calculating standings and determining who advances.
	// Returns the stage standings with advancing participants marked.
	CompleteStage(ctx context.Context, req CompleteStageRequest) (*CompleteStageResponse, error)

	// GetAdvancingParticipants returns participants who advanced from a completed stage.
	GetAdvancingParticipants(ctx context.Context, stageID uint64) ([]AdvancingParticipant, error)

	// ReseedFinalBracket deletes the final bracket and regenerates it from advancing participants.
	ReseedFinalBracket(ctx context.Context, req ReseedFinalBracketRequest) error

	// ReseedStage reseeds ALL groups in a stage with proper cross-group distribution.
	// This deletes all matches, redistributes participants across groups using serpentine
	// seeding, and regenerates brackets for each group.
	ReseedStage(ctx context.Context, req ReseedStageRequest) error
}

// ReseedFinalBracketRequest contains the parameters for reseeding a final bracket
type ReseedFinalBracketRequest struct {
	TournamentID uint64
	StageID      uint64 // The final stage ID
	Format       string // Format to regenerate (single_elimination, double_elimination)
}

// ReseedStageRequest contains the parameters for reseeding all groups in a stage
type ReseedStageRequest struct {
	TournamentID uint64
	StageID      uint64
	Format       string // Format for all groups (single_elimination, double_elimination, swiss)
}

// CompleteStageRequest contains the parameters for completing a stage
type CompleteStageRequest struct {
	TournamentID      uint64
	StageID           uint64
	GroupIDs          []uint64                   // Groups in this stage (from tournament service)
	AdvancingPerGroup int                        // How many participants advance from each group
	RankingCriteria   []domain.RankingCriterion  // Criteria for ranking (empty = use bracket placement)
	Format            string                     // Stage format (single_elimination, double_elimination, swiss)
	IsThresholdSwiss  bool                       // True if this is a threshold-based Swiss stage
}

// CompleteStageResponse contains the results of completing a stage
type CompleteStageResponse struct {
	StageStandings       []*domain.StageStanding `json:"stage_standings"`
	AdvancingParticipants []AdvancingParticipant  `json:"advancing_participants"`
}

// AdvancingParticipant represents a participant who advances to the next stage
type AdvancingParticipant struct {
	ParticipantID   uint64  `json:"participant_id"`
	ParticipantName string  `json:"participant_name"`
	IconURL         *string `json:"icon_url,omitempty"`
	StageRank       int     `json:"stage_rank"`
	GroupID         uint64  `json:"group_id"`
	GroupRank       int     `json:"group_rank"`
}

type multiStageService struct {
	matchRepo        repository.MatchRepository
	setRepo          repository.SetRepository
	groupRepo        repository.GroupRepository
	swissRepo        repository.SwissRepository
	stageRepo        repository.StageRepository
	bracketSvc       BracketService
	tournamentClient client.TournamentClient
}

// NewMultiStageService creates a new multi-stage service
func NewMultiStageService(
	matchRepo repository.MatchRepository,
	setRepo repository.SetRepository,
	groupRepo repository.GroupRepository,
) MultiStageService {
	return &multiStageService{
		matchRepo: matchRepo,
		setRepo:   setRepo,
		groupRepo: groupRepo,
	}
}

// NewMultiStageServiceWithSwiss creates a multi-stage service with Swiss support
func NewMultiStageServiceWithSwiss(
	matchRepo repository.MatchRepository,
	setRepo repository.SetRepository,
	groupRepo repository.GroupRepository,
	swissRepo repository.SwissRepository,
) MultiStageService {
	return &multiStageService{
		matchRepo: matchRepo,
		setRepo:   setRepo,
		groupRepo: groupRepo,
		swissRepo: swissRepo,
	}
}

// NewMultiStageServiceFull creates a multi-stage service with all capabilities including reseed
func NewMultiStageServiceFull(
	matchRepo repository.MatchRepository,
	setRepo repository.SetRepository,
	groupRepo repository.GroupRepository,
	stageRepo repository.StageRepository,
	bracketSvc BracketService,
	swissRepo repository.SwissRepository,
	tournamentClient client.TournamentClient,
) MultiStageService {
	return &multiStageService{
		matchRepo:        matchRepo,
		setRepo:          setRepo,
		groupRepo:        groupRepo,
		stageRepo:        stageRepo,
		bracketSvc:       bracketSvc,
		swissRepo:        swissRepo,
		tournamentClient: tournamentClient,
	}
}

// CompleteStage finalizes a stage by calculating standings and determining who advances
func (s *multiStageService) CompleteStage(ctx context.Context, req CompleteStageRequest) (*CompleteStageResponse, error) {
	// For threshold-based Swiss, use Swiss standings directly
	if req.IsThresholdSwiss && s.swissRepo != nil && len(req.GroupIDs) == 1 {
		return s.completeThresholdSwissStage(ctx, req)
	}

	if len(req.GroupIDs) == 0 {
		return nil, fmt.Errorf("no groups provided")
	}

	// Collect group standings from all groups
	groupStandingsMap := make(map[uint64][]*domain.GroupStanding)

	for _, groupID := range req.GroupIDs {
		// Get matches for this group
		matches, err := s.groupRepo.GetMatchesByGroup(ctx, req.TournamentID, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to get matches for group %d: %w", groupID, err)
		}

		// Load sets for matches
		if len(matches) > 0 {
			matchIDs := make([]uint64, len(matches))
			for i, m := range matches {
				matchIDs[i] = m.ID
			}
			setsMap, err := s.setRepo.GetByMatchIDs(ctx, matchIDs)
			if err != nil {
				return nil, fmt.Errorf("failed to get sets for group %d: %w", groupID, err)
			}
			for _, m := range matches {
				if sets, ok := setsMap[m.ID]; ok {
					m.Sets = sets
				}
			}
		}

		// Get existing standings or create initial ones
		standings, err := s.groupRepo.GetGroupStandings(ctx, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to get standings for group %d: %w", groupID, err)
		}

		// If no standings exist, create them from match participants
		if len(standings) == 0 {
			standings = s.createStandingsFromMatches(req.TournamentID, req.StageID, groupID, matches)
		}

		// Calculate final standings
		standings = engine.CalculateGroupStandingsWithFormat(standings, matches, req.RankingCriteria, req.Format)

		// Update standings in database
		for _, standing := range standings {
			if standing.ID > 0 {
				if err := s.groupRepo.UpdateGroupStanding(ctx, standing); err != nil {
					return nil, fmt.Errorf("failed to update standing: %w", err)
				}
			} else {
				// Create new standings
				if err := s.groupRepo.CreateGroupStandings(ctx, []*domain.GroupStanding{standing}); err != nil {
					return nil, fmt.Errorf("failed to create standing: %w", err)
				}
			}
		}

		groupStandingsMap[groupID] = standings
	}

	// Default criteria for cross-group comparison
	criteria := req.RankingCriteria
	if len(criteria) == 0 {
		criteria = []domain.RankingCriterion{
			domain.CriterionMatchWins,
			domain.CriterionSetDifferential,
			domain.CriterionPointsDifferential,
		}
	}

	// Aggregate into stage standings
	stageStandings := engine.AggregateStageStandings(groupStandingsMap, req.AdvancingPerGroup, criteria)

	// Save stage standings
	if err := s.groupRepo.CreateStageStandings(ctx, stageStandings); err != nil {
		return nil, fmt.Errorf("failed to create stage standings: %w", err)
	}

	// Build list of advancing participants
	var advancing []AdvancingParticipant
	for _, ss := range stageStandings {
		if ss.Advances {
			// Find the participant name from group standings
			var name string
			var iconURL *string
			for _, gs := range groupStandingsMap[ss.GroupID] {
				if gs.ParticipantID == ss.ParticipantID {
					name = gs.ParticipantName
					iconURL = gs.ParticipantIconURL
					break
				}
			}

			stageRank := 0
			if ss.StageRank != nil {
				stageRank = *ss.StageRank
			}

			advancing = append(advancing, AdvancingParticipant{
				ParticipantID:   ss.ParticipantID,
				ParticipantName: name,
				IconURL:         iconURL,
				StageRank:       stageRank,
				GroupID:         ss.GroupID,
				GroupRank:       ss.GroupRank,
			})
		}
	}

	// Sort advancing by stage rank
	sort.Slice(advancing, func(i, j int) bool {
		return advancing[i].StageRank < advancing[j].StageRank
	})

	return &CompleteStageResponse{
		StageStandings:        stageStandings,
		AdvancingParticipants: advancing,
	}, nil
}

// completeThresholdSwissStage handles stage completion for threshold-based Swiss stages.
// In threshold mode, advancing participants are those with status='advanced'.
func (s *multiStageService) completeThresholdSwissStage(ctx context.Context, req CompleteStageRequest) (*CompleteStageResponse, error) {
	groupID := req.GroupIDs[0]

	// Get advanced standings from Swiss repository
	advancedStandings, err := s.swissRepo.GetAdvancedStandings(ctx, req.TournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get advanced standings: %w", err)
	}

	// Get all standings for stage standings
	allStandings, err := s.swissRepo.GetStandings(ctx, req.TournamentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all standings: %w", err)
	}

	// Build stage standings from Swiss standings
	var stageStandings []*domain.StageStanding
	for rank, ss := range allStandings {
		stageRank := rank + 1
		advances := ss.Status == domain.SwissStatusAdvanced
		stageStandings = append(stageStandings, &domain.StageStanding{
			TournamentID:  ss.TournamentID,
			StageID:       req.StageID,
			ParticipantID: ss.ParticipantID,
			GroupID:       groupID,
			GroupRank:     rank + 1,
			StageRank:     &stageRank,
			MatchWins:     ss.Wins,
			MatchLosses:   ss.Losses,
			SetWins:       ss.GameWins,
			SetLosses:     ss.GameLosses,
			Advances:      advances,
		})
	}

	// Save stage standings
	if err := s.groupRepo.CreateStageStandings(ctx, stageStandings); err != nil {
		return nil, fmt.Errorf("failed to create stage standings: %w", err)
	}

	// Build list of advancing participants (those with status='advanced')
	var advancing []AdvancingParticipant
	for rank, ss := range advancedStandings {
		advancing = append(advancing, AdvancingParticipant{
			ParticipantID:   ss.ParticipantID,
			ParticipantName: ss.ParticipantName,
			IconURL:         ss.ParticipantIconURL,
			StageRank:       rank + 1,
			GroupID:         groupID,
			GroupRank:       rank + 1,
		})
	}

	return &CompleteStageResponse{
		StageStandings:        stageStandings,
		AdvancingParticipants: advancing,
	}, nil
}

// createStandingsFromMatches creates initial standings from match participants
func (s *multiStageService) createStandingsFromMatches(
	tournamentID, stageID, groupID uint64,
	matches []*domain.Match,
) []*domain.GroupStanding {
	// Collect unique participants
	participantMap := make(map[uint64]*domain.GroupStanding)
	seed := 1

	for _, m := range matches {
		if m.Participant1ID != nil {
			if _, exists := participantMap[*m.Participant1ID]; !exists {
				name := ""
				if m.Participant1Name != nil {
					name = *m.Participant1Name
				}
				participantMap[*m.Participant1ID] = &domain.GroupStanding{
					TournamentID:       tournamentID,
					StageID:            stageID,
					GroupID:            groupID,
					ParticipantID:      *m.Participant1ID,
					ParticipantName:    name,
					ParticipantIconURL: m.Participant1IconURL,
					Seed:               seed,
				}
				seed++
			}
		}
		if m.Participant2ID != nil {
			if _, exists := participantMap[*m.Participant2ID]; !exists {
				name := ""
				if m.Participant2Name != nil {
					name = *m.Participant2Name
				}
				participantMap[*m.Participant2ID] = &domain.GroupStanding{
					TournamentID:       tournamentID,
					StageID:            stageID,
					GroupID:            groupID,
					ParticipantID:      *m.Participant2ID,
					ParticipantName:    name,
					ParticipantIconURL: m.Participant2IconURL,
					Seed:               seed,
				}
				seed++
			}
		}
	}

	standings := make([]*domain.GroupStanding, 0, len(participantMap))
	for _, s := range participantMap {
		standings = append(standings, s)
	}

	return standings
}

// GetAdvancingParticipants returns participants who advanced from a completed stage
func (s *multiStageService) GetAdvancingParticipants(ctx context.Context, stageID uint64) ([]AdvancingParticipant, error) {
	// Get advancing standings from repository
	standings, err := s.groupRepo.GetAdvancingParticipants(ctx, stageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get advancing participants: %w", err)
	}

	// Convert to AdvancingParticipant
	var advancing []AdvancingParticipant
	for _, ss := range standings {
		stageRank := 0
		if ss.StageRank != nil {
			stageRank = *ss.StageRank
		}

		// We need participant name - get it from group standings
		gs, err := s.groupRepo.GetGroupStandings(ctx, ss.GroupID)
		if err != nil {
			continue
		}

		var name string
		var iconURL *string
		for _, g := range gs {
			if g.ParticipantID == ss.ParticipantID {
				name = g.ParticipantName
				iconURL = g.ParticipantIconURL
				break
			}
		}

		advancing = append(advancing, AdvancingParticipant{
			ParticipantID:   ss.ParticipantID,
			ParticipantName: name,
			IconURL:         iconURL,
			StageRank:       stageRank,
			GroupID:         ss.GroupID,
			GroupRank:       ss.GroupRank,
		})
	}

	// Sort by stage rank
	sort.Slice(advancing, func(i, j int) bool {
		return advancing[i].StageRank < advancing[j].StageRank
	})

	return advancing, nil
}

// ReseedFinalBracket deletes the final bracket and regenerates it from advancing participants.
// This is used to regenerate the final bracket with fresh pairings.
func (s *multiStageService) ReseedFinalBracket(ctx context.Context, req ReseedFinalBracketRequest) error {
	if s.stageRepo == nil || s.bracketSvc == nil {
		return fmt.Errorf("reseed requires stageRepo and bracketSvc")
	}

	// Finals use groupID = 0
	groupID := uint64(0)

	var participants []domain.Participant

	// For Swiss tournaments, get fresh rankings from Swiss standings.
	// This ensures reseed uses the same ordering as initial bracket generation
	// (GetAdvancedStandings orders by: losses ASC, game_diff DESC, opponent_wins DESC, seed ASC)
	if s.swissRepo != nil {
		advancedStandings, err := s.swissRepo.GetAdvancedStandings(ctx, req.TournamentID)
		if err == nil && len(advancedStandings) >= 2 {
			// Use Swiss standings directly (same as initial bracket generation)
			participants = make([]domain.Participant, len(advancedStandings))
			for i, ss := range advancedStandings {
				iconURL := ""
				if ss.ParticipantIconURL != nil {
					iconURL = *ss.ParticipantIconURL
				}
				participants[i] = domain.Participant{
					ID:      ss.ParticipantID,
					Name:    ss.ParticipantName,
					IconURL: iconURL,
					Seed:    i + 1, // Rank from sorted Swiss standings
				}
			}
		}
	}

	// Fallback to stage_standings for non-Swiss formats or if Swiss query failed/returned insufficient data
	if len(participants) < 2 {
		standings, err := s.groupRepo.GetAdvancingParticipantsByTournament(ctx, req.TournamentID)
		if err != nil {
			return fmt.Errorf("failed to get advancing participants: %w", err)
		}

		if len(standings) < 2 {
			return fmt.Errorf("not enough participants to generate bracket (need at least 2, have %d)", len(standings))
		}

		// Convert stage standings to domain.Participant for bracket generation
		// We need to get participant names from group standings
		participants = make([]domain.Participant, len(standings))
		for i, ss := range standings {
			// Get participant name from group standings
			gs, err := s.groupRepo.GetGroupStandings(ctx, ss.GroupID)
			if err != nil {
				return fmt.Errorf("failed to get group standings: %w", err)
			}

			var name string
			var iconURL string
			for _, g := range gs {
				if g.ParticipantID == ss.ParticipantID {
					name = g.ParticipantName
					if g.ParticipantIconURL != nil {
						iconURL = *g.ParticipantIconURL
					}
					break
				}
			}

			stageRank := i + 1 // Default rank
			if ss.StageRank != nil {
				stageRank = *ss.StageRank
			}

			participants[i] = domain.Participant{
				ID:      ss.ParticipantID,
				Name:    name,
				IconURL: iconURL,
				Seed:    stageRank, // Use stage rank as seed
			}
		}
	}

	// Delete existing matches for this stage
	if err := s.matchRepo.DeleteByTournamentStageGroup(ctx, req.TournamentID, req.StageID, groupID); err != nil {
		return fmt.Errorf("failed to delete matches: %w", err)
	}

	// Delete existing bracket stages for this stage
	if err := s.stageRepo.DeleteByTournamentStageGroup(ctx, req.TournamentID, req.StageID, groupID); err != nil {
		return fmt.Errorf("failed to delete bracket stages: %w", err)
	}

	// Regenerate the bracket using the bracket service
	params := engine.GroupBracketParams{
		TournamentID: req.TournamentID,
		StageID:      req.StageID,
		GroupID:      groupID,
		Format:       req.Format,
		Participants: participants,
	}

	_, err := s.bracketSvc.GenerateGroupBracket(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to regenerate bracket: %w", err)
	}

	return nil
}

// ReseedStage reseeds ALL groups in a stage with proper cross-group distribution.
// This:
// 1. Gets all groups and participants from tournament service
// 2. Determines the correct seed order (from previous stage rankings or tournament seed)
// 3. Calculates new group assignments using serpentine seeding
// 4. Updates assignments in tournament service
// 5. Deletes all matches/standings/bracket_stages for the stage
// 6. Regenerates brackets for each group
func (s *multiStageService) ReseedStage(ctx context.Context, req ReseedStageRequest) error {
	if s.tournamentClient == nil || s.bracketSvc == nil {
		return fmt.Errorf("reseed stage requires tournamentClient and bracketSvc")
	}

	// 1. Get all groups for this stage from tournament service
	groups, err := s.tournamentClient.GetStageGroups(ctx, req.StageID)
	if err != nil {
		return fmt.Errorf("failed to get groups: %w", err)
	}

	if len(groups) == 0 {
		return fmt.Errorf("no groups found for stage %d", req.StageID)
	}

	// 2. Get all participants from tournament service
	participants, err := s.tournamentClient.GetStageParticipants(ctx, req.StageID)
	if err != nil {
		return fmt.Errorf("failed to get participants: %w", err)
	}

	if len(participants) < 2 {
		return fmt.Errorf("not enough participants to reseed (need at least 2, have %d)", len(participants))
	}

	// 3. Determine the correct seed order
	// Participants NEW to this stage (underdogs) get lower seeds (1-N)
	// Participants ADVANCING from previous stage (favorites) get higher seeds (N+1 to total)
	// This ensures advancing participants are visually on bottom (favorites) in matches
	advancingStandings, _ := s.groupRepo.GetAdvancingParticipantsByTournament(ctx, req.TournamentID)

	// Build a map of participantID -> stageRank from previous stage
	// Filter to only include standings from stages OTHER than the current one
	rankMap := make(map[uint64]int)
	for _, st := range advancingStandings {
		// Only use standings from a different stage (previous stage)
		if st.StageID != req.StageID && st.StageRank != nil {
			rankMap[st.ParticipantID] = *st.StageRank
		}
	}

	// Separate participants into two groups: new (no previous rank) and advancing (has previous rank)
	var newParticipants, advancingParticipants []client.StageParticipantResponse
	for _, p := range participants {
		if _, hasRank := rankMap[p.ParticipantID]; hasRank {
			advancingParticipants = append(advancingParticipants, p)
		} else {
			newParticipants = append(newParticipants, p)
		}
	}

	// Sort new participants by GlobalSeed (tournament seed)
	sort.SliceStable(newParticipants, func(i, j int) bool {
		return newParticipants[i].GlobalSeed < newParticipants[j].GlobalSeed
	})

	// Sort advancing participants by their previous stage rank
	sort.SliceStable(advancingParticipants, func(i, j int) bool {
		return rankMap[advancingParticipants[i].ParticipantID] < rankMap[advancingParticipants[j].ParticipantID]
	})

	// Combine: new participants first (seeds 1-N), then advancing (seeds N+1 to total)
	participants = append(newParticipants, advancingParticipants...)

	// 3. Calculate serpentine group distribution
	// Serpentine pattern: Row 0 (R→L), Row 1 (L→R), Row 2 (R→L), ...
	// For 8 participants, 2 groups: Group B gets 1,4,5,8 and Group A gets 2,3,6,7
	// This puts the top seed in the "later" group (B), balancing competitive strength
	numGroups := len(groups)

	// Build group order -> group ID mapping
	groupOrderToID := make(map[int]uint64)
	for _, g := range groups {
		groupOrderToID[g.GroupOrder] = g.ID
	}

	// Apply serpentine distribution
	var assignments []client.AssignmentUpdateRequest
	groupCounts := make(map[int]int) // track seed_in_group for each group

	for i, p := range participants {
		row := i / numGroups
		posInRow := i % numGroups

		var groupOrder int
		if row%2 == 0 {
			// Even rows: right to left (N-1, N-2, ..., 0)
			// Seed 1 goes to Group B (order N-1), Seed 2 goes to Group A (order 0)
			groupOrder = numGroups - 1 - posInRow
		} else {
			// Odd rows: left to right (0, 1, 2, ...)
			// Seed 3 goes to Group A (order 0), Seed 4 goes to Group B (order N-1)
			groupOrder = posInRow
		}

		groupID := groupOrderToID[groupOrder]
		groupCounts[groupOrder]++
		seedInGroup := groupCounts[groupOrder]

		assignments = append(assignments, client.AssignmentUpdateRequest{
			GroupID:       groupID,
			ParticipantID: p.ParticipantID,
			SeedInGroup:   seedInGroup,
		})
	}

	// 4. Update assignments in tournament service
	if err := s.tournamentClient.UpdateStageAssignments(ctx, req.StageID, assignments); err != nil {
		return fmt.Errorf("failed to update assignments: %w", err)
	}

	// 5. Delete all matches, standings, and bracket stages for this stage
	for _, group := range groups {
		// Delete matches
		if err := s.matchRepo.DeleteByTournamentStageGroup(ctx, req.TournamentID, req.StageID, group.ID); err != nil {
			return fmt.Errorf("failed to delete matches for group %d: %w", group.ID, err)
		}

		// Delete group standings
		if s.groupRepo != nil {
			if err := s.groupRepo.DeleteGroupStandingsByGroup(ctx, group.ID); err != nil {
				return fmt.Errorf("failed to delete standings for group %d: %w", group.ID, err)
			}
		}

		// Delete bracket stages
		if s.stageRepo != nil {
			if err := s.stageRepo.DeleteByTournamentStageGroup(ctx, req.TournamentID, req.StageID, group.ID); err != nil {
				return fmt.Errorf("failed to delete bracket stages for group %d: %w", group.ID, err)
			}
		}

		// Delete Swiss data if applicable (pairings and reset standings)
		if s.swissRepo != nil {
			// Delete all pairings from round 1 onwards (effectively deletes all)
			_ = s.swissRepo.DeletePairingsByRoundRangeForGroup(ctx, req.TournamentID, req.StageID, group.ID, 1)
			// Reset standings to initial state
			_ = s.swissRepo.ResetStandingsForGroup(ctx, req.TournamentID, req.StageID, group.ID)
		}
	}

	// 6. Regenerate brackets for each group
	// Build participant lookup by ID
	participantLookup := make(map[uint64]client.StageParticipantResponse)
	for _, p := range participants {
		participantLookup[p.ParticipantID] = p
	}

	// Group assignments by group ID
	assignmentsByGroup := make(map[uint64][]client.AssignmentUpdateRequest)
	for _, a := range assignments {
		assignmentsByGroup[a.GroupID] = append(assignmentsByGroup[a.GroupID], a)
	}

	// Generate bracket for each group
	for _, group := range groups {
		groupAssignments := assignmentsByGroup[group.ID]
		if len(groupAssignments) == 0 {
			continue
		}

		// Sort by seed_in_group
		sort.Slice(groupAssignments, func(i, j int) bool {
			return groupAssignments[i].SeedInGroup < groupAssignments[j].SeedInGroup
		})

		// Convert to domain.Participant
		groupParticipants := make([]domain.Participant, len(groupAssignments))
		for i, a := range groupAssignments {
			p := participantLookup[a.ParticipantID]
			iconURL := ""
			if p.IconURL != nil {
				iconURL = *p.IconURL
			}
			groupParticipants[i] = domain.Participant{
				ID:      a.ParticipantID,
				Name:    p.Name,
				IconURL: iconURL,
				Seed:    a.SeedInGroup,
			}
		}

		// Generate the bracket
		params := engine.GroupBracketParams{
			TournamentID: req.TournamentID,
			StageID:      req.StageID,
			GroupID:      group.ID,
			Format:       req.Format,
			Participants: groupParticipants,
		}

		_, err := s.bracketSvc.GenerateGroupBracket(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to generate bracket for group %d: %w", group.ID, err)
		}
	}

	return nil
}
