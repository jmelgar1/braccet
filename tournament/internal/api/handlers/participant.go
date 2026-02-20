package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/braccet/tournament/internal/api/middleware"
	"github.com/braccet/tournament/internal/domain"
	"github.com/braccet/tournament/internal/repository"
	"github.com/go-chi/chi/v5"
)

type ParticipantHandler struct {
	participantRepo repository.ParticipantRepository
	tournamentRepo  repository.TournamentRepository
}

func NewParticipantHandler(participantRepo repository.ParticipantRepository, tournamentRepo repository.TournamentRepository) *ParticipantHandler {
	return &ParticipantHandler{
		participantRepo: participantRepo,
		tournamentRepo:  tournamentRepo,
	}
}

// Request/Response types

type AddParticipantRequest struct {
	UserID      *uint64 `json:"user_id,omitempty"`
	DisplayName string  `json:"display_name"`
}

type UpdateSeedingRequest struct {
	Seeds map[uint64]uint `json:"seeds"`
}

type ParticipantResponse struct {
	ID           uint64  `json:"id"`
	TournamentID uint64  `json:"tournament_id"`
	UserID       *uint64 `json:"user_id,omitempty"`
	DisplayName  string  `json:"display_name"`
	Seed         *uint   `json:"seed,omitempty"`
	Status       string  `json:"status"`
	CheckedInAt  *string `json:"checked_in_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

func toParticipantResponse(p *domain.Participant) ParticipantResponse {
	resp := ParticipantResponse{
		ID:           p.ID,
		TournamentID: p.TournamentID,
		UserID:       p.UserID,
		DisplayName:  p.DisplayName,
		Seed:         p.Seed,
		Status:       string(p.Status),
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
	}
	if p.CheckedInAt != nil {
		checkedInAt := p.CheckedInAt.Format(time.RFC3339)
		resp.CheckedInAt = &checkedInAt
	}
	return resp
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

	response := make([]ParticipantResponse, len(participants))
	for i, p := range participants {
		response[i] = toParticipantResponse(p)
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
		TournamentID: tournament.ID,
		UserID:       req.UserID,
		DisplayName:  req.DisplayName,
		Status:       domain.ParticipantRegistered,
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

	writeJSON(w, http.StatusCreated, toParticipantResponse(created))
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
