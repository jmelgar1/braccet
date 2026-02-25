package service

import (
	"context"
	"fmt"
	"sort"

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
	matchRepo repository.MatchRepository
	setRepo   repository.SetRepository
	groupRepo repository.GroupRepository
	swissRepo repository.SwissRepository
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
