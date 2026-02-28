package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/braccet/tournament/internal/api/middleware"
	"github.com/braccet/tournament/internal/client"
	"github.com/braccet/tournament/internal/domain"
	"github.com/braccet/tournament/internal/repository"
	"github.com/go-chi/chi/v5"
)

type TournamentHandler struct {
	repo            repository.TournamentRepository
	participantRepo repository.ParticipantRepository
	stageRepo       repository.StageRepository
	bracketClient   client.BracketClient
	communityClient client.CommunityClient
}

func NewTournamentHandler(repo repository.TournamentRepository, participantRepo repository.ParticipantRepository, stageRepo repository.StageRepository, bracketClient client.BracketClient, communityClient client.CommunityClient) *TournamentHandler {
	return &TournamentHandler{
		repo:            repo,
		participantRepo: participantRepo,
		stageRepo:       stageRepo,
		bracketClient:   bracketClient,
		communityClient: communityClient,
	}
}

// Request/Response types

type CreateTournamentRequest struct {
	Name            string   `json:"name"`
	Description     *string  `json:"description,omitempty"`
	Game            *string  `json:"game,omitempty"`
	Format          string   `json:"format"`
	MaxParticipants *uint    `json:"max_participants,omitempty"`
	SwissRounds     *int     `json:"swiss_rounds,omitempty"`
	StartsAt        *string  `json:"starts_at,omitempty"`
	CommunityID     *uint64  `json:"community_id,omitempty"`
	EloSystemID     *uint64  `json:"elo_system_id,omitempty"`
	PRSystemID      *uint64  `json:"pr_system_id,omitempty"`
	TournamentClass *string  `json:"tournament_class,omitempty"`
	PrizePoolUSD    *float64 `json:"prize_pool_usd,omitempty"`
}

type UpdateTournamentRequest struct {
	Name              *string                    `json:"name,omitempty"`
	Description       *string                    `json:"description,omitempty"`
	Game              *string                    `json:"game,omitempty"`
	Format            *string                    `json:"format,omitempty"`
	Status            *string                    `json:"status,omitempty"`
	MaxParticipants   *uint                      `json:"max_participants,omitempty"`
	SwissRounds       *int                       `json:"swiss_rounds,omitempty"`
	RegistrationOpen  *bool                      `json:"registration_open,omitempty"`
	StartsAt          *string                    `json:"starts_at,omitempty"`
	CommunityID       *uint64                    `json:"community_id,omitempty"`
	EloSystemID       *uint64                    `json:"elo_system_id,omitempty"`
	PRSystemID        *uint64                    `json:"pr_system_id,omitempty"`
	TournamentClass   *string                    `json:"tournament_class,omitempty"`
	PrizePoolUSD      *float64                   `json:"prize_pool_usd,omitempty"`
	PrizeDistribution *PrizeDistributionRequest  `json:"prize_distribution,omitempty"`
}

// PrizeDistributionRequest is the request format for prize distribution
type PrizeDistributionRequest struct {
	Mode  string                   `json:"mode"` // "percentage" or "amount"
	Tiers []PlacementTierRequest   `json:"tiers"`
}

// PlacementTierRequest is a single tier in the prize distribution request
type PlacementTierRequest struct {
	Placement string  `json:"placement"`
	Low       int     `json:"low"`
	High      int     `json:"high"`
	Value     float64 `json:"value"`
}

type TournamentResponse struct {
	ID                uint64                     `json:"id"`
	Slug              string                     `json:"slug"`
	OrganizerID       uint64                     `json:"organizer_id"`
	CommunityID       *uint64                    `json:"community_id,omitempty"`
	EloSystemID       *uint64                    `json:"elo_system_id,omitempty"`
	PRSystemID        *uint64                    `json:"pr_system_id,omitempty"`
	Name              string                     `json:"name"`
	Description       *string                    `json:"description,omitempty"`
	Game              *string                    `json:"game,omitempty"`
	Format            string                     `json:"format"`
	Status            string                     `json:"status"`
	MaxParticipants   *uint                      `json:"max_participants,omitempty"`
	SwissRounds       *int                       `json:"swiss_rounds,omitempty"`
	ParticipantCount  *int                       `json:"participant_count,omitempty"`
	ParticipantIcons  []string                   `json:"participant_icons,omitempty"` // First N participant icon URLs for preview
	RegistrationOpen  bool                       `json:"registration_open"`
	LogoURL           *string                    `json:"logo_url,omitempty"`
	StartsAt          *string                    `json:"starts_at,omitempty"`
	TournamentClass   *string                    `json:"tournament_class,omitempty"`
	PrizePoolUSD      *float64                   `json:"prize_pool_usd,omitempty"`
	PrizeDistribution *PrizeDistributionResponse `json:"prize_distribution,omitempty"`
	EventID           *uint64                    `json:"event_id,omitempty"`
	EventRole         *string                    `json:"event_role,omitempty"` // "qualifier" or "main"
	CreatedAt         string                     `json:"created_at"`
	UpdatedAt         string                     `json:"updated_at"`
}

// PrizeDistributionResponse is the response format for prize distribution
type PrizeDistributionResponse struct {
	Mode            string             `json:"mode"`
	Tiers           []PlacementTierRequest `json:"tiers"`
	ComputedAmounts map[string]float64 `json:"computed_amounts,omitempty"` // Placement -> USD amount (for percentage mode)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

const slugChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// validTournamentClasses defines the allowed tournament class values
var validTournamentClasses = map[string]domain.TournamentClass{
	"major":       domain.TournamentClassMajor,
	"world_lan":   domain.TournamentClassWorldLAN,
	"continental": domain.TournamentClassContinental,
	"regional":    domain.TournamentClassRegional,
	"online":      domain.TournamentClassOnline,
}

// parseTournamentClass validates and converts a string to TournamentClass
func parseTournamentClass(s string) (*domain.TournamentClass, bool) {
	if s == "" {
		return nil, true
	}
	if class, ok := validTournamentClasses[s]; ok {
		return &class, true
	}
	return nil, false
}

func generateSlug() string {
	b := make([]byte, 8)
	rand.Read(b)
	var sb strings.Builder
	for _, v := range b {
		sb.WriteByte(slugChars[int(v)%len(slugChars)])
	}
	return sb.String()
}

func toTournamentResponse(t *domain.Tournament) TournamentResponse {
	resp := TournamentResponse{
		ID:               t.ID,
		Slug:             t.Slug,
		OrganizerID:      t.OrganizerID,
		CommunityID:      t.CommunityID,
		EloSystemID:      t.EloSystemID,
		PRSystemID:       t.PRSystemID,
		Name:             t.Name,
		Description:      t.Description,
		Game:             t.Game,
		Format:           string(t.Format),
		Status:           string(t.Status),
		MaxParticipants:  t.MaxParticipants,
		SwissRounds:      t.SwissRounds,
		RegistrationOpen: t.RegistrationOpen,
		LogoURL:          t.LogoURL,
		PrizePoolUSD:     t.PrizePoolUSD,
		EventID:          t.EventID,
		EventRole:        t.EventRole,
		CreatedAt:        t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        t.UpdatedAt.Format(time.RFC3339),
	}
	if t.StartsAt != nil {
		startsAt := t.StartsAt.Format(time.RFC3339)
		resp.StartsAt = &startsAt
	}
	if t.TournamentClass != nil {
		class := string(*t.TournamentClass)
		resp.TournamentClass = &class
	}

	// Parse prize distribution from settings
	if len(t.Settings) > 0 && string(t.Settings) != "{}" {
		var settings domain.TournamentSettings
		if err := json.Unmarshal(t.Settings, &settings); err == nil && settings.PrizeDistribution != nil {
			pd := settings.PrizeDistribution
			prizeResp := &PrizeDistributionResponse{
				Mode:  string(pd.Mode),
				Tiers: make([]PlacementTierRequest, len(pd.Tiers)),
			}
			for i, tier := range pd.Tiers {
				prizeResp.Tiers[i] = PlacementTierRequest{
					Placement: tier.Placement,
					Low:       tier.Low,
					High:      tier.High,
					Value:     tier.Value,
				}
			}
			// Compute actual amounts if in percentage mode
			if pd.Mode == domain.PrizeDistModePercentage && t.PrizePoolUSD != nil {
				prizeResp.ComputedAmounts = make(map[string]float64)
				for _, tier := range pd.Tiers {
					amount := (*t.PrizePoolUSD * tier.Value) / 100.0
					prizeResp.ComputedAmounts[tier.Placement] = amount
				}
			}
			resp.PrizeDistribution = prizeResp
		}
	}

	return resp
}

const maxParticipantIconsPreview = 8

// buildTournamentListResponse builds the tournament list response with participant counts and icons.
func (h *TournamentHandler) buildTournamentListResponse(ctx context.Context, tournaments []*domain.Tournament) []TournamentResponse {
	if len(tournaments) == 0 {
		return []TournamentResponse{}
	}

	// Collect tournament IDs
	tournamentIDs := make([]uint64, len(tournaments))
	for i, t := range tournaments {
		tournamentIDs[i] = t.ID
	}

	// Get community member IDs for all tournaments (limited to preview count)
	memberIDsByTournament, err := h.participantRepo.GetCommunityMemberIDsByTournaments(ctx, tournamentIDs, maxParticipantIconsPreview)
	if err != nil {
		log.Printf("Warning: failed to fetch participant member IDs: %v", err)
		memberIDsByTournament = make(map[uint64][]uint64)
	}

	// Collect all unique member IDs to fetch icons
	memberIDSet := make(map[uint64]bool)
	for _, memberIDs := range memberIDsByTournament {
		for _, id := range memberIDs {
			memberIDSet[id] = true
		}
	}
	allMemberIDs := make([]uint64, 0, len(memberIDSet))
	for id := range memberIDSet {
		allMemberIDs = append(allMemberIDs, id)
	}

	// Fetch icon URLs for all members
	var memberData map[uint64]client.MemberDataResponse
	if len(allMemberIDs) > 0 {
		memberData, err = h.communityClient.GetBulkMemberData(ctx, allMemberIDs)
		if err != nil {
			log.Printf("Warning: failed to fetch member icons: %v", err)
			memberData = make(map[uint64]client.MemberDataResponse)
		}
	}

	// Build response with counts and icons
	response := make([]TournamentResponse, len(tournaments))
	for i, t := range tournaments {
		resp := toTournamentResponse(t)

		// Fetch participant count
		count, err := h.participantRepo.CountByTournament(ctx, t.ID)
		if err == nil {
			resp.ParticipantCount = &count
		}

		// Build icon list for this tournament
		if memberIDs, ok := memberIDsByTournament[t.ID]; ok && len(memberIDs) > 0 {
			icons := make([]string, 0, len(memberIDs))
			for _, memberID := range memberIDs {
				if data, ok := memberData[memberID]; ok && data.IconURL != nil && *data.IconURL != "" {
					icons = append(icons, *data.IconURL)
				}
			}
			if len(icons) > 0 {
				resp.ParticipantIcons = icons
			}
		}

		response[i] = resp
	}

	return response
}

// List returns all tournaments for the authenticated user
func (h *TournamentHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tournaments, err := h.repo.ListByOrganizer(r.Context(), userID)
	if err != nil {
		log.Printf("Error fetching tournaments for user %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch tournaments")
		return
	}

	response := h.buildTournamentListResponse(r.Context(), tournaments)
	writeJSON(w, http.StatusOK, response)
}

// ListByCommunity returns all tournaments for a specific community
func (h *TournamentHandler) ListByCommunity(w http.ResponseWriter, r *http.Request) {
	communityIDStr := chi.URLParam(r, "communityId")
	communityID, err := strconv.ParseUint(communityIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid community ID")
		return
	}

	tournaments, err := h.repo.ListByCommunityID(r.Context(), communityID)
	if err != nil {
		log.Printf("Error fetching tournaments for community %d: %v", communityID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch tournaments")
		return
	}

	response := h.buildTournamentListResponse(r.Context(), tournaments)
	writeJSON(w, http.StatusOK, response)
}

// Create creates a new tournament
func (h *TournamentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateTournamentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	format := domain.TournamentFormat(req.Format)
	if format != domain.FormatSingleElimination && format != domain.FormatDoubleElimination && format != domain.FormatSwiss {
		writeError(w, http.StatusBadRequest, "format must be 'single_elimination', 'double_elimination', or 'swiss'")
		return
	}

	// Validate tournament class if provided
	var tournamentClass *domain.TournamentClass
	if req.TournamentClass != nil {
		var valid bool
		tournamentClass, valid = parseTournamentClass(*req.TournamentClass)
		if !valid {
			writeError(w, http.StatusBadRequest, "tournament_class must be one of: major, world_lan, continental, regional, online")
			return
		}
	}

	tournament := &domain.Tournament{
		Slug:             generateSlug(),
		OrganizerID:      userID,
		CommunityID:      req.CommunityID,
		EloSystemID:      req.EloSystemID,
		PRSystemID:       req.PRSystemID,
		Name:             req.Name,
		Description:      req.Description,
		Game:             req.Game,
		Format:           format,
		Status:           domain.StatusRegistration,
		MaxParticipants:  req.MaxParticipants,
		SwissRounds:      req.SwissRounds,
		RegistrationOpen: true,
		Settings:         json.RawMessage(`{}`),
		TournamentClass:  tournamentClass,
		PrizePoolUSD:     req.PrizePoolUSD,
	}

	if req.StartsAt != nil {
		startsAt, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starts_at format (use RFC3339)")
			return
		}
		tournament.StartsAt = &startsAt
	}

	if err := h.repo.Create(r.Context(), tournament); err != nil {
		log.Printf("Error creating tournament: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create tournament")
		return
	}

	// Fetch the created tournament to get timestamps
	created, err := h.repo.GetBySlug(r.Context(), tournament.Slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created tournament")
		return
	}

	writeJSON(w, http.StatusCreated, toTournamentResponse(created))
}

// Get returns a single tournament by slug
func (h *TournamentHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	tournament, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	resp := toTournamentResponse(tournament)

	// Populate participant count
	count, err := h.participantRepo.CountByTournament(r.Context(), tournament.ID)
	if err == nil {
		resp.ParticipantCount = &count
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetByID returns a single tournament by ID (internal endpoint for service-to-service calls)
func (h *TournamentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	tournament, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	writeJSON(w, http.StatusOK, toTournamentResponse(tournament))
}

// Update updates a tournament
func (h *TournamentHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	tournament, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	// Only organizer can update
	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only update your own tournaments")
		return
	}

	var req UpdateTournamentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Apply updates
	if req.Name != nil {
		tournament.Name = *req.Name
	}
	if req.Description != nil {
		tournament.Description = req.Description
	}
	if req.Game != nil {
		tournament.Game = req.Game
	}
	if req.Format != nil {
		format := domain.TournamentFormat(*req.Format)
		if format != domain.FormatSingleElimination && format != domain.FormatDoubleElimination && format != domain.FormatSwiss {
			writeError(w, http.StatusBadRequest, "format must be 'single_elimination', 'double_elimination', or 'swiss'")
			return
		}
		tournament.Format = format
	}
	if req.Status != nil {
		newStatus := domain.TournamentStatus(*req.Status)
		// Validate: can only complete a tournament if all matches are done
		if newStatus == domain.StatusCompleted && tournament.Status == domain.StatusInProgress {
			isComplete, err := h.bracketClient.IsBracketComplete(r.Context(), tournament.ID)
			if err != nil {
				log.Printf("Error checking bracket completion: %v", err)
				writeError(w, http.StatusInternalServerError, "failed to verify bracket completion")
				return
			}
			if !isComplete {
				writeError(w, http.StatusBadRequest, "cannot end tournament: not all matches are completed")
				return
			}
		}
		// Resetting tournament: in_progress -> registration
		if newStatus == domain.StatusRegistration && tournament.Status == domain.StatusInProgress {
			// 1. Delete bracket data (includes ELO reversion, group standings, stage standings)
			if err := h.bracketClient.DeleteBracket(r.Context(), tournament.ID); err != nil {
				log.Printf("Error deleting bracket for tournament %d: %v", tournament.ID, err)
				writeError(w, http.StatusInternalServerError, "failed to delete bracket data")
				return
			}

			// 2. Reset all participant statuses to registered
			if err := h.participantRepo.ResetAllStatuses(r.Context(), tournament.ID, domain.ParticipantRegistered); err != nil {
				log.Printf("Error resetting participant statuses for tournament %d: %v", tournament.ID, err)
				writeError(w, http.StatusInternalServerError, "failed to reset participant statuses")
				return
			}

			// 3. For multi-stage tournaments, clean up stage-related data
			if tournament.Format == domain.FormatMultiStage && h.stageRepo != nil {
				// Delete group assignments
				if err := h.stageRepo.DeleteAssignmentsByTournament(r.Context(), tournament.ID); err != nil {
					log.Printf("Error deleting group assignments for tournament %d: %v", tournament.ID, err)
					writeError(w, http.StatusInternalServerError, "failed to delete group assignments")
					return
				}

				// Delete groups
				if err := h.stageRepo.DeleteGroupsByTournament(r.Context(), tournament.ID); err != nil {
					log.Printf("Error deleting groups for tournament %d: %v", tournament.ID, err)
					writeError(w, http.StatusInternalServerError, "failed to delete groups")
					return
				}

				// Reset stage statuses (first group stage active, all not complete)
				if err := h.stageRepo.ResetStageStatuses(r.Context(), tournament.ID); err != nil {
					log.Printf("Error resetting stage statuses for tournament %d: %v", tournament.ID, err)
					writeError(w, http.StatusInternalServerError, "failed to reset stage statuses")
					return
				}
			}

			// 4. Re-open registration
			tournament.RegistrationOpen = true
		}
		// When starting a tournament, link participants to community members and close registration
		if newStatus == domain.StatusInProgress && tournament.Status == domain.StatusRegistration {
			// Automatically close registration when starting
			tournament.RegistrationOpen = false

			if tournament.CommunityID != nil {
				if err := h.linkParticipantsToCommunity(r.Context(), tournament); err != nil {
					log.Printf("Error linking participants to community: %v", err)
					writeError(w, http.StatusInternalServerError, "failed to link participants to community")
					return
				}
			}

			// For single-stage tournaments, generate the bracket now
			if tournament.Format != domain.FormatMultiStage {
				if err := h.generateSingleStageBracket(r.Context(), tournament); err != nil {
					log.Printf("Error generating bracket for tournament %d: %v", tournament.ID, err)
					writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate bracket: %v", err))
					return
				}
			}
		}
		// Close registration for any non-registration status
		if newStatus != domain.StatusRegistration {
			tournament.RegistrationOpen = false
		}
		tournament.Status = newStatus
	}
	if req.MaxParticipants != nil {
		tournament.MaxParticipants = req.MaxParticipants
	}
	if req.SwissRounds != nil {
		tournament.SwissRounds = req.SwissRounds
	}
	if req.RegistrationOpen != nil {
		// Only allow opening registration if tournament is in registration status
		if *req.RegistrationOpen && tournament.Status != domain.StatusRegistration {
			writeError(w, http.StatusBadRequest, "cannot open registration for a tournament that is not in registration status")
			return
		}
		tournament.RegistrationOpen = *req.RegistrationOpen
	}
	if req.StartsAt != nil {
		startsAt, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starts_at format (use RFC3339)")
			return
		}
		tournament.StartsAt = &startsAt
	}
	if req.CommunityID != nil {
		tournament.CommunityID = req.CommunityID
	}
	if req.EloSystemID != nil {
		tournament.EloSystemID = req.EloSystemID
	}
	if req.PRSystemID != nil {
		tournament.PRSystemID = req.PRSystemID
	}
	if req.TournamentClass != nil {
		// Only allow updating tournament class before tournament starts
		if tournament.Status != domain.StatusRegistration {
			writeError(w, http.StatusBadRequest, "cannot change tournament class after tournament has started")
			return
		}
		tournamentClass, valid := parseTournamentClass(*req.TournamentClass)
		if !valid {
			writeError(w, http.StatusBadRequest, "tournament_class must be one of: major, world_lan, continental, regional, online")
			return
		}
		tournament.TournamentClass = tournamentClass
	}
	if req.PrizePoolUSD != nil {
		// Only allow updating prize pool before tournament starts
		if tournament.Status != domain.StatusRegistration {
			writeError(w, http.StatusBadRequest, "cannot change prize pool after tournament has started")
			return
		}
		tournament.PrizePoolUSD = req.PrizePoolUSD
	}
	if req.PrizeDistribution != nil {
		// Only allow updating prize distribution before tournament starts
		if tournament.Status != domain.StatusRegistration {
			writeError(w, http.StatusBadRequest, "cannot change prize distribution after tournament has started")
			return
		}

		// Determine the effective prize pool (could be updated in same request)
		effectivePrizePool := tournament.PrizePoolUSD
		if req.PrizePoolUSD != nil {
			effectivePrizePool = req.PrizePoolUSD
		}

		if err := validatePrizeDistribution(req.PrizeDistribution, effectivePrizePool); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Convert request to domain model and store in settings
		pd := &domain.PrizeDistribution{
			Mode:  domain.PrizeDistributionMode(req.PrizeDistribution.Mode),
			Tiers: make([]domain.PlacementTier, len(req.PrizeDistribution.Tiers)),
		}
		for i, t := range req.PrizeDistribution.Tiers {
			pd.Tiers[i] = domain.PlacementTier{
				Placement: t.Placement,
				Low:       t.Low,
				High:      t.High,
				Value:     t.Value,
			}
		}

		settings := domain.TournamentSettings{PrizeDistribution: pd}
		settingsJSON, err := json.Marshal(settings)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to serialize prize distribution")
			return
		}
		tournament.Settings = settingsJSON
	}

	if err := h.repo.Update(r.Context(), tournament); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update tournament")
		return
	}

	// Fetch updated tournament
	updated, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated tournament")
		return
	}

	writeJSON(w, http.StatusOK, toTournamentResponse(updated))
}

// Delete deletes a tournament and cleans up orphaned community members
func (h *TournamentHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

	tournament, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	// Only organizer can delete
	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only delete your own tournaments")
		return
	}

	// BEFORE deleting: identify orphaned community members
	var orphanedMemberIDs []uint64
	if tournament.CommunityID != nil {
		orphanedMemberIDs, err = h.participantRepo.GetOrphanedCommunityMemberIDs(
			r.Context(), tournament.ID, *tournament.CommunityID)
		if err != nil {
			log.Printf("Warning: failed to get orphaned members for tournament %d: %v", tournament.ID, err)
			// Continue with deletion - cleanup is best-effort
		}
	}

	// Delete the tournament (CASCADE deletes participants)
	if err := h.repo.Delete(r.Context(), slug); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete tournament")
		return
	}

	// AFTER deleting: clean up orphaned community members
	if len(orphanedMemberIDs) > 0 && tournament.CommunityID != nil {
		deleted, err := h.communityClient.DeleteMembers(
			r.Context(), *tournament.CommunityID, orphanedMemberIDs)
		if err != nil {
			log.Printf("Warning: failed to cleanup %d community members for community %d: %v",
				len(orphanedMemberIDs), *tournament.CommunityID, err)
			// Don't fail the request - tournament is deleted, cleanup is best-effort
		} else if deleted > 0 {
			log.Printf("Cleaned up %d orphaned community members from community %d", deleted, *tournament.CommunityID)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// linkParticipantsToCommunity links all participants to community members when a tournament starts.
// For participants without a community_member_id, it finds or creates a ghost member.
func (h *TournamentHandler) linkParticipantsToCommunity(ctx context.Context, tournament *domain.Tournament) error {
	if tournament.CommunityID == nil {
		return nil
	}

	participants, err := h.participantRepo.GetByTournament(ctx, tournament.ID)
	if err != nil {
		return err
	}

	for _, p := range participants {
		if p.CommunityMemberID != nil {
			// Already linked
			continue
		}

		// Find or create community member
		member, err := h.communityClient.FindOrCreateGhostMember(ctx, *tournament.CommunityID, p.DisplayName)
		if err != nil {
			return err
		}

		// Update participant with community member ID
		if err := h.participantRepo.UpdateCommunityMemberID(ctx, p.ID, member.ID); err != nil {
			return err
		}
	}

	return nil
}

// generateSingleStageBracket creates a bracket for a single-stage (non-multi-stage) tournament.
// This is called when the tournament status changes to in_progress.
func (h *TournamentHandler) generateSingleStageBracket(ctx context.Context, tournament *domain.Tournament) error {
	// Get participants
	participants, err := h.participantRepo.GetByTournament(ctx, tournament.ID)
	if err != nil {
		return fmt.Errorf("failed to get participants: %w", err)
	}

	if len(participants) < 2 {
		return fmt.Errorf("at least 2 participants required")
	}

	// Fetch icon URLs for participants with community member IDs
	var memberIDs []uint64
	for _, p := range participants {
		if p.CommunityMemberID != nil {
			memberIDs = append(memberIDs, *p.CommunityMemberID)
		}
	}
	memberData, err := h.communityClient.GetBulkMemberData(ctx, memberIDs)
	if err != nil {
		log.Printf("Error fetching member data for icons: %v", err)
		// Non-fatal, continue without icons
		memberData = make(map[uint64]client.MemberDataResponse)
	}

	// Build participant list for bracket request
	bracketParticipants := make([]client.Participant, len(participants))
	for i, p := range participants {
		seed := 0
		if p.Seed != nil {
			seed = int(*p.Seed)
		}
		var iconURL string
		if p.CommunityMemberID != nil {
			if data, ok := memberData[*p.CommunityMemberID]; ok && data.IconURL != nil {
				iconURL = *data.IconURL
			}
		}
		bracketParticipants[i] = client.Participant{
			ID:      p.ID,
			Name:    p.DisplayName,
			Seed:    seed,
			IconURL: iconURL,
		}
	}

	// Create bracket request
	req := client.CreateBracketRequest{
		TournamentID: tournament.ID,
		Format:       string(tournament.Format),
		Participants: bracketParticipants,
		SwissRounds:  tournament.SwissRounds,
	}

	log.Printf("Creating bracket for tournament %d: format=%s, participants=%d", tournament.ID, tournament.Format, len(bracketParticipants))
	if err := h.bracketClient.CreateBracket(ctx, req); err != nil {
		return fmt.Errorf("bracket service: %w", err)
	}
	return nil
}

// validatePrizeDistribution validates the prize distribution request
func validatePrizeDistribution(pd *PrizeDistributionRequest, totalPrizeUSD *float64) error {
	if pd == nil {
		return nil
	}

	if pd.Mode != "percentage" && pd.Mode != "amount" {
		return fmt.Errorf("mode must be 'percentage' or 'amount'")
	}

	if len(pd.Tiers) == 0 {
		return fmt.Errorf("at least one tier is required")
	}

	// Validate tier values are non-negative and bounds are valid
	for _, tier := range pd.Tiers {
		if tier.Value < 0 {
			return fmt.Errorf("tier values must be non-negative")
		}
		if tier.Low > tier.High {
			return fmt.Errorf("tier low bound cannot exceed high bound")
		}
		if tier.Low < 1 {
			return fmt.Errorf("tier low bound must be at least 1")
		}
	}

	if pd.Mode == "percentage" {
		var sum float64
		for _, tier := range pd.Tiers {
			sum += tier.Value
		}
		// Allow small floating point tolerance
		if sum < 99.99 || sum > 100.01 {
			return fmt.Errorf("percentages must sum to 100%%, got %.2f%%", sum)
		}
	} else { // amount mode
		if totalPrizeUSD == nil {
			return fmt.Errorf("total prize pool required when using amount mode")
		}
		var sum float64
		for _, tier := range pd.Tiers {
			sum += tier.Value
		}
		if sum > *totalPrizeUSD {
			return fmt.Errorf("tier amounts ($%.2f) exceed total prize pool ($%.2f)", sum, *totalPrizeUSD)
		}
	}

	return nil
}

// SuggestedPrizeTiersResponse is the response for the prize tiers endpoint
type SuggestedPrizeTiersResponse struct {
	ParticipantCount int                    `json:"participant_count"`
	Tiers            []domain.PlacementTier `json:"tiers"`
}

// GetSuggestedPrizeTiers returns suggested prize tiers for a tournament based on format and participant count
func (h *TournamentHandler) GetSuggestedPrizeTiers(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid tournament slug")
		return
	}

	tournament, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	// Determine participant count
	count := h.determineParticipantCount(r, tournament)

	// Build tier generation config based on tournament format
	config := h.buildTierConfig(r.Context(), tournament, count)

	tiers := domain.GeneratePlacementTiersConfig(config)

	writeJSON(w, http.StatusOK, SuggestedPrizeTiersResponse{
		ParticipantCount: count,
		Tiers:            tiers,
	})
}

// determineParticipantCount gets the participant count from query param, actual count, or max_participants
func (h *TournamentHandler) determineParticipantCount(r *http.Request, tournament *domain.Tournament) int {
	count := 8 // default

	if participantsStr := r.URL.Query().Get("participants"); participantsStr != "" {
		if parsed, err := strconv.Atoi(participantsStr); err == nil && parsed >= 2 {
			return parsed
		}
	}

	// Fall back to actual participant count or max_participants
	actualCount, err := h.participantRepo.CountByTournament(r.Context(), tournament.ID)
	if err == nil && actualCount >= 2 {
		return actualCount
	}

	if tournament.MaxParticipants != nil && int(*tournament.MaxParticipants) >= 2 {
		return int(*tournament.MaxParticipants)
	}

	return count
}

// buildTierConfig creates a TierGenerationConfig based on tournament format
func (h *TournamentHandler) buildTierConfig(ctx context.Context, tournament *domain.Tournament, participantCount int) domain.TierGenerationConfig {
	config := domain.TierGenerationConfig{
		Format:           tournament.Format,
		ParticipantCount: participantCount,
	}

	switch tournament.Format {
	case domain.FormatMultiStage:
		config.Stages = h.buildMultiStageConfig(ctx, tournament.ID, participantCount)
	case domain.FormatSwiss:
		// For single-stage Swiss, check for threshold mode via query params
		// (actual threshold values are stored at stage level, but we can preview with params)
		if winsStr := ctx.Value("wins_to_advance"); winsStr != nil {
			// Reserved for future: query param support for Swiss threshold preview
		}
	}

	return config
}

// buildMultiStageConfig creates StageTierConfig array for multi-stage tournaments
func (h *TournamentHandler) buildMultiStageConfig(ctx context.Context, tournamentID uint64, totalParticipants int) []domain.StageTierConfig {
	if h.stageRepo == nil {
		return nil
	}

	stages, err := h.stageRepo.GetStagesByTournament(ctx, tournamentID)
	if err != nil || len(stages) == 0 {
		return nil
	}

	// Separate final stage from group stages
	var finalStage *domain.TournamentStage
	var groupStages []*domain.TournamentStage

	for _, s := range stages {
		if s.StageType == domain.StageTypeFinal {
			finalStage = s
		} else {
			groupStages = append(groupStages, s)
		}
	}

	// Sort group stages by order (descending - later stages have higher order)
	for i := 0; i < len(groupStages)-1; i++ {
		for j := 0; j < len(groupStages)-i-1; j++ {
			if groupStages[j].StageOrder < groupStages[j+1].StageOrder {
				groupStages[j], groupStages[j+1] = groupStages[j+1], groupStages[j]
			}
		}
	}

	var stageConfigs []domain.StageTierConfig

	// Calculate participants at each stage, working backwards
	remaining := totalParticipants

	// Process group stages from latest to earliest
	for _, gs := range groupStages {
		// Use expected_participants if set, otherwise calculate
		stageParticipants := remaining
		if gs.ExpectedParticipants != nil && *gs.ExpectedParticipants > 0 {
			stageParticipants = *gs.ExpectedParticipants
		}

		advancing := 0
		if gs.AdvancingPerGroup != nil && gs.ParticipantsPerGroup != nil {
			// Number of groups = participants / participants_per_group
			numGroups := stageParticipants / *gs.ParticipantsPerGroup
			if numGroups < 1 {
				numGroups = 1
			}
			advancing = numGroups * *gs.AdvancingPerGroup
		} else {
			// Default: half advance
			advancing = stageParticipants / 2
		}

		stageConfigs = append(stageConfigs, domain.StageTierConfig{
			StageOrder:        gs.StageOrder,
			StageType:         gs.StageType,
			Format:            gs.Format,
			Participants:      stageParticipants,
			Advancing:         advancing,
			WinsToAdvance:     gs.WinsToAdvance,
			LossesToEliminate: gs.LossesToEliminate,
		})

		remaining = advancing
	}

	// Add final stage
	if finalStage != nil && remaining > 0 {
		// Use expected_participants if set, otherwise use remaining from group stages
		finalParticipants := remaining
		if finalStage.ExpectedParticipants != nil && *finalStage.ExpectedParticipants > 0 {
			finalParticipants = *finalStage.ExpectedParticipants
		}

		stageConfigs = append(stageConfigs, domain.StageTierConfig{
			StageOrder:        0,
			StageType:         domain.StageTypeFinal,
			Format:            finalStage.Format,
			Participants:      finalParticipants,
			Advancing:         0, // No one advances from final
			WinsToAdvance:     finalStage.WinsToAdvance,
			LossesToEliminate: finalStage.LossesToEliminate,
		})
	}

	return stageConfigs
}
