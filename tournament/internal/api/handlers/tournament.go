package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
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
	bracketClient   client.BracketClient
	communityClient client.CommunityClient
}

func NewTournamentHandler(repo repository.TournamentRepository, participantRepo repository.ParticipantRepository, bracketClient client.BracketClient, communityClient client.CommunityClient) *TournamentHandler {
	return &TournamentHandler{
		repo:            repo,
		participantRepo: participantRepo,
		bracketClient:   bracketClient,
		communityClient: communityClient,
	}
}

// Request/Response types

type CreateTournamentRequest struct {
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	Game            *string `json:"game,omitempty"`
	Format          string  `json:"format"`
	MaxParticipants *uint   `json:"max_participants,omitempty"`
	SwissRounds     *int    `json:"swiss_rounds,omitempty"`
	StartsAt        *string `json:"starts_at,omitempty"`
	CommunityID     *uint64 `json:"community_id,omitempty"`
	EloSystemID     *uint64 `json:"elo_system_id,omitempty"`
}

type UpdateTournamentRequest struct {
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	Game             *string `json:"game,omitempty"`
	Format           *string `json:"format,omitempty"`
	Status           *string `json:"status,omitempty"`
	MaxParticipants  *uint   `json:"max_participants,omitempty"`
	SwissRounds      *int    `json:"swiss_rounds,omitempty"`
	RegistrationOpen *bool   `json:"registration_open,omitempty"`
	StartsAt         *string `json:"starts_at,omitempty"`
	CommunityID      *uint64 `json:"community_id,omitempty"`
	EloSystemID      *uint64 `json:"elo_system_id,omitempty"`
}

type TournamentResponse struct {
	ID               uint64  `json:"id"`
	Slug             string  `json:"slug"`
	OrganizerID      uint64  `json:"organizer_id"`
	CommunityID      *uint64 `json:"community_id,omitempty"`
	EloSystemID      *uint64 `json:"elo_system_id,omitempty"`
	Name             string  `json:"name"`
	Description      *string `json:"description,omitempty"`
	Game             *string `json:"game,omitempty"`
	Format           string  `json:"format"`
	Status           string  `json:"status"`
	MaxParticipants  *uint   `json:"max_participants,omitempty"`
	SwissRounds      *int    `json:"swiss_rounds,omitempty"`
	ParticipantCount *int    `json:"participant_count,omitempty"`
	RegistrationOpen bool    `json:"registration_open"`
	StartsAt         *string `json:"starts_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
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
		Name:             t.Name,
		Description:      t.Description,
		Game:             t.Game,
		Format:           string(t.Format),
		Status:           string(t.Status),
		MaxParticipants:  t.MaxParticipants,
		SwissRounds:      t.SwissRounds,
		RegistrationOpen: t.RegistrationOpen,
		CreatedAt:        t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        t.UpdatedAt.Format(time.RFC3339),
	}
	if t.StartsAt != nil {
		startsAt := t.StartsAt.Format(time.RFC3339)
		resp.StartsAt = &startsAt
	}
	return resp
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

	response := make([]TournamentResponse, len(tournaments))
	for i, t := range tournaments {
		resp := toTournamentResponse(t)
		// Fetch participant count for each tournament
		count, err := h.participantRepo.CountByTournament(r.Context(), t.ID)
		if err == nil {
			resp.ParticipantCount = &count
		}
		response[i] = resp
	}

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

	response := make([]TournamentResponse, len(tournaments))
	for i, t := range tournaments {
		resp := toTournamentResponse(t)
		// Fetch participant count for each tournament
		count, err := h.participantRepo.CountByTournament(r.Context(), t.ID)
		if err == nil {
			resp.ParticipantCount = &count
		}
		response[i] = resp
	}

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

	tournament := &domain.Tournament{
		Slug:             generateSlug(),
		OrganizerID:      userID,
		CommunityID:      req.CommunityID,
		EloSystemID:      req.EloSystemID,
		Name:             req.Name,
		Description:      req.Description,
		Game:             req.Game,
		Format:           format,
		Status:           domain.StatusRegistration,
		MaxParticipants:  req.MaxParticipants,
		SwissRounds:      req.SwissRounds,
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

	writeJSON(w, http.StatusOK, toTournamentResponse(tournament))
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
			// 1. Delete bracket data (includes ELO reversion)
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

			// 3. Re-open registration
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
