package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/braccet/tournament/internal/api/middleware"
	"github.com/braccet/tournament/internal/repository"
	"github.com/braccet/tournament/internal/service"
	"github.com/go-chi/chi/v5"
)

type UploadHandler struct {
	storageService service.StorageService
	eventRepo      repository.EventRepository
	tournamentRepo repository.TournamentRepository
}

func NewUploadHandler(
	storageService service.StorageService,
	eventRepo repository.EventRepository,
	tournamentRepo repository.TournamentRepository,
) *UploadHandler {
	return &UploadHandler{
		storageService: storageService,
		eventRepo:      eventRepo,
		tournamentRepo: tournamentRepo,
	}
}

type GetUploadURLRequest struct {
	ContentType string `json:"content_type"`
}

// GetEventLogoUploadURL generates a presigned URL for uploading an event logo
// POST /events/{slug}/logo/upload-url
func (h *UploadHandler) GetEventLogoUploadURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get event by slug
	slug := chi.URLParam(r, "slug")
	event, err := h.eventRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	// Check if user is organizer
	if event.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can upload event logos")
		return
	}

	// Parse request
	var req GetUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ContentType == "" {
		writeError(w, http.StatusBadRequest, "content_type is required")
		return
	}

	// Validate content type
	if !h.storageService.ValidateContentType(req.ContentType) {
		writeError(w, http.StatusBadRequest, "invalid content type: must be image/jpeg, image/png, image/webp, or image/svg+xml")
		return
	}

	// Generate presigned URL
	response, err := h.storageService.GenerateEventLogoUploadURL(r.Context(), event.ID, req.ContentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate upload URL")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// UpdateEventLogoURL updates the event's logo_url after upload
// PUT /events/{slug}/logo
func (h *UploadHandler) UpdateEventLogoURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get event by slug
	slug := chi.URLParam(r, "slug")
	event, err := h.eventRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	// Check if user is organizer
	if event.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can update event logos")
		return
	}

	// Parse request
	var req struct {
		LogoURL string `json:"logo_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.LogoURL == "" {
		writeError(w, http.StatusBadRequest, "logo_url is required")
		return
	}

	// Update event logo URL
	event.LogoURL = &req.LogoURL
	if err := h.eventRepo.Update(r.Context(), event); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update event logo")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"logo_url": req.LogoURL})
}

// GetTournamentLogoUploadURL generates a presigned URL for uploading a tournament logo
// POST /tournaments/{slug}/logo/upload-url
func (h *UploadHandler) GetTournamentLogoUploadURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get tournament by slug
	slug := chi.URLParam(r, "slug")
	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	// Check if user is organizer
	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can upload tournament logos")
		return
	}

	// Parse request
	var req GetUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ContentType == "" {
		writeError(w, http.StatusBadRequest, "content_type is required")
		return
	}

	// Validate content type
	if !h.storageService.ValidateContentType(req.ContentType) {
		writeError(w, http.StatusBadRequest, "invalid content type: must be image/jpeg, image/png, image/webp, or image/svg+xml")
		return
	}

	// Generate presigned URL
	response, err := h.storageService.GenerateTournamentLogoUploadURL(r.Context(), tournament.ID, req.ContentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate upload URL")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// UpdateTournamentLogoURL updates the tournament's logo_url after upload
// PUT /tournaments/{slug}/logo
func (h *UploadHandler) UpdateTournamentLogoURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get tournament by slug
	slug := chi.URLParam(r, "slug")
	tournament, err := h.tournamentRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	// Check if user is organizer
	if tournament.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "only the organizer can update tournament logos")
		return
	}

	// Parse request
	var req struct {
		LogoURL string `json:"logo_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.LogoURL == "" {
		writeError(w, http.StatusBadRequest, "logo_url is required")
		return
	}

	// Update tournament logo URL
	tournament.LogoURL = &req.LogoURL
	if err := h.tournamentRepo.Update(r.Context(), tournament); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update tournament logo")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"logo_url": req.LogoURL})
}
