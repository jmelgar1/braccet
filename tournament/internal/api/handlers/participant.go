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

type ParticipantHandler struct {
	participantRepo repository.ParticipantRepository
	tournamentRepo  repository.TournamentRepository
	bracketClient   client.BracketClient
	communityClient client.CommunityClient
}

func NewParticipantHandler(participantRepo repository.ParticipantRepository, tournamentRepo repository.TournamentRepository, bracketClient client.BracketClient, communityClient client.CommunityClient) *ParticipantHandler {
	return &ParticipantHandler{
		participantRepo: participantRepo,
		tournamentRepo:  tournamentRepo,
		bracketClient:   bracketClient,
		communityClient: communityClient,
	}
}

// Request/Response types

type AddParticipantRequest struct {
	UserID            *uint64 `json:"user_id,omitempty"`
	CommunityMemberID *uint64 `json:"community_member_id,omitempty"`
	DisplayName       string  `json:"display_name"`
}

type UpdateSeedingRequest struct {
	Seeds map[uint64]uint `json:"seeds"`
}

type ParticipantResponse struct {
	ID                uint64  `json:"id"`
	TournamentID      uint64  `json:"tournament_id"`
	UserID            *uint64 `json:"user_id,omitempty"`
	CommunityMemberID *uint64 `json:"community_member_id,omitempty"`
	DisplayName       string  `json:"display_name"`
	IconURL           *string `json:"icon_url,omitempty"`
	Region            *string `json:"region,omitempty"`
	Seed              *uint   `json:"seed,omitempty"`
	Status            string  `json:"status"`
	CheckedInAt       *string `json:"checked_in_at,omitempty"`
	EloRating         *int    `json:"elo_rating,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

func toParticipantResponse(p *domain.Participant) ParticipantResponse {
	resp := ParticipantResponse{
		ID:                p.ID,
		TournamentID:      p.TournamentID,
		UserID:            p.UserID,
		CommunityMemberID: p.CommunityMemberID,
		DisplayName:       p.DisplayName,
		Seed:              p.Seed,
		Status:            string(p.Status),
		CreatedAt:         p.CreatedAt.Format(time.RFC3339),
	}
	if p.CheckedInAt != nil {
		checkedInAt := p.CheckedInAt.Format(time.RFC3339)
		resp.CheckedInAt = &checkedInAt
	}
	return resp
}

func toParticipantResponseWithExtras(p *domain.Participant, eloRating *int, iconURL *string, region *string) ParticipantResponse {
	resp := toParticipantResponse(p)
	resp.EloRating = eloRating
	resp.IconURL = iconURL
	resp.Region = region
	return resp
}

// GetByID returns a single participant by ID (internal endpoint)
func (h *ParticipantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid participant ID")
		return
	}

	participant, err := h.participantRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			writeError(w, http.StatusNotFound, "participant not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch participant")
		return
	}

	writeJSON(w, http.StatusOK, toParticipantResponse(participant))
}

// List returns all participants for a tournament
func (h *ParticipantHandler) List(w http.ResponseWriter, r *http.Request) {
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

	participants, err := h.participantRepo.GetByTournament(r.Context(), tournament.ID)
	if err != nil {
		log.Printf("Error fetching participants for tournament %d: %v", tournament.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch participants")
		return
	}

	// Collect community_member_ids for enrichment
	var memberIDs []uint64
	for _, p := range participants {
		if p.CommunityMemberID != nil {
			memberIDs = append(memberIDs, *p.CommunityMemberID)
		}
	}

	// Fetch ELO ratings if tournament has an ELO system
	var eloRatings map[uint64]int
	if tournament.EloSystemID != nil && len(memberIDs) > 0 {
		eloRatings, err = h.communityClient.GetBulkMemberRatings(r.Context(), *tournament.EloSystemID, memberIDs)
		if err != nil {
			log.Printf("Warning: failed to fetch ELO ratings: %v", err)
			// Don't fail the request - just continue without ELO
		}
	}

	// Fetch icon URLs and regions for community members
	var memberData map[uint64]client.MemberDataResponse
	if len(memberIDs) > 0 {
		memberData, err = h.communityClient.GetBulkMemberData(r.Context(), memberIDs)
		if err != nil {
			log.Printf("Warning: failed to fetch member data: %v", err)
			// Don't fail the request - just continue without data
		}
	}

	response := make([]ParticipantResponse, len(participants))
	for i, p := range participants {
		var eloRating *int
		var iconURL *string
		var region *string
		if p.CommunityMemberID != nil {
			if eloRatings != nil {
				if rating, ok := eloRatings[*p.CommunityMemberID]; ok {
					eloRating = &rating
				}
			}
			if memberData != nil {
				if data, ok := memberData[*p.CommunityMemberID]; ok {
					iconURL = data.IconURL
					region = data.Region
				}
			}
		}
		response[i] = toParticipantResponseWithExtras(p, eloRating, iconURL, region)
	}

	writeJSON(w, http.StatusOK, response)
}

// Add adds a participant to a tournament
func (h *ParticipantHandler) Add(w http.ResponseWriter, r *http.Request) {
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

	// Only allow adding participants during registration phase
	if tournament.Status != domain.StatusRegistration {
		writeError(w, http.StatusBadRequest, "participants can only be added during registration")
		return
	}

	isOrganizer := tournament.OrganizerID == userID

	var req AddParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}

	// Authorization logic:
	// - Organizer can add anyone
	// - Non-organizer can only self-register if registration is open
	if !isOrganizer {
		if !tournament.RegistrationOpen {
			writeError(w, http.StatusForbidden, "registration is closed")
			return
		}
		// Self-registration: user_id must be their own or nil
		if req.UserID != nil && *req.UserID != userID {
			writeError(w, http.StatusForbidden, "you can only register yourself")
			return
		}
		// Force self-registration to use authenticated user's ID
		req.UserID = &userID
	}

	// Check for duplicate registration (only if user_id is provided)
	if req.UserID != nil {
		existing, err := h.participantRepo.GetByTournamentAndUser(r.Context(), tournament.ID, *req.UserID)
		if err == nil && existing != nil {
			writeError(w, http.StatusConflict, "user is already registered for this tournament")
			return
		}
		if err != nil && !errors.Is(err, repository.ErrParticipantNotFound) {
			writeError(w, http.StatusInternalServerError, "failed to check existing registration")
			return
		}
	}

	// Check max participants limit
	if tournament.MaxParticipants != nil {
		count, err := h.participantRepo.CountByTournament(r.Context(), tournament.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check participant count")
			return
		}
		if count >= int(*tournament.MaxParticipants) {
			writeError(w, http.StatusConflict, "tournament has reached maximum participants")
			return
		}
	}

	participant := &domain.Participant{
		TournamentID:      tournament.ID,
		UserID:            req.UserID,
		CommunityMemberID: req.CommunityMemberID,
		DisplayName:       req.DisplayName,
		Status:            domain.ParticipantRegistered,
	}

	if err := h.participantRepo.Create(r.Context(), participant); err != nil {
		log.Printf("Error creating participant: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to add participant")
		return
	}

	// Fetch the created participant
	created, err := h.participantRepo.GetByID(r.Context(), participant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created participant")
		return
	}

	// Fetch icon, region, and ELO if participant is linked to a community member
	var iconURL *string
	var region *string
	var eloRating *int
	if created.CommunityMemberID != nil {
		// Fetch icon and region
		memberData, err := h.communityClient.GetBulkMemberData(r.Context(), []uint64{*created.CommunityMemberID})
		if err == nil {
			if data, ok := memberData[*created.CommunityMemberID]; ok {
				iconURL = data.IconURL
				region = data.Region
			}
		}

		// Fetch ELO rating if tournament has an ELO system
		if tournament.EloSystemID != nil {
			ratings, err := h.communityClient.GetBulkMemberRatings(r.Context(), *tournament.EloSystemID, []uint64{*created.CommunityMemberID})
			if err == nil {
				if rating, ok := ratings[*created.CommunityMemberID]; ok {
					eloRating = &rating
				}
			}
		}
	}

	writeJSON(w, http.StatusCreated, toParticipantResponseWithExtras(created, eloRating, iconURL, region))
}

// Remove removes a participant from a tournament
func (h *ParticipantHandler) Remove(w http.ResponseWriter, r *http.Request) {
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

	participantIDStr := chi.URLParam(r, "participantId")
	participantID, err := strconv.ParseUint(participantIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid participant id")
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

	// Only allow removing participants during registration phase
	if tournament.Status != domain.StatusRegistration {
		writeError(w, http.StatusBadRequest, "participants can only be removed during registration")
		return
	}

	participant, err := h.participantRepo.GetByID(r.Context(), participantID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			writeError(w, http.StatusNotFound, "participant not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch participant")
		return
	}

	// Verify participant belongs to this tournament
	if participant.TournamentID != tournament.ID {
		writeError(w, http.StatusNotFound, "participant not found in this tournament")
		return
	}

	isOrganizer := tournament.OrganizerID == userID
	isSelf := participant.UserID != nil && *participant.UserID == userID

	// Authorization: organizer can remove anyone, users can remove themselves
	if !isOrganizer && !isSelf {
		writeError(w, http.StatusForbidden, "you can only remove yourself from the tournament")
		return
	}

	if err := h.participantRepo.Delete(r.Context(), participantID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove participant")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateSeeding updates seeds for participants (organizer only)
func (h *ParticipantHandler) UpdateSeeding(w http.ResponseWriter, r *http.Request) {
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

	// Only allow seeding changes during registration phase
	if tournament.Status != domain.StatusRegistration {
		writeError(w, http.StatusBadRequest, "seeding can only be updated during registration")
		return
	}

	// Only organizer can update seeding
	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can update seeding")
		return
	}

	var req UpdateSeedingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Seeds) == 0 {
		writeError(w, http.StatusBadRequest, "seeds map is required")
		return
	}

	if err := h.participantRepo.UpdateSeeding(r.Context(), tournament.ID, req.Seeds); err != nil {
		log.Printf("Error updating seeding: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update seeding")
		return
	}

	// Return updated participants list
	participants, err := h.participantRepo.GetByTournament(r.Context(), tournament.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch participants")
		return
	}

	response := make([]ParticipantResponse, len(participants))
	for i, p := range participants {
		response[i] = toParticipantResponse(p)
	}

	writeJSON(w, http.StatusOK, response)
}

// Withdraw withdraws a participant from an in-progress tournament
func (h *ParticipantHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
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

	participantIDStr := chi.URLParam(r, "participantId")
	participantID, err := strconv.ParseUint(participantIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid participant id")
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

	participant, err := h.participantRepo.GetByID(r.Context(), participantID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			writeError(w, http.StatusNotFound, "participant not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch participant")
		return
	}

	// Verify participant belongs to this tournament
	if participant.TournamentID != tournament.ID {
		writeError(w, http.StatusNotFound, "participant not found in this tournament")
		return
	}

	// Validate: Only allow withdrawal during in_progress tournament
	if tournament.Status != domain.StatusInProgress {
		writeError(w, http.StatusBadRequest, "withdrawal only allowed during in-progress tournaments")
		return
	}

	isOrganizer := tournament.OrganizerID == userID
	isSelf := participant.UserID != nil && *participant.UserID == userID

	// Authorization: organizer can withdraw anyone, participants can withdraw themselves
	if !isOrganizer && !isSelf {
		writeError(w, http.StatusForbidden, "you can only withdraw yourself")
		return
	}

	// Cannot withdraw if already eliminated/disqualified/withdrawn
	if participant.Status == domain.ParticipantEliminated ||
		participant.Status == domain.ParticipantDisqualified ||
		participant.Status == domain.ParticipantWithdrawn {
		writeError(w, http.StatusBadRequest, "participant is not active in tournament")
		return
	}

	// Update participant status to withdrawn
	if err := h.participantRepo.UpdateStatus(r.Context(), participantID, domain.ParticipantWithdrawn); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update participant status")
		return
	}

	// Notify bracket service to process forfeits
	if err := h.bracketClient.ProcessWithdrawal(r.Context(), tournament.ID, participantID); err != nil {
		log.Printf("Warning: failed to process bracket forfeit: %v", err)
		// Don't fail the request - status is updated, bracket service can retry
	}

	w.WriteHeader(http.StatusNoContent)
}

// Promote promotes a participant to a community member (creates a ghost member and links it)
// POST /tournaments/{slug}/participants/{participantId}/promote
func (h *ParticipantHandler) Promote(w http.ResponseWriter, r *http.Request) {
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

	participantIDStr := chi.URLParam(r, "participantId")
	participantID, err := strconv.ParseUint(participantIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid participant id")
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

	// Only organizer can promote participants
	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can promote participants")
		return
	}

	// Must be a community tournament
	if tournament.CommunityID == nil {
		writeError(w, http.StatusBadRequest, "can only promote participants in community tournaments")
		return
	}

	participant, err := h.participantRepo.GetByID(r.Context(), participantID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			writeError(w, http.StatusNotFound, "participant not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch participant")
		return
	}

	// Verify participant belongs to this tournament
	if participant.TournamentID != tournament.ID {
		writeError(w, http.StatusNotFound, "participant not found in this tournament")
		return
	}

	// Already has a community member - nothing to do
	if participant.CommunityMemberID != nil {
		// Return the participant as-is with member data
		memberData, err := h.communityClient.GetBulkMemberData(r.Context(), []uint64{*participant.CommunityMemberID})
		var iconURL *string
		var region *string
		if err == nil {
			if data, ok := memberData[*participant.CommunityMemberID]; ok {
				iconURL = data.IconURL
				region = data.Region
			}
		}
		writeJSON(w, http.StatusOK, toParticipantResponseWithExtras(participant, nil, iconURL, region))
		return
	}

	// Create ghost member in community service
	member, err := h.communityClient.FindOrCreateGhostMember(r.Context(), *tournament.CommunityID, participant.DisplayName)
	if err != nil {
		log.Printf("Error creating ghost member: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create community member")
		return
	}

	// Update participant with the new community_member_id
	if err := h.participantRepo.UpdateCommunityMemberID(r.Context(), participantID, member.ID); err != nil {
		log.Printf("Error updating participant community_member_id: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to link participant to member")
		return
	}

	// Fetch the updated participant
	updated, err := h.participantRepo.GetByID(r.Context(), participantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated participant")
		return
	}

	// Return with member data (region from member response)
	writeJSON(w, http.StatusOK, toParticipantResponseWithExtras(updated, nil, nil, member.Region))
}

// SearchMemberResponse is a simplified response for autocomplete
type SearchMemberResponse struct {
	ID          uint64 `json:"id"`
	DisplayName string `json:"display_name"`
}

// SearchAvailableMembers searches community members not yet in the tournament
// GET /tournaments/{slug}/participants/search?q=query
func (h *ParticipantHandler) SearchAvailableMembers(w http.ResponseWriter, r *http.Request) {
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

	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusOK, []SearchMemberResponse{})
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

	// Only organizer can search
	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only organizer can search members")
		return
	}

	// Must be a community tournament
	if tournament.CommunityID == nil {
		writeJSON(w, http.StatusOK, []SearchMemberResponse{})
		return
	}

	// Get existing participant community_member_ids
	participants, err := h.participantRepo.GetByTournament(r.Context(), tournament.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch participants")
		return
	}

	excludeIDs := make([]uint64, 0, len(participants))
	for _, p := range participants {
		if p.CommunityMemberID != nil {
			excludeIDs = append(excludeIDs, *p.CommunityMemberID)
		}
	}

	// Search community members
	results, err := h.communityClient.SearchMembers(r.Context(), *tournament.CommunityID, query, excludeIDs)
	if err != nil {
		log.Printf("Error searching members: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to search members")
		return
	}

	response := make([]SearchMemberResponse, len(results))
	for i, m := range results {
		response[i] = SearchMemberResponse{
			ID:          m.ID,
			DisplayName: m.DisplayName,
		}
	}

	writeJSON(w, http.StatusOK, response)
}
