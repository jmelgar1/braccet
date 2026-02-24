package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/braccet/tournament/internal/api/middleware"
	"github.com/braccet/tournament/internal/client"
	"github.com/braccet/tournament/internal/domain"
	"github.com/braccet/tournament/internal/repository"
	"github.com/go-chi/chi/v5"
)

// StageHandler handles multi-stage tournament operations
type StageHandler struct {
	tournamentRepo  repository.TournamentRepository
	participantRepo repository.ParticipantRepository
	stageRepo       repository.StageRepository
	bracketClient   client.BracketClient
}

func NewStageHandler(
	tournamentRepo repository.TournamentRepository,
	participantRepo repository.ParticipantRepository,
	stageRepo repository.StageRepository,
	bracketClient client.BracketClient,
) *StageHandler {
	return &StageHandler{
		tournamentRepo:  tournamentRepo,
		participantRepo: participantRepo,
		stageRepo:       stageRepo,
		bracketClient:   bracketClient,
	}
}

// Request/Response types

type StageConfigRequest struct {
	StageOrder           int      `json:"stage_order"`
	Format               string   `json:"format"`
	ParticipantsPerGroup *int     `json:"participants_per_group,omitempty"`
	AdvancingPerGroup    *int     `json:"advancing_per_group,omitempty"`
	SwissRounds          *int     `json:"swiss_rounds,omitempty"`
	SkipFinals           *bool    `json:"skip_finals,omitempty"`
	RankingCriteria      []string `json:"ranking_criteria,omitempty"`
	PlacementMatches     *bool    `json:"placement_matches,omitempty"`
	PlacementDepth       *int     `json:"placement_depth,omitempty"`
}

type CreateMultiStageTournamentRequest struct {
	Name            string               `json:"name"`
	Description     *string              `json:"description,omitempty"`
	Game            *string              `json:"game,omitempty"`
	MaxParticipants *uint                `json:"max_participants,omitempty"`
	StartsAt        *string              `json:"starts_at,omitempty"`
	CommunityID     *uint64              `json:"community_id,omitempty"`
	EloSystemID     *uint64              `json:"elo_system_id,omitempty"`
	GroupStages     []StageConfigRequest `json:"group_stages"`
	FinalStage      *StageConfigRequest  `json:"final_stage,omitempty"`
}

type StageResponse struct {
	ID                   uint64   `json:"id"`
	TournamentID         uint64   `json:"tournament_id"`
	StageOrder           int      `json:"stage_order"`
	StageType            string   `json:"stage_type"`
	Format               string   `json:"format"`
	ParticipantsPerGroup *int     `json:"participants_per_group,omitempty"`
	AdvancingPerGroup    *int     `json:"advancing_per_group,omitempty"`
	SwissRounds          *int     `json:"swiss_rounds,omitempty"`
	SkipFinals           bool     `json:"skip_finals"`
	IsActive             bool     `json:"is_active"`
	IsComplete           bool     `json:"is_complete"`
	RankingCriteria      []string `json:"ranking_criteria,omitempty"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
}

type GroupResponse struct {
	ID             uint64   `json:"id"`
	StageID        uint64   `json:"stage_id"`
	GroupName      string   `json:"group_name"`
	GroupOrder     int      `json:"group_order"`
	ParticipantIDs []uint64 `json:"participant_ids,omitempty"`
}

type UpdateStageRequest struct {
	Format               *string  `json:"format,omitempty"`
	ParticipantsPerGroup *int     `json:"participants_per_group,omitempty"`
	AdvancingPerGroup    *int     `json:"advancing_per_group,omitempty"`
	SwissRounds          *int     `json:"swiss_rounds,omitempty"`
	SkipFinals           *bool    `json:"skip_finals,omitempty"`
	RankingCriteria      []string `json:"ranking_criteria,omitempty"`
}

type MultiStageTournamentResponse struct {
	TournamentResponse
	Stages       []StageResponse `json:"stages,omitempty"`
	CurrentStage *StageResponse  `json:"current_stage,omitempty"`
}

// Seed assignment types
type StageSeedAssignmentResponse struct {
	ID               uint64 `json:"id"`
	StageID          uint64 `json:"stage_id"`
	ParticipantID    uint64 `json:"participant_id"`
	TargetGroupOrder int    `json:"target_group_order"`
	CreatedAt        string `json:"created_at"`
}

type SeedInput struct {
	ParticipantID    uint64 `json:"participant_id"`
	TargetGroupOrder int    `json:"target_group_order"`
}

type UpdateStageSeedsRequest struct {
	Seeds []SeedInput `json:"seeds"`
}

// Stage participant pool types (for assigning participants to starting stages)
type StagePoolConfigInput struct {
	StageID uint64 `json:"stage_id"`
	Count   int    `json:"count"`
}

type UpdateStagePoolRequest struct {
	Config []StagePoolConfigInput `json:"config"`
}

type StagePoolEntryResponse struct {
	ParticipantID uint64 `json:"participant_id"`
	StageID       uint64 `json:"stage_id"`
	StageOrder    int    `json:"stage_order"`
}

type StagePoolResponse struct {
	Entries []StagePoolEntryResponse `json:"entries"`
}

func toStageResponse(s *domain.TournamentStage, criteria []*domain.StageRankingCriteria) StageResponse {
	resp := StageResponse{
		ID:                   s.ID,
		TournamentID:         s.TournamentID,
		StageOrder:           s.StageOrder,
		StageType:            string(s.StageType),
		Format:               string(s.Format),
		ParticipantsPerGroup: s.ParticipantsPerGroup,
		AdvancingPerGroup:    s.AdvancingPerGroup,
		SwissRounds:          s.SwissRounds,
		SkipFinals:           s.SkipFinals,
		IsActive:             s.IsActive,
		IsComplete:           s.IsComplete,
		CreatedAt:            s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            s.UpdatedAt.Format(time.RFC3339),
	}

	if len(criteria) > 0 {
		resp.RankingCriteria = make([]string, len(criteria))
		for i, c := range criteria {
			resp.RankingCriteria[i] = string(c.Criterion)
		}
	}

	return resp
}

func toGroupResponse(g *domain.StageGroup, assignments []*domain.GroupAssignment) GroupResponse {
	resp := GroupResponse{
		ID:         g.ID,
		StageID:    g.StageID,
		GroupName:  g.GroupName,
		GroupOrder: g.GroupOrder,
	}

	if len(assignments) > 0 {
		resp.ParticipantIDs = make([]uint64, len(assignments))
		for i, a := range assignments {
			resp.ParticipantIDs[i] = a.ParticipantID
		}
	}

	return resp
}

// CreateMultiStage creates a new multi-stage tournament
func (h *StageHandler) CreateMultiStage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateMultiStageTournamentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if len(req.GroupStages) == 0 {
		writeError(w, http.StatusBadRequest, "at least one group stage is required")
		return
	}

	if len(req.GroupStages) > 3 {
		writeError(w, http.StatusBadRequest, "maximum of 3 group stages allowed")
		return
	}

	// Validate each group stage
	for i, stage := range req.GroupStages {
		if err := validateStageConfig(stage, true); err != nil {
			writeError(w, http.StatusBadRequest, "group stage "+strconv.Itoa(i+1)+": "+err.Error())
			return
		}
	}

	// Validate final stage if provided
	if req.FinalStage != nil {
		if err := validateStageConfig(*req.FinalStage, false); err != nil {
			writeError(w, http.StatusBadRequest, "final stage: "+err.Error())
			return
		}
	}

	// Create the tournament
	tournament := &domain.Tournament{
		Slug:             generateSlug(),
		OrganizerID:      userID,
		CommunityID:      req.CommunityID,
		EloSystemID:      req.EloSystemID,
		Name:             req.Name,
		Description:      req.Description,
		Game:             req.Game,
		Format:           domain.FormatMultiStage,
		Status:           domain.StatusRegistration,
		MaxParticipants:  req.MaxParticipants,
		RegistrationOpen: true,
		Settings:         json.RawMessage(`{}`),
	}

	if req.StartsAt != nil {
		startsAt, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starts_at format (use RFC3339)")
			return
		}
		tournament.StartsAt = &startsAt
	}

	if err := h.tournamentRepo.Create(r.Context(), tournament); err != nil {
		log.Printf("Error creating multi-stage tournament: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create tournament")
		return
	}

	// Create stages
	var stages []*domain.TournamentStage

	// Create group stages (order 1, 2, 3)
	for i, stageReq := range req.GroupStages {
		skipFinals := false
		if stageReq.SkipFinals != nil {
			skipFinals = *stageReq.SkipFinals
		}
		stage := &domain.TournamentStage{
			TournamentID:         tournament.ID,
			StageOrder:           i + 1,
			StageType:            domain.StageTypeGroup,
			Format:               domain.TournamentFormat(stageReq.Format),
			ParticipantsPerGroup: stageReq.ParticipantsPerGroup,
			AdvancingPerGroup:    stageReq.AdvancingPerGroup,
			SwissRounds:          stageReq.SwissRounds,
			VenueType:            domain.VenueTypeOnline, // Default to online
			SkipFinals:           skipFinals,
			IsActive:             i == 0, // First stage is active by default
			IsComplete:           false,
		}

		if err := h.stageRepo.CreateStage(r.Context(), stage); err != nil {
			log.Printf("Error creating stage %d for tournament %d: %v", i+1, tournament.ID, err)
			writeError(w, http.StatusInternalServerError, "failed to create stage")
			return
		}

		// Create ranking criteria
		if len(stageReq.RankingCriteria) > 0 {
			criteria := make([]*domain.StageRankingCriteria, len(stageReq.RankingCriteria))
			for j, c := range stageReq.RankingCriteria {
				criteria[j] = &domain.StageRankingCriteria{
					StageID:   stage.ID,
					Criterion: domain.RankingCriterion(c),
					Priority:  j + 1,
				}
			}
			if err := h.stageRepo.CreateRankingCriteria(r.Context(), criteria); err != nil {
				log.Printf("Error creating ranking criteria for stage %d: %v", stage.ID, err)
				writeError(w, http.StatusInternalServerError, "failed to create ranking criteria")
				return
			}
		}

		// Create placement config if enabled
		if stageReq.PlacementMatches != nil && *stageReq.PlacementMatches {
			depth := 1
			if stageReq.PlacementDepth != nil {
				depth = *stageReq.PlacementDepth
			}
			config := &domain.PlacementMatchConfig{
				StageID: stage.ID,
				Enabled: true,
				Depth:   depth,
			}
			if err := h.stageRepo.CreatePlacementConfig(r.Context(), config); err != nil {
				log.Printf("Error creating placement config for stage %d: %v", stage.ID, err)
				// Non-fatal, continue
			}
		}

		stages = append(stages, stage)
	}

	// Create final stage (order 0)
	if req.FinalStage != nil {
		finalSkipFinals := false
		if req.FinalStage.SkipFinals != nil {
			finalSkipFinals = *req.FinalStage.SkipFinals
		}
		finalStage := &domain.TournamentStage{
			TournamentID: tournament.ID,
			StageOrder:   0, // Final stage uses order 0
			StageType:    domain.StageTypeFinal,
			Format:       domain.TournamentFormat(req.FinalStage.Format),
			SwissRounds:  req.FinalStage.SwissRounds,
			VenueType:    domain.VenueTypeOnline, // Default to online
			SkipFinals:   finalSkipFinals,
			IsActive:     false,
			IsComplete:   false,
		}

		if err := h.stageRepo.CreateStage(r.Context(), finalStage); err != nil {
			log.Printf("Error creating final stage for tournament %d: %v", tournament.ID, err)
			writeError(w, http.StatusInternalServerError, "failed to create final stage")
			return
		}

		stages = append(stages, finalStage)
	}

	// Fetch the created tournament to get timestamps
	created, err := h.tournamentRepo.GetBySlug(r.Context(), tournament.Slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created tournament")
		return
	}

	// Build response
	stageResponses := make([]StageResponse, len(stages))
	var currentStage *StageResponse
	for i, s := range stages {
		criteria, _ := h.stageRepo.GetRankingCriteria(r.Context(), s.ID)
		stageResponses[i] = toStageResponse(s, criteria)
		if s.IsActive {
			currentStage = &stageResponses[i]
		}
	}

	response := MultiStageTournamentResponse{
		TournamentResponse: toTournamentResponse(created),
		Stages:             stageResponses,
		CurrentStage:       currentStage,
	}

	writeJSON(w, http.StatusCreated, response)
}

// GetStages returns all stages for a tournament
func (h *StageHandler) GetStages(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	if tournament.Format != domain.FormatMultiStage {
		writeError(w, http.StatusBadRequest, "tournament is not a multi-stage tournament")
		return
	}

	stages, err := h.stageRepo.GetStagesByTournament(r.Context(), tournament.ID)
	if err != nil {
		log.Printf("Error fetching stages for tournament %d: %v", tournament.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch stages")
		return
	}

	stageResponses := make([]StageResponse, len(stages))
	for i, s := range stages {
		criteria, _ := h.stageRepo.GetRankingCriteria(r.Context(), s.ID)
		stageResponses[i] = toStageResponse(s, criteria)
	}

	writeJSON(w, http.StatusOK, stageResponses)
}

// GetGroups returns all groups for a stage
func (h *StageHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	stageIDStr := chi.URLParam(r, "stageId")

	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	stageID, err := strconv.ParseUint(stageIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage ID")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	// Verify stage belongs to this tournament
	stage, err := h.stageRepo.GetStageByID(r.Context(), stageID)
	if err != nil {
		if errors.Is(err, repository.ErrStageNotFound) {
			writeError(w, http.StatusNotFound, "stage not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch stage")
		return
	}

	if stage.TournamentID != tournament.ID {
		writeError(w, http.StatusNotFound, "stage not found")
		return
	}

	groups, err := h.stageRepo.GetGroupsByStage(r.Context(), stageID)
	if err != nil {
		log.Printf("Error fetching groups for stage %d: %v", stageID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch groups")
		return
	}

	// Fetch assignments for each group
	groupResponses := make([]GroupResponse, len(groups))
	for i, g := range groups {
		assignments, _ := h.stageRepo.GetAssignmentsByGroup(r.Context(), g.ID)
		groupResponses[i] = toGroupResponse(g, assignments)
	}

	writeJSON(w, http.StatusOK, groupResponses)
}

// StartStage starts a stage by creating groups and assigning participants
func (h *StageHandler) StartStage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can start stages")
		return
	}

	if tournament.Format != domain.FormatMultiStage {
		writeError(w, http.StatusBadRequest, "tournament is not a multi-stage tournament")
		return
	}

	// Get active stage
	activeStage, err := h.stageRepo.GetActiveStage(r.Context(), tournament.ID)
	if err != nil {
		log.Printf("Error fetching active stage for tournament %d: %v", tournament.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch active stage")
		return
	}

	if activeStage == nil {
		writeError(w, http.StatusBadRequest, "no active stage found")
		return
	}

	// Check if groups already exist for this stage
	existingGroups, err := h.stageRepo.GetGroupsByStage(r.Context(), activeStage.ID)
	if err != nil {
		log.Printf("Error checking existing groups: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to check existing groups")
		return
	}

	if len(existingGroups) > 0 {
		writeError(w, http.StatusBadRequest, "stage already started - groups exist")
		return
	}

	// Get participants for this stage
	// First check if there's a stage pool configured
	poolEntries, err := h.stageRepo.GetParticipantPoolByStage(r.Context(), activeStage.ID)
	if err != nil {
		log.Printf("Error fetching participant pool: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch participant pool")
		return
	}

	var participants []*domain.Participant
	if len(poolEntries) == 0 {
		// No pool configured - use all participants (backward compatibility)
		participants, err = h.participantRepo.GetByTournament(r.Context(), tournament.ID)
		if err != nil {
			log.Printf("Error fetching participants: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to fetch participants")
			return
		}
	} else {
		// Pool configured - only use participants assigned to this stage
		participantIDs := make([]uint64, len(poolEntries))
		for i, e := range poolEntries {
			participantIDs[i] = e.ParticipantID
		}
		participants, err = h.participantRepo.GetByIDs(r.Context(), participantIDs)
		if err != nil {
			log.Printf("Error fetching participants by IDs: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to fetch participants")
			return
		}
	}

	if len(participants) < 2 {
		writeError(w, http.StatusBadRequest, "at least 2 participants required to start")
		return
	}

	// Validate participant count for stage format
	if err := validateParticipantCountForStage(activeStage, len(participants)); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Calculate number of groups
	participantsPerGroup := 4 // default
	if activeStage.ParticipantsPerGroup != nil {
		participantsPerGroup = *activeStage.ParticipantsPerGroup
	}

	numGroups := (len(participants) + participantsPerGroup - 1) / participantsPerGroup

	// Create groups
	groups := make([]*domain.StageGroup, numGroups)
	for i := 0; i < numGroups; i++ {
		groups[i] = &domain.StageGroup{
			TournamentID: tournament.ID,
			StageID:      activeStage.ID,
			GroupName:    domain.GroupNameFromOrder(i),
			GroupOrder:   i,
		}
	}

	if err := h.stageRepo.CreateGroups(r.Context(), groups); err != nil {
		log.Printf("Error creating groups: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create groups")
		return
	}

	// Get manual seed assignments
	manualSeeds, err := h.stageRepo.GetSeedAssignmentsByStage(r.Context(), activeStage.ID)
	if err != nil {
		log.Printf("Error fetching manual seeds: %v", err)
		// Non-fatal, continue without manual seeds
		manualSeeds = nil
	}

	// Apply serpentine seeding to assign participants to groups, respecting manual seeds
	assignments := serpentineAssignmentWithSeeds(participants, groups, activeStage.ID, tournament.ID, manualSeeds)

	if err := h.stageRepo.CreateAssignments(r.Context(), assignments); err != nil {
		log.Printf("Error creating assignments: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create assignments")
		return
	}

	// Create brackets for each group
	// Build a map of group assignments for easy lookup
	groupAssignmentsMap := make(map[uint64][]*domain.GroupAssignment)
	for _, a := range assignments {
		groupAssignmentsMap[a.GroupID] = append(groupAssignmentsMap[a.GroupID], a)
	}

	// Create a map of participant ID to participant for lookup
	participantMap := make(map[uint64]*domain.Participant)
	for _, p := range participants {
		participantMap[p.ID] = p
	}

	for _, group := range groups {
		groupAssignments := groupAssignmentsMap[group.ID]
		if len(groupAssignments) < 2 {
			continue // Skip groups with fewer than 2 participants
		}

		// Build participant list for this group
		groupParticipants := make([]client.Participant, 0, len(groupAssignments))
		for _, a := range groupAssignments {
			p := participantMap[a.ParticipantID]
			if p != nil {
				seed := 0
				if a.SeedInGroup != nil {
					seed = *a.SeedInGroup
				}
				groupParticipants = append(groupParticipants, client.Participant{
					ID:   p.ID,
					Name: p.DisplayName,
					Seed: seed,
					// IconURL is fetched from member data if needed
				})
			}
		}

		// Call bracket service to create the bracket
		bracketReq := client.CreateGroupBracketRequest{
			TournamentID: tournament.ID,
			StageID:      activeStage.ID,
			GroupID:      group.ID,
			Format:       string(activeStage.Format),
			Participants: groupParticipants,
			SwissRounds:  activeStage.SwissRounds,
			SkipFinals:   activeStage.SkipFinals,
		}

		if err := h.bracketClient.CreateGroupBracket(r.Context(), bracketReq); err != nil {
			log.Printf("Error creating bracket for group %d: %v", group.ID, err)
			writeError(w, http.StatusInternalServerError, "failed to create bracket for group "+group.GroupName)
			return
		}
	}

	// Update tournament status to in_progress if in registration
	if tournament.Status == domain.StatusRegistration {
		tournament.Status = domain.StatusInProgress
		tournament.RegistrationOpen = false
		if err := h.tournamentRepo.Update(r.Context(), tournament); err != nil {
			log.Printf("Error updating tournament status: %v", err)
			// Non-fatal, continue
		}
	}

	// Fetch and return groups
	groupResponses := make([]GroupResponse, len(groups))
	for i, g := range groups {
		groupAssignments, _ := h.stageRepo.GetAssignmentsByGroup(r.Context(), g.ID)
		groupResponses[i] = toGroupResponse(g, groupAssignments)
	}

	writeJSON(w, http.StatusOK, groupResponses)
}

// AdvanceStage advances the tournament to the next stage
func (h *StageHandler) AdvanceStage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can advance stages")
		return
	}

	if tournament.Format != domain.FormatMultiStage {
		writeError(w, http.StatusBadRequest, "tournament is not a multi-stage tournament")
		return
	}

	// Get all stages
	stages, err := h.stageRepo.GetStagesByTournament(r.Context(), tournament.ID)
	if err != nil {
		log.Printf("Error fetching stages: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch stages")
		return
	}

	// Find active stage
	var activeStage *domain.TournamentStage
	var activeIndex int
	for i, s := range stages {
		if s.IsActive {
			activeStage = s
			activeIndex = i
			break
		}
	}

	if activeStage == nil {
		writeError(w, http.StatusBadRequest, "no active stage found")
		return
	}

	// Check if all groups in current stage are complete
	groups, err := h.stageRepo.GetGroupsByStage(r.Context(), activeStage.ID)
	if err != nil {
		log.Printf("Error fetching groups: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch groups")
		return
	}

	// TODO: Check with bracket service if all brackets in this stage are complete
	// For now, we'll allow advancing manually

	// Mark current stage as complete and inactive
	activeStage.IsActive = false
	activeStage.IsComplete = true
	if err := h.stageRepo.UpdateStage(r.Context(), activeStage); err != nil {
		log.Printf("Error updating stage: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update stage")
		return
	}

	// Find next stage
	var nextStage *domain.TournamentStage
	if activeStage.StageType == domain.StageTypeGroup {
		// Look for next group stage or final stage
		for i := activeIndex + 1; i < len(stages); i++ {
			if stages[i].StageType == domain.StageTypeGroup || stages[i].StageType == domain.StageTypeFinal {
				nextStage = stages[i]
				break
			}
		}
		// Also check for final stage at order 0
		for _, s := range stages {
			if s.StageOrder == 0 && s.StageType == domain.StageTypeFinal && nextStage == nil {
				nextStage = s
				break
			}
		}
	}

	if nextStage == nil {
		// Tournament complete
		tournament.Status = domain.StatusCompleted
		if err := h.tournamentRepo.Update(r.Context(), tournament); err != nil {
			log.Printf("Error updating tournament status: %v", err)
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "tournament completed"})
		return
	}

	// Activate next stage
	nextStage.IsActive = true
	if err := h.stageRepo.UpdateStage(r.Context(), nextStage); err != nil {
		log.Printf("Error activating next stage: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to activate next stage")
		return
	}

	// Get advancing participants from current stage
	// TODO: Get standings from bracket service and determine who advances
	advancingPerGroup := 2 // default
	if activeStage.AdvancingPerGroup != nil {
		advancingPerGroup = *activeStage.AdvancingPerGroup
	}

	// For now, just return the next stage info
	_ = advancingPerGroup
	_ = groups

	criteria, _ := h.stageRepo.GetRankingCriteria(r.Context(), nextStage.ID)
	writeJSON(w, http.StatusOK, toStageResponse(nextStage, criteria))
}

// UpdateStage updates a stage's configuration (only allowed before tournament starts)
func (h *StageHandler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	stageIDStr := chi.URLParam(r, "stageId")

	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	stageID, err := strconv.ParseUint(stageIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage ID")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can update stages")
		return
	}

	// Only allow updates when tournament is in registration
	if tournament.Status != domain.StatusRegistration {
		writeError(w, http.StatusBadRequest, "stage settings can only be modified before the tournament starts")
		return
	}

	if tournament.Format != domain.FormatMultiStage {
		writeError(w, http.StatusBadRequest, "tournament is not a multi-stage tournament")
		return
	}

	// Verify stage belongs to this tournament
	stage, err := h.stageRepo.GetStageByID(r.Context(), stageID)
	if err != nil {
		if errors.Is(err, repository.ErrStageNotFound) {
			writeError(w, http.StatusNotFound, "stage not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch stage")
		return
	}

	if stage.TournamentID != tournament.ID {
		writeError(w, http.StatusNotFound, "stage not found")
		return
	}

	var req UpdateStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Apply updates
	isGroupStage := stage.StageType == domain.StageTypeGroup

	if req.Format != nil {
		format := domain.TournamentFormat(*req.Format)
		if format != domain.FormatSingleElimination &&
			format != domain.FormatDoubleElimination &&
			format != domain.FormatSwiss {
			writeError(w, http.StatusBadRequest, "format must be 'single_elimination', 'double_elimination', or 'swiss'")
			return
		}
		stage.Format = format
	}

	if req.ParticipantsPerGroup != nil {
		if !isGroupStage {
			writeError(w, http.StatusBadRequest, "participants_per_group only applies to group stages")
			return
		}
		if *req.ParticipantsPerGroup < 2 {
			writeError(w, http.StatusBadRequest, "participants_per_group must be at least 2")
			return
		}
		stage.ParticipantsPerGroup = req.ParticipantsPerGroup
	}

	if req.AdvancingPerGroup != nil {
		if !isGroupStage {
			writeError(w, http.StatusBadRequest, "advancing_per_group only applies to group stages")
			return
		}
		if *req.AdvancingPerGroup < 1 {
			writeError(w, http.StatusBadRequest, "advancing_per_group must be at least 1")
			return
		}
		stage.AdvancingPerGroup = req.AdvancingPerGroup
	}

	// Validate participants > advancing
	if isGroupStage && stage.ParticipantsPerGroup != nil && stage.AdvancingPerGroup != nil {
		if *stage.AdvancingPerGroup >= *stage.ParticipantsPerGroup {
			writeError(w, http.StatusBadRequest, "advancing_per_group must be less than participants_per_group")
			return
		}
	}

	if req.SwissRounds != nil {
		stage.SwissRounds = req.SwissRounds
	}

	if req.SkipFinals != nil {
		stage.SkipFinals = *req.SkipFinals
	}

	// Update stage
	if err := h.stageRepo.UpdateStage(r.Context(), stage); err != nil {
		log.Printf("Error updating stage %d: %v", stageID, err)
		writeError(w, http.StatusInternalServerError, "failed to update stage")
		return
	}

	// Update ranking criteria if provided
	if req.RankingCriteria != nil {
		// Validate criteria
		for _, c := range req.RankingCriteria {
			if !domain.IsValidRankingCriterion(domain.RankingCriterion(c)) {
				writeError(w, http.StatusBadRequest, "invalid ranking criterion: "+c)
				return
			}
		}

		// Delete existing and create new
		if err := h.stageRepo.DeleteRankingCriteria(r.Context(), stageID); err != nil {
			log.Printf("Error deleting ranking criteria for stage %d: %v", stageID, err)
			writeError(w, http.StatusInternalServerError, "failed to update ranking criteria")
			return
		}

		if len(req.RankingCriteria) > 0 {
			criteria := make([]*domain.StageRankingCriteria, len(req.RankingCriteria))
			for i, c := range req.RankingCriteria {
				criteria[i] = &domain.StageRankingCriteria{
					StageID:   stageID,
					Criterion: domain.RankingCriterion(c),
					Priority:  i + 1,
				}
			}
			if err := h.stageRepo.CreateRankingCriteria(r.Context(), criteria); err != nil {
				log.Printf("Error creating ranking criteria for stage %d: %v", stageID, err)
				writeError(w, http.StatusInternalServerError, "failed to update ranking criteria")
				return
			}
		}
	}

	// Fetch updated criteria and return response
	updatedCriteria, _ := h.stageRepo.GetRankingCriteria(r.Context(), stageID)
	writeJSON(w, http.StatusOK, toStageResponse(stage, updatedCriteria))
}

// Helper functions

func validateStageConfig(stage StageConfigRequest, isGroupStage bool) error {
	format := domain.TournamentFormat(stage.Format)
	if format != domain.FormatSingleElimination &&
		format != domain.FormatDoubleElimination &&
		format != domain.FormatSwiss {
		return errors.New("format must be 'single_elimination', 'double_elimination', or 'swiss'")
	}

	if isGroupStage {
		if stage.ParticipantsPerGroup == nil {
			return errors.New("participants_per_group is required")
		}
		if *stage.ParticipantsPerGroup < 2 {
			return errors.New("participants_per_group must be at least 2")
		}
		if stage.AdvancingPerGroup == nil {
			return errors.New("advancing_per_group is required")
		}
		if *stage.AdvancingPerGroup < 1 {
			return errors.New("advancing_per_group must be at least 1")
		}
		if *stage.AdvancingPerGroup >= *stage.ParticipantsPerGroup {
			return errors.New("advancing_per_group must be less than participants_per_group")
		}
	}

	// Validate ranking criteria
	for _, c := range stage.RankingCriteria {
		if !domain.IsValidRankingCriterion(domain.RankingCriterion(c)) {
			return errors.New("invalid ranking criterion: " + c)
		}
	}

	return nil
}

// serpentineAssignment distributes participants across groups using serpentine seeding
// Serpentine pattern: if we have 9 participants and 3 groups:
// Group A: 1, 6, 7 (positions 0, 5, 6)
// Group B: 2, 5, 8 (positions 1, 4, 7)
// Group C: 3, 4, 9 (positions 2, 3, 8)
func serpentineAssignment(
	participants []*domain.Participant,
	groups []*domain.StageGroup,
	stageID, tournamentID uint64,
) []*domain.GroupAssignment {
	return serpentineAssignmentWithSeeds(participants, groups, stageID, tournamentID, nil)
}

// serpentineAssignmentWithSeeds distributes participants across groups using serpentine seeding
// while respecting manual seed assignments. Manual seeds are placed first, then remaining
// participants fill via serpentine around the manual seeds.
func serpentineAssignmentWithSeeds(
	participants []*domain.Participant,
	groups []*domain.StageGroup,
	stageID, tournamentID uint64,
	manualSeeds []*domain.StageSeedAssignment,
) []*domain.GroupAssignment {
	numGroups := len(groups)
	assignments := make([]*domain.GroupAssignment, 0, len(participants))

	// Create a map from group order to group ID
	groupOrderToID := make(map[int]uint64)
	for _, g := range groups {
		groupOrderToID[g.GroupOrder] = g.ID
	}

	// Build manual seed map: participantID -> targetGroupOrder
	manualSeedMap := make(map[uint64]int)
	for _, seed := range manualSeeds {
		manualSeedMap[seed.ParticipantID] = seed.TargetGroupOrder
	}

	// Track how many participants are in each group
	groupCounts := make(map[int]int)

	// Track which participants have been assigned
	assignedParticipants := make(map[uint64]bool)

	// 1. First, place manually seeded participants
	for _, p := range participants {
		if groupOrder, ok := manualSeedMap[p.ID]; ok {
			groupID := groupOrderToID[groupOrder]
			seedInGroup := groupCounts[groupOrder] + 1

			assignments = append(assignments, &domain.GroupAssignment{
				TournamentID:  tournamentID,
				StageID:       stageID,
				GroupID:       groupID,
				ParticipantID: p.ID,
				SeedInGroup:   &seedInGroup,
			})

			groupCounts[groupOrder]++
			assignedParticipants[p.ID] = true
		}
	}

	// 2. Remaining participants fill via serpentine
	// Collect unseeded participants in their original order
	unseededParticipants := make([]*domain.Participant, 0)
	for _, p := range participants {
		if !assignedParticipants[p.ID] {
			unseededParticipants = append(unseededParticipants, p)
		}
	}

	// Apply serpentine to unseeded participants
	for i, p := range unseededParticipants {
		// Find the next available slot using serpentine pattern
		// We need to account for already-filled slots from manual seeds
		row := i / numGroups
		posInRow := i % numGroups

		var groupOrder int
		if row%2 == 0 {
			// Even rows: left to right (0, 1, 2, ...)
			groupOrder = posInRow
		} else {
			// Odd rows: right to left (2, 1, 0, ...)
			groupOrder = numGroups - 1 - posInRow
		}

		groupID := groupOrderToID[groupOrder]
		seedInGroup := groupCounts[groupOrder] + 1

		assignments = append(assignments, &domain.GroupAssignment{
			TournamentID:  tournamentID,
			StageID:       stageID,
			GroupID:       groupID,
			ParticipantID: p.ID,
			SeedInGroup:   &seedInGroup,
		})

		groupCounts[groupOrder]++
	}

	return assignments
}

// GetStageSeeds returns the manual seed assignments for a stage
func (h *StageHandler) GetStageSeeds(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	stageIDStr := chi.URLParam(r, "stageId")

	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	stageID, err := strconv.ParseUint(stageIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage ID")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	// Verify stage belongs to this tournament
	stage, err := h.stageRepo.GetStageByID(r.Context(), stageID)
	if err != nil {
		if errors.Is(err, repository.ErrStageNotFound) {
			writeError(w, http.StatusNotFound, "stage not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch stage")
		return
	}

	if stage.TournamentID != tournament.ID {
		writeError(w, http.StatusNotFound, "stage not found")
		return
	}

	// Fetch seed assignments
	seeds, err := h.stageRepo.GetSeedAssignmentsByStage(r.Context(), stageID)
	if err != nil {
		log.Printf("Error fetching seed assignments for stage %d: %v", stageID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch seed assignments")
		return
	}

	// Build response
	response := make([]StageSeedAssignmentResponse, len(seeds))
	for i, s := range seeds {
		response[i] = StageSeedAssignmentResponse{
			ID:               s.ID,
			StageID:          s.StageID,
			ParticipantID:    s.ParticipantID,
			TargetGroupOrder: s.TargetGroupOrder,
			CreatedAt:        s.CreatedAt.Format(time.RFC3339),
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// UpdateStageSeeds updates the manual seed assignments for a stage
func (h *StageHandler) UpdateStageSeeds(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	stageIDStr := chi.URLParam(r, "stageId")

	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	stageID, err := strconv.ParseUint(stageIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage ID")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can update seed assignments")
		return
	}

	// Verify stage belongs to this tournament
	stage, err := h.stageRepo.GetStageByID(r.Context(), stageID)
	if err != nil {
		if errors.Is(err, repository.ErrStageNotFound) {
			writeError(w, http.StatusNotFound, "stage not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch stage")
		return
	}

	if stage.TournamentID != tournament.ID {
		writeError(w, http.StatusNotFound, "stage not found")
		return
	}

	// Only allow seeding before stage has started (no groups created yet)
	existingGroups, err := h.stageRepo.GetGroupsByStage(r.Context(), stageID)
	if err != nil {
		log.Printf("Error checking existing groups: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to check stage status")
		return
	}

	if len(existingGroups) > 0 {
		writeError(w, http.StatusBadRequest, "cannot modify seeds after stage has started")
		return
	}

	var req UpdateStageSeedsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get participants to validate participant IDs
	participants, err := h.participantRepo.GetByTournament(r.Context(), tournament.ID)
	if err != nil {
		log.Printf("Error fetching participants: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to validate participants")
		return
	}

	participantIDs := make(map[uint64]bool)
	for _, p := range participants {
		participantIDs[p.ID] = true
	}

	// Calculate expected number of groups
	participantsPerGroup := 4 // default
	if stage.ParticipantsPerGroup != nil {
		participantsPerGroup = *stage.ParticipantsPerGroup
	}
	expectedGroups := (len(participants) + participantsPerGroup - 1) / participantsPerGroup

	// Validate seeds
	seenParticipants := make(map[uint64]bool)
	for _, seed := range req.Seeds {
		if !participantIDs[seed.ParticipantID] {
			writeError(w, http.StatusBadRequest, "invalid participant_id: "+strconv.FormatUint(seed.ParticipantID, 10))
			return
		}
		if seenParticipants[seed.ParticipantID] {
			writeError(w, http.StatusBadRequest, "duplicate participant_id: "+strconv.FormatUint(seed.ParticipantID, 10))
			return
		}
		seenParticipants[seed.ParticipantID] = true

		if seed.TargetGroupOrder < 0 || seed.TargetGroupOrder >= expectedGroups {
			writeError(w, http.StatusBadRequest, "target_group_order must be between 0 and "+strconv.Itoa(expectedGroups-1))
			return
		}
	}

	// Convert to domain objects
	assignments := make([]*domain.StageSeedAssignment, len(req.Seeds))
	for i, seed := range req.Seeds {
		assignments[i] = &domain.StageSeedAssignment{
			ParticipantID:    seed.ParticipantID,
			TargetGroupOrder: seed.TargetGroupOrder,
		}
	}

	// Update seed assignments
	if err := h.stageRepo.UpdateSeedAssignments(r.Context(), stageID, tournament.ID, assignments); err != nil {
		log.Printf("Error updating seed assignments: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update seed assignments")
		return
	}

	// Fetch and return updated seeds
	seeds, err := h.stageRepo.GetSeedAssignmentsByStage(r.Context(), stageID)
	if err != nil {
		log.Printf("Error fetching updated seed assignments: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch updated seeds")
		return
	}

	response := make([]StageSeedAssignmentResponse, len(seeds))
	for i, s := range seeds {
		response[i] = StageSeedAssignmentResponse{
			ID:               s.ID,
			StageID:          s.StageID,
			ParticipantID:    s.ParticipantID,
			TargetGroupOrder: s.TargetGroupOrder,
			CreatedAt:        s.CreatedAt.Format(time.RFC3339),
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// Stage Participant Pool handlers

// GetStagePool returns the current participant pool assignments for a tournament
func (h *StageHandler) GetStagePool(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	if tournament.Format != domain.FormatMultiStage {
		writeError(w, http.StatusBadRequest, "tournament is not a multi-stage tournament")
		return
	}

	// Get pool entries
	entries, err := h.stageRepo.GetParticipantPoolByTournament(r.Context(), tournament.ID)
	if err != nil {
		log.Printf("Error fetching participant pool: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch participant pool")
		return
	}

	// Get stages for stage_order lookup
	stages, err := h.stageRepo.GetStagesByTournament(r.Context(), tournament.ID)
	if err != nil {
		log.Printf("Error fetching stages: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch stages")
		return
	}

	stageOrderMap := make(map[uint64]int)
	for _, s := range stages {
		stageOrderMap[s.ID] = s.StageOrder
	}

	poolResponse := StagePoolResponse{
		Entries: make([]StagePoolEntryResponse, len(entries)),
	}
	for i, e := range entries {
		poolResponse.Entries[i] = StagePoolEntryResponse{
			ParticipantID: e.ParticipantID,
			StageID:       e.StageID,
			StageOrder:    stageOrderMap[e.StageID],
		}
	}

	writeJSON(w, http.StatusOK, poolResponse)
}

// UpdateStagePool auto-seeds participants to stages based on tournament seeding
func (h *StageHandler) UpdateStagePool(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can update the stage pool")
		return
	}

	if tournament.Status != domain.StatusRegistration {
		writeError(w, http.StatusBadRequest, "tournament must be in registration status")
		return
	}

	if tournament.Format != domain.FormatMultiStage {
		writeError(w, http.StatusBadRequest, "tournament is not a multi-stage tournament")
		return
	}

	var req UpdateStagePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get stages
	stages, err := h.stageRepo.GetStagesByTournament(r.Context(), tournament.ID)
	if err != nil {
		log.Printf("Error fetching stages: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch stages")
		return
	}

	stageMap := make(map[uint64]*domain.TournamentStage)
	for _, s := range stages {
		stageMap[s.ID] = s
	}

	// Validate all stage IDs belong to this tournament
	for _, c := range req.Config {
		if _, exists := stageMap[c.StageID]; !exists {
			writeError(w, http.StatusBadRequest, "stage not found in tournament")
			return
		}
		if c.Count < 0 {
			writeError(w, http.StatusBadRequest, "count cannot be negative")
			return
		}
	}

	// Get participants ordered by seed
	participants, err := h.participantRepo.GetByTournament(r.Context(), tournament.ID)
	if err != nil {
		log.Printf("Error fetching participants: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch participants")
		return
	}

	// Validate total counts
	totalConfigured := 0
	for _, c := range req.Config {
		totalConfigured += c.Count
	}

	if totalConfigured > len(participants) {
		writeError(w, http.StatusBadRequest, "total count exceeds number of participants")
		return
	}

	// Validate participant counts for each stage format
	for _, c := range req.Config {
		stage := stageMap[c.StageID]
		if c.Count > 0 {
			if err := validateParticipantCountForStage(stage, c.Count); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
		}
	}

	// Find the first stage (lowest stage_order > 0) for remaining participants
	var firstStageID uint64
	firstStageOrder := 999
	for _, s := range stages {
		if s.StageOrder > 0 && s.StageOrder < firstStageOrder {
			firstStageOrder = s.StageOrder
			firstStageID = s.ID
		}
	}

	// Build pool entries
	entries := make([]*domain.StageParticipantPool, 0, len(participants))
	cursor := 0

	// Process configured stages (take top N participants for each)
	for _, c := range req.Config {
		for i := 0; i < c.Count && cursor < len(participants); i++ {
			entries = append(entries, &domain.StageParticipantPool{
				TournamentID:  tournament.ID,
				StageID:       c.StageID,
				ParticipantID: participants[cursor].ID,
			})
			cursor++
		}
	}

	// Remaining participants go to first stage
	for cursor < len(participants) {
		entries = append(entries, &domain.StageParticipantPool{
			TournamentID:  tournament.ID,
			StageID:       firstStageID,
			ParticipantID: participants[cursor].ID,
		})
		cursor++
	}

	// Validate remaining participants count for first stage
	remainingCount := len(participants) - totalConfigured
	if remainingCount > 0 && firstStageID != 0 {
		firstStage := stageMap[firstStageID]
		if err := validateParticipantCountForStage(firstStage, remainingCount); err != nil {
			writeError(w, http.StatusBadRequest, "remaining participants: "+err.Error())
			return
		}
	}

	// Save pool entries
	if err := h.stageRepo.ReplaceParticipantPool(r.Context(), tournament.ID, entries); err != nil {
		log.Printf("Error saving participant pool: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to save participant pool")
		return
	}

	// Build response
	stageOrderMap := make(map[uint64]int)
	for _, s := range stages {
		stageOrderMap[s.ID] = s.StageOrder
	}

	poolResponse := StagePoolResponse{
		Entries: make([]StagePoolEntryResponse, len(entries)),
	}
	for i, e := range entries {
		poolResponse.Entries[i] = StagePoolEntryResponse{
			ParticipantID: e.ParticipantID,
			StageID:       e.StageID,
			StageOrder:    stageOrderMap[e.StageID],
		}
	}

	writeJSON(w, http.StatusOK, poolResponse)
}

// ClearStagePool removes all stage pool assignments
func (h *StageHandler) ClearStagePool(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can clear the stage pool")
		return
	}

	if tournament.Status != domain.StatusRegistration {
		writeError(w, http.StatusBadRequest, "tournament must be in registration status")
		return
	}

	if tournament.Format != domain.FormatMultiStage {
		writeError(w, http.StatusBadRequest, "tournament is not a multi-stage tournament")
		return
	}

	if err := h.stageRepo.DeleteParticipantPoolByTournament(r.Context(), tournament.ID); err != nil {
		log.Printf("Error clearing participant pool: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to clear participant pool")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateParticipantCountForStage validates that the participant count is valid for a stage format
func validateParticipantCountForStage(stage *domain.TournamentStage, count int) error {
	if count < 2 {
		return errors.New("need at least 2 participants for stage")
	}

	if stage.StageType == domain.StageTypeGroup {
		perGroup := 4
		if stage.ParticipantsPerGroup != nil {
			perGroup = *stage.ParticipantsPerGroup
		}

		// Need enough participants to fill at least one group
		if count < perGroup {
			return errors.New("need at least " + strconv.Itoa(perGroup) + " participants for group stage, got " + strconv.Itoa(count))
		}
	}

	return nil
}
