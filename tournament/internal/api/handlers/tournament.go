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

type TournamentHandler struct {
	repo repository.TournamentRepository
}

func NewTournamentHandler(repo repository.TournamentRepository) *TournamentHandler {
	return &TournamentHandler{repo: repo}
}

// Request/Response types

type CreateTournamentRequest struct {
	Name            string  `json:"name"`
	Description     *string `json:"description,omitempty"`
	Game            *string `json:"game,omitempty"`
	Format          string  `json:"format"`
	MaxParticipants *uint   `json:"max_participants,omitempty"`
	StartsAt        *string `json:"starts_at,omitempty"`
}

type UpdateTournamentRequest struct {
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	Game             *string `json:"game,omitempty"`
	Format           *string `json:"format,omitempty"`
	Status           *string `json:"status,omitempty"`
	MaxParticipants  *uint   `json:"max_participants,omitempty"`
	RegistrationOpen *bool   `json:"registration_open,omitempty"`
	StartsAt         *string `json:"starts_at,omitempty"`
}

type TournamentResponse struct {
	ID               uint64  `json:"id"`
	OrganizerID      uint64  `json:"organizer_id"`
	Name             string  `json:"name"`
	Description      *string `json:"description,omitempty"`
	Game             *string `json:"game,omitempty"`
	Format           string  `json:"format"`
	Status           string  `json:"status"`
	MaxParticipants  *uint   `json:"max_participants,omitempty"`
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

func toTournamentResponse(t *domain.Tournament) TournamentResponse {
	resp := TournamentResponse{
		ID:               t.ID,
		OrganizerID:      t.OrganizerID,
		Name:             t.Name,
		Description:      t.Description,
		Game:             t.Game,
		Format:           string(t.Format),
		Status:           string(t.Status),
		MaxParticipants:  t.MaxParticipants,
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
		response[i] = toTournamentResponse(t)
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
	if format != domain.FormatSingleElimination && format != domain.FormatDoubleElimination {
		writeError(w, http.StatusBadRequest, "format must be 'single_elimination' or 'double_elimination'")
		return
	}

	tournament := &domain.Tournament{
		OrganizerID:      userID,
		Name:             req.Name,
		Description:      req.Description,
		Game:             req.Game,
		Format:           format,
		Status:           domain.StatusDraft,
		MaxParticipants:  req.MaxParticipants,
		RegistrationOpen: false,
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
	created, err := h.repo.GetByID(r.Context(), tournament.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch created tournament")
		return
	}

	writeJSON(w, http.StatusCreated, toTournamentResponse(created))
}

// Get returns a single tournament by ID
func (h *TournamentHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament id")
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

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament id")
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
		if format != domain.FormatSingleElimination && format != domain.FormatDoubleElimination {
			writeError(w, http.StatusBadRequest, "format must be 'single_elimination' or 'double_elimination'")
			return
		}
		tournament.Format = format
	}
	if req.Status != nil {
		tournament.Status = domain.TournamentStatus(*req.Status)
	}
	if req.MaxParticipants != nil {
		tournament.MaxParticipants = req.MaxParticipants
	}
	if req.RegistrationOpen != nil {
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

	if err := h.repo.Update(r.Context(), tournament); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update tournament")
		return
	}

	// Fetch updated tournament
	updated, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated tournament")
		return
	}

	writeJSON(w, http.StatusOK, toTournamentResponse(updated))
}

// Delete deletes a tournament
func (h *TournamentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament id")
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

	// Only organizer can delete
	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only delete your own tournaments")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete tournament")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
