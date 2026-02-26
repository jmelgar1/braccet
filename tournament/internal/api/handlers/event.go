package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/braccet/tournament/internal/api/middleware"
	"github.com/braccet/tournament/internal/client"
	"github.com/braccet/tournament/internal/domain"
	"github.com/braccet/tournament/internal/repository"
	"github.com/go-chi/chi/v5"
)

type EventHandler struct {
	eventRepo       repository.EventRepository
	tournamentRepo  repository.TournamentRepository
	participantRepo repository.ParticipantRepository
	communityClient client.CommunityClient
}

func NewEventHandler(eventRepo repository.EventRepository, tournamentRepo repository.TournamentRepository, participantRepo repository.ParticipantRepository, communityClient client.CommunityClient) *EventHandler {
	return &EventHandler{
		eventRepo:       eventRepo,
		tournamentRepo:  tournamentRepo,
		participantRepo: participantRepo,
		communityClient: communityClient,
	}
}

// Request/Response types

type CreateEventRequest struct {
	CommunityID uint64  `json:"community_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Game        *string `json:"game,omitempty"`
	StartsAt    *string `json:"starts_at,omitempty"`
}

type UpdateEventRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Game        *string `json:"game,omitempty"`
	Status      *string `json:"status,omitempty"`
	StartsAt    *string `json:"starts_at,omitempty"`
}

type AddTournamentToEventRequest struct {
	TournamentID       uint64  `json:"tournament_id"`
	Role               string  `json:"role"` // "qualifier" or "main"
	AdvancingCount     *int    `json:"advancing_count,omitempty"`
	TargetTournamentID *uint64 `json:"target_tournament_id,omitempty"`
}

type UpdateEventTournamentRequest struct {
	Role               *string `json:"role,omitempty"`
	AdvancingCount     *int    `json:"advancing_count,omitempty"`
	TargetTournamentID *uint64 `json:"target_tournament_id,omitempty"`
	DisplayOrder       *int    `json:"display_order,omitempty"`
}

type EventResponse struct {
	ID          uint64                     `json:"id"`
	Slug        string                     `json:"slug"`
	CommunityID uint64                     `json:"community_id"`
	OrganizerID uint64                     `json:"organizer_id"`
	Name        string                     `json:"name"`
	Description *string                    `json:"description,omitempty"`
	Game        *string                    `json:"game,omitempty"`
	Status      string                     `json:"status"`
	StartsAt    *string                    `json:"starts_at,omitempty"`
	LogoURL     *string                    `json:"logo_url,omitempty"`
	Tournaments []EventTournamentResponse  `json:"tournaments,omitempty"`
	CreatedAt   string                     `json:"created_at"`
	UpdatedAt   string                     `json:"updated_at"`
}

type EventTournamentResponse struct {
	ID                 uint64  `json:"id"`
	TournamentID       uint64  `json:"tournament_id"`
	TournamentSlug     string  `json:"tournament_slug"`
	TournamentName     string  `json:"tournament_name"`
	TournamentStatus   string  `json:"tournament_status"`
	Role               string  `json:"role"`
	DisplayOrder       int     `json:"display_order"`
	AdvancingCount     *int    `json:"advancing_count,omitempty"`
	TargetTournamentID *uint64 `json:"target_tournament_id,omitempty"`
	ParticipantCount   int     `json:"participant_count"`
	AdvancedCount      int     `json:"advanced_count,omitempty"`
}

type AdvancementResponse struct {
	AdvancedCount int                           `json:"advanced_count"`
	Participants  []AdvancedParticipantResponse `json:"participants"`
}

type AdvancedParticipantResponse struct {
	SourceParticipantID uint64 `json:"source_participant_id"`
	TargetParticipantID uint64 `json:"target_participant_id"`
	DisplayName         string `json:"display_name"`
	Placement           int    `json:"placement"`
}

func toEventResponse(e *domain.Event) EventResponse {
	resp := EventResponse{
		ID:          e.ID,
		Slug:        e.Slug,
		CommunityID: e.CommunityID,
		OrganizerID: e.OrganizerID,
		Name:        e.Name,
		Description: e.Description,
		Game:        e.Game,
		Status:      string(e.Status),
		LogoURL:     e.LogoURL,
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   e.UpdatedAt.Format(time.RFC3339),
	}
	if e.StartsAt != nil {
		startsAt := e.StartsAt.Format(time.RFC3339)
		resp.StartsAt = &startsAt
	}
	return resp
}

// List returns all events for the authenticated user
func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	events, err := h.eventRepo.ListByOrganizer(r.Context(), userID)
	if err != nil {
		log.Printf("Error fetching events for user %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch events")
		return
	}

	response := make([]EventResponse, len(events))
	for i, e := range events {
		response[i] = toEventResponse(e)
	}

	writeJSON(w, http.StatusOK, response)
}

// ListByCommunity returns all events for a specific community
func (h *EventHandler) ListByCommunity(w http.ResponseWriter, r *http.Request) {
	communityIDStr := chi.URLParam(r, "communityId")
	communityID, err := strconv.ParseUint(communityIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid community ID")
		return
	}

	events, err := h.eventRepo.ListByCommunity(r.Context(), communityID)
	if err != nil {
		log.Printf("Error fetching events for community %d: %v", communityID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch events")
		return
	}

	response := make([]EventResponse, len(events))
	for i, e := range events {
		response[i] = toEventResponse(e)
	}

	writeJSON(w, http.StatusOK, response)
}

// Create creates a new event
func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.CommunityID == 0 {
		writeError(w, http.StatusBadRequest, "community_id is required")
		return
	}

	event := &domain.Event{
		Slug:        generateSlug(),
		CommunityID: req.CommunityID,
		OrganizerID: userID,
		Name:        req.Name,
		Description: req.Description,
		Game:        req.Game,
		Status:      domain.EventStatusDraft,
	}

	if req.StartsAt != nil {
		startsAt, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starts_at format (use RFC3339)")
			return
		}
		event.StartsAt = &startsAt
	}

	if err := h.eventRepo.Create(r.Context(), event); err != nil {
		log.Printf("Error creating event: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create event")
		return
	}

	writeJSON(w, http.StatusCreated, toEventResponse(event))
}

// Get returns a single event by slug with its tournaments
func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid event slug")
		return
	}

	event, err := h.eventRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	resp := toEventResponse(event)

	// Fetch tournaments linked to this event
	eventTournaments, err := h.eventRepo.GetTournamentsByEvent(r.Context(), event.ID)
	if err != nil {
		log.Printf("Error fetching tournaments for event %d: %v", event.ID, err)
	} else {
		resp.Tournaments = make([]EventTournamentResponse, 0, len(eventTournaments))
		for _, et := range eventTournaments {
			tournament, err := h.tournamentRepo.GetByID(r.Context(), et.TournamentID)
			if err != nil {
				log.Printf("Error fetching tournament %d: %v", et.TournamentID, err)
				continue
			}

			participantCount, _ := h.participantRepo.CountByTournament(r.Context(), et.TournamentID)

			// Count advanced participants (if this is a qualifier)
			var advancedCount int
			if et.Role == domain.EventRoleQualifier {
				advancements, _ := h.eventRepo.GetAdvancementsBySource(r.Context(), et.TournamentID)
				advancedCount = len(advancements)
			}

			resp.Tournaments = append(resp.Tournaments, EventTournamentResponse{
				ID:                 et.ID,
				TournamentID:       et.TournamentID,
				TournamentSlug:     tournament.Slug,
				TournamentName:     tournament.Name,
				TournamentStatus:   string(tournament.Status),
				Role:               string(et.Role),
				DisplayOrder:       et.DisplayOrder,
				AdvancingCount:     et.AdvancingCount,
				TargetTournamentID: et.TargetTournamentID,
				ParticipantCount:   participantCount,
				AdvancedCount:      advancedCount,
			})
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListAvailableTournaments returns tournaments from the same community that are not assigned to any event
func (h *EventHandler) ListAvailableTournaments(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid event slug")
		return
	}

	event, err := h.eventRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	tournaments, err := h.tournamentRepo.ListAvailableForEvent(r.Context(), event.CommunityID)
	if err != nil {
		log.Printf("Error fetching available tournaments: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch available tournaments")
		return
	}

	response := make([]TournamentResponse, len(tournaments))
	for i, t := range tournaments {
		response[i] = toTournamentResponse(t)
	}

	writeJSON(w, http.StatusOK, response)
}

// Update updates an event
func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid event slug")
		return
	}

	event, err := h.eventRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	// Only organizer can update
	if event.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only update your own events")
		return
	}

	var req UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Apply updates
	if req.Name != nil {
		event.Name = *req.Name
	}
	if req.Description != nil {
		event.Description = req.Description
	}
	if req.Game != nil {
		event.Game = req.Game
	}
	if req.Status != nil {
		event.Status = domain.EventStatus(*req.Status)
	}
	if req.StartsAt != nil {
		startsAt, err := time.Parse(time.RFC3339, *req.StartsAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid starts_at format (use RFC3339)")
			return
		}
		event.StartsAt = &startsAt
	}

	if err := h.eventRepo.Update(r.Context(), event); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update event")
		return
	}

	// Fetch updated event
	updated, err := h.eventRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated event")
		return
	}

	writeJSON(w, http.StatusOK, toEventResponse(updated))
}

// Delete deletes an event
func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid event slug")
		return
	}

	event, err := h.eventRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	// Only organizer can delete
	if event.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only delete your own events")
		return
	}

	if err := h.eventRepo.Delete(r.Context(), slug); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete event")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddTournament adds a tournament to an event
func (h *EventHandler) AddTournament(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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

	if event.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only modify your own events")
		return
	}

	var req AddTournamentToEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate tournament exists and belongs to same community
	tournament, err := h.tournamentRepo.GetByID(r.Context(), req.TournamentID)
	if err != nil {
		if errors.Is(err, repository.ErrTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch tournament")
		return
	}

	if tournament.CommunityID == nil || *tournament.CommunityID != event.CommunityID {
		writeError(w, http.StatusBadRequest, "tournament must belong to the same community as the event")
		return
	}

	role := domain.EventTournamentRole(req.Role)
	if role != domain.EventRoleQualifier && role != domain.EventRoleMain {
		writeError(w, http.StatusBadRequest, "role must be 'qualifier' or 'main'")
		return
	}

	// Determine display order and check for existing main event
	existingTournaments, _ := h.eventRepo.GetTournamentsByEvent(r.Context(), event.ID)
	displayOrder := len(existingTournaments)

	// Find if main event exists
	var mainEventTournament *domain.EventTournament
	for _, et := range existingTournaments {
		if et.Role == domain.EventRoleMain {
			mainEventTournament = et
			break
		}
	}

	// Enforce: must add main event first
	if role == domain.EventRoleQualifier && mainEventTournament == nil {
		writeError(w, http.StatusBadRequest, "a main event must be added before qualifiers")
		return
	}

	// Enforce: only one main event allowed
	if role == domain.EventRoleMain && mainEventTournament != nil {
		writeError(w, http.StatusBadRequest, "event already has a main event")
		return
	}

	// If adding a qualifier, sync ELO system with main event
	if role == domain.EventRoleQualifier && mainEventTournament != nil {
		mainTournament, err := h.tournamentRepo.GetByID(r.Context(), mainEventTournament.TournamentID)
		if err == nil && mainTournament.EloSystemID != nil {
			// Update the qualifier tournament to use the same ELO system
			if tournament.EloSystemID == nil || *tournament.EloSystemID != *mainTournament.EloSystemID {
				tournament.EloSystemID = mainTournament.EloSystemID
				if err := h.tournamentRepo.Update(r.Context(), tournament); err != nil {
					log.Printf("Warning: failed to sync ELO system for qualifier: %v", err)
				}
			}
		}
	}

	et := &domain.EventTournament{
		EventID:            event.ID,
		TournamentID:       req.TournamentID,
		Role:               role,
		DisplayOrder:       displayOrder,
		AdvancingCount:     req.AdvancingCount,
		TargetTournamentID: req.TargetTournamentID,
	}

	if err := h.eventRepo.AddTournament(r.Context(), et); err != nil {
		log.Printf("Error adding tournament to event: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to add tournament to event")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":             et.ID,
		"event_id":       et.EventID,
		"tournament_id":  et.TournamentID,
		"role":           string(et.Role),
		"display_order":  et.DisplayOrder,
		"advancing_count": et.AdvancingCount,
	})
}

// UpdateTournament updates an event tournament's settings
func (h *EventHandler) UpdateTournament(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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

	if event.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only modify your own events")
		return
	}

	tournamentIDStr := chi.URLParam(r, "tournamentId")
	tournamentID, err := strconv.ParseUint(tournamentIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	// Get the event tournament
	et, err := h.eventRepo.GetEventTournamentByTournamentID(r.Context(), tournamentID)
	if err != nil {
		if errors.Is(err, repository.ErrEventTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not in event")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch event tournament")
		return
	}

	if et.EventID != event.ID {
		writeError(w, http.StatusNotFound, "tournament not in this event")
		return
	}

	var req UpdateEventTournamentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Role != nil {
		role := domain.EventTournamentRole(*req.Role)
		if role != domain.EventRoleQualifier && role != domain.EventRoleMain {
			writeError(w, http.StatusBadRequest, "role must be 'qualifier' or 'main'")
			return
		}
		et.Role = role
	}
	if req.AdvancingCount != nil {
		et.AdvancingCount = req.AdvancingCount
	}
	if req.TargetTournamentID != nil {
		et.TargetTournamentID = req.TargetTournamentID
	}
	if req.DisplayOrder != nil {
		et.DisplayOrder = *req.DisplayOrder
	}

	if err := h.eventRepo.UpdateTournament(r.Context(), et); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update event tournament")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveTournament removes a tournament from an event
func (h *EventHandler) RemoveTournament(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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

	if event.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only modify your own events")
		return
	}

	tournamentIDStr := chi.URLParam(r, "tournamentId")
	tournamentID, err := strconv.ParseUint(tournamentIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	if err := h.eventRepo.RemoveTournament(r.Context(), event.ID, tournamentID); err != nil {
		if errors.Is(err, repository.ErrEventTournamentNotFound) {
			writeError(w, http.StatusNotFound, "tournament not in event")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to remove tournament from event")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetQualificationStatus returns an overview of all qualifiers and their advancement status
func (h *EventHandler) GetQualificationStatus(w http.ResponseWriter, r *http.Request) {
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

	eventTournaments, err := h.eventRepo.GetTournamentsByEvent(r.Context(), event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch event tournaments")
		return
	}

	type QualifierStatus struct {
		TournamentID    uint64   `json:"tournament_id"`
		TournamentName  string   `json:"tournament_name"`
		TournamentSlug  string   `json:"tournament_slug"`
		Status          string   `json:"status"`
		AdvancingCount  *int     `json:"advancing_count,omitempty"`
		AdvancedIDs     []uint64 `json:"advanced_ids,omitempty"`
	}

	type MainEventStatus struct {
		TournamentID       uint64 `json:"tournament_id"`
		TournamentName     string `json:"tournament_name"`
		TournamentSlug     string `json:"tournament_slug"`
		Status             string `json:"status"`
		RegisteredCount    int    `json:"registered_count"`
		ExpectedFromQuals  int    `json:"expected_from_quals"`
		ActualFromQuals    int    `json:"actual_from_quals"`
	}

	qualifiers := []QualifierStatus{}
	var mainEvent *MainEventStatus
	expectedFromQuals := 0

	for _, et := range eventTournaments {
		tournament, err := h.tournamentRepo.GetByID(r.Context(), et.TournamentID)
		if err != nil {
			continue
		}

		if et.Role == domain.EventRoleQualifier {
			qs := QualifierStatus{
				TournamentID:   et.TournamentID,
				TournamentName: tournament.Name,
				TournamentSlug: tournament.Slug,
				Status:         string(tournament.Status),
				AdvancingCount: et.AdvancingCount,
			}

			// Get advanced participant IDs
			advancements, _ := h.eventRepo.GetAdvancementsBySource(r.Context(), et.TournamentID)
			for _, adv := range advancements {
				qs.AdvancedIDs = append(qs.AdvancedIDs, adv.SourceParticipantID)
			}

			if et.AdvancingCount != nil {
				expectedFromQuals += *et.AdvancingCount
			}

			qualifiers = append(qualifiers, qs)
		} else if et.Role == domain.EventRoleMain {
			count, _ := h.participantRepo.CountByTournament(r.Context(), et.TournamentID)
			advancements, _ := h.eventRepo.GetAdvancementsByTarget(r.Context(), et.TournamentID)

			mainEvent = &MainEventStatus{
				TournamentID:      et.TournamentID,
				TournamentName:    tournament.Name,
				TournamentSlug:    tournament.Slug,
				Status:            string(tournament.Status),
				RegisteredCount:   count,
				ActualFromQuals:   len(advancements),
			}
		}
	}

	if mainEvent != nil {
		mainEvent.ExpectedFromQuals = expectedFromQuals
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"qualifiers": qualifiers,
		"main_event": mainEvent,
	})
}

// ProcessQualifierAdvancement handles advancing participants from a completed qualifier to the target tournament
// This is called automatically when a qualifier tournament completes, or can be triggered manually
// Seeding is based on ELO ratings from the target tournament's ELO system, NOT by qualification placement
func (h *EventHandler) ProcessQualifierAdvancement(ctx context.Context, tournamentID uint64) (*AdvancementResponse, error) {
	// Get event tournament info
	et, err := h.eventRepo.GetEventTournamentByTournamentID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Verify this is a qualifier with a target
	if et.Role != domain.EventRoleQualifier {
		return nil, errors.New("tournament is not a qualifier")
	}
	if et.TargetTournamentID == nil {
		return nil, errors.New("qualifier has no target tournament set")
	}
	if et.AdvancingCount == nil || *et.AdvancingCount <= 0 {
		return nil, errors.New("qualifier has no advancing count set")
	}

	// Verify source tournament is completed
	sourceTournament, err := h.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, err
	}
	if sourceTournament.Status != domain.StatusCompleted {
		return nil, errors.New("qualifier tournament is not completed")
	}

	// Get target tournament to get ELO system for seeding
	targetTournament, err := h.tournamentRepo.GetByID(ctx, *et.TargetTournamentID)
	if err != nil {
		return nil, err
	}

	// Get source participants ordered by seed (as a proxy for placement)
	// TODO: Get actual final standings from bracket service
	sourceParticipants, err := h.participantRepo.GetByTournament(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	// Sort by seed as proxy for placement (lower seed = better placement)
	advancingCount := *et.AdvancingCount
	if advancingCount > len(sourceParticipants) {
		advancingCount = len(sourceParticipants)
	}

	// Check for duplicates - don't advance if already in target (same community_member_id)
	targetParticipants, err := h.participantRepo.GetByTournament(ctx, *et.TargetTournamentID)
	if err != nil {
		return nil, err
	}
	existingMemberIDs := make(map[uint64]bool)
	for _, p := range targetParticipants {
		if p.CommunityMemberID != nil {
			existingMemberIDs[*p.CommunityMemberID] = true
		}
	}

	// Collect participants to advance (top N by current seed/placement)
	var toAdvance []*domain.Participant
	for i := 0; i < advancingCount && i < len(sourceParticipants); i++ {
		sourceP := sourceParticipants[i]

		// Skip if already in target tournament
		if sourceP.CommunityMemberID != nil && existingMemberIDs[*sourceP.CommunityMemberID] {
			log.Printf("Skipping participant %s - already in target tournament", sourceP.DisplayName)
			continue
		}
		toAdvance = append(toAdvance, sourceP)
	}

	// Get ELO ratings for all advancing participants
	var eloRatings map[uint64]int
	if targetTournament.EloSystemID != nil && h.communityClient != nil {
		memberIDs := make([]uint64, 0, len(toAdvance))
		for _, p := range toAdvance {
			if p.CommunityMemberID != nil {
				memberIDs = append(memberIDs, *p.CommunityMemberID)
			}
		}

		if len(memberIDs) > 0 {
			ratings, err := h.communityClient.GetBulkMemberRatings(ctx, *targetTournament.EloSystemID, memberIDs)
			if err != nil {
				log.Printf("Warning: failed to get ELO ratings for seeding: %v", err)
			} else {
				eloRatings = ratings
			}
		}
	}

	// Sort advancing participants by ELO rating (descending) for seeding
	if len(eloRatings) > 0 {
		sort.Slice(toAdvance, func(i, j int) bool {
			ratingI := 0
			ratingJ := 0
			if toAdvance[i].CommunityMemberID != nil {
				ratingI = eloRatings[*toAdvance[i].CommunityMemberID]
			}
			if toAdvance[j].CommunityMemberID != nil {
				ratingJ = eloRatings[*toAdvance[j].CommunityMemberID]
			}
			return ratingI > ratingJ // Higher rating = better seed
		})
		log.Printf("Sorted %d advancing participants by ELO rating", len(toAdvance))
	}

	response := &AdvancementResponse{
		Participants: []AdvancedParticipantResponse{},
	}

	// Create participants in target tournament with ELO-based seeds
	for i, sourceP := range toAdvance {
		// Create new participant in target tournament
		// Note: Seed will be assigned later when the tournament seeds are finalized
		// For now, we just register them
		targetP := &domain.Participant{
			TournamentID:      *et.TargetTournamentID,
			UserID:            sourceP.UserID,
			CommunityMemberID: sourceP.CommunityMemberID,
			DisplayName:       sourceP.DisplayName,
			Status:            domain.ParticipantRegistered,
		}

		if err := h.participantRepo.Create(ctx, targetP); err != nil {
			log.Printf("Error creating participant in target tournament: %v", err)
			continue
		}

		// Record advancement with ELO-based order (not source placement)
		finalPlacement := i + 1 // ELO-based order
		if sourceP.Seed != nil {
			finalPlacement = int(*sourceP.Seed) // Store original seed from qualifier
		}
		adv := &domain.EventAdvancement{
			EventID:             et.EventID,
			SourceTournamentID:  tournamentID,
			SourceParticipantID: sourceP.ID,
			TargetTournamentID:  *et.TargetTournamentID,
			TargetParticipantID: targetP.ID,
			FinalPlacement:      finalPlacement,
		}

		if err := h.eventRepo.CreateAdvancement(ctx, adv); err != nil {
			log.Printf("Error recording advancement: %v", err)
			continue
		}

		eloRating := 0
		if sourceP.CommunityMemberID != nil && eloRatings != nil {
			eloRating = eloRatings[*sourceP.CommunityMemberID]
		}

		response.AdvancedCount++
		response.Participants = append(response.Participants, AdvancedParticipantResponse{
			SourceParticipantID: sourceP.ID,
			TargetParticipantID: targetP.ID,
			DisplayName:         sourceP.DisplayName,
			Placement:           i + 1, // ELO-based rank
		})

		log.Printf("Advanced %s to target tournament (ELO rank: %d, rating: %d)", sourceP.DisplayName, i+1, eloRating)
	}

	return response, nil
}

// TriggerAdvancement manually triggers advancement for a qualifier
func (h *EventHandler) TriggerAdvancement(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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

	if event.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only modify your own events")
		return
	}

	tournamentIDStr := chi.URLParam(r, "tournamentId")
	tournamentID, err := strconv.ParseUint(tournamentIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	result, err := h.ProcessQualifierAdvancement(r.Context(), tournamentID)
	if err != nil {
		log.Printf("Error processing qualifier advancement: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// RollbackAdvancement removes advanced participants when a qualifier is reset
func (h *EventHandler) RollbackAdvancement(ctx context.Context, tournamentID uint64) error {
	// Get advancement records for this qualifier
	advancements, err := h.eventRepo.GetAdvancementsBySource(ctx, tournamentID)
	if err != nil {
		return err
	}

	// Delete participants from target tournament and remove advancement records
	for _, adv := range advancements {
		// Delete participant from target tournament
		if err := h.participantRepo.Delete(ctx, adv.TargetParticipantID); err != nil {
			log.Printf("Warning: failed to delete participant %d: %v", adv.TargetParticipantID, err)
		}
	}

	// Delete all advancement records for this source tournament
	if err := h.eventRepo.DeleteAdvancementsBySource(ctx, tournamentID); err != nil {
		return err
	}

	log.Printf("Rolled back %d advancements from tournament %d", len(advancements), tournamentID)
	return nil
}

// Graph-related types for DAG editor

type GraphNode struct {
	TournamentID     uint64 `json:"tournament_id"`
	TournamentSlug   string `json:"tournament_slug"`
	TournamentName   string `json:"tournament_name"`
	Role             string `json:"role"`
	Status           string `json:"status"`
	ParticipantCount int    `json:"participant_count"`
	AdvancedCount    int    `json:"advanced_count"`
	AdvancingCount   *int   `json:"advancing_count,omitempty"`
	PositionX        int    `json:"position_x"`
	PositionY        int    `json:"position_y"`
}

type GraphEdge struct {
	SourceTournamentID uint64 `json:"source_tournament_id"`
	TargetTournamentID uint64 `json:"target_tournament_id"`
	AdvancingCount     int    `json:"advancing_count"`
}

type EventGraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type ConnectionUpdate struct {
	TournamentID       uint64  `json:"tournament_id"`
	TargetTournamentID *uint64 `json:"target_tournament_id"`
	AdvancingCount     *int    `json:"advancing_count"`
	PositionX          int     `json:"position_x"`
	PositionY          int     `json:"position_y"`
}

type UpdateConnectionsRequest struct {
	Connections []ConnectionUpdate `json:"connections"`
}

// GetGraph returns the tournament DAG structure for the visual editor
func (h *EventHandler) GetGraph(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid event slug")
		return
	}

	event, err := h.eventRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch event")
		return
	}

	eventTournaments, err := h.eventRepo.GetTournamentsByEvent(r.Context(), event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch event tournaments")
		return
	}

	nodes := make([]GraphNode, 0, len(eventTournaments))
	edges := make([]GraphEdge, 0)

	for _, et := range eventTournaments {
		tournament, err := h.tournamentRepo.GetByID(r.Context(), et.TournamentID)
		if err != nil {
			log.Printf("Error fetching tournament %d: %v", et.TournamentID, err)
			continue
		}

		participantCount, _ := h.participantRepo.CountByTournament(r.Context(), et.TournamentID)

		var advancedCount int
		if et.Role == domain.EventRoleQualifier {
			advancements, _ := h.eventRepo.GetAdvancementsBySource(r.Context(), et.TournamentID)
			advancedCount = len(advancements)
		}

		nodes = append(nodes, GraphNode{
			TournamentID:     et.TournamentID,
			TournamentSlug:   tournament.Slug,
			TournamentName:   tournament.Name,
			Role:             string(et.Role),
			Status:           string(tournament.Status),
			ParticipantCount: participantCount,
			AdvancedCount:    advancedCount,
			AdvancingCount:   et.AdvancingCount,
			PositionX:        et.PositionX,
			PositionY:        et.PositionY,
		})

		// Create edge if this tournament has a target
		if et.TargetTournamentID != nil && et.AdvancingCount != nil {
			edges = append(edges, GraphEdge{
				SourceTournamentID: et.TournamentID,
				TargetTournamentID: *et.TargetTournamentID,
				AdvancingCount:     *et.AdvancingCount,
			})
		}
	}

	writeJSON(w, http.StatusOK, EventGraphResponse{
		Nodes: nodes,
		Edges: edges,
	})
}

// UpdateConnections batch updates tournament connections and positions
func (h *EventHandler) UpdateConnections(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

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

	if event.OrganizerID != userID {
		writeError(w, http.StatusForbidden, "you can only modify your own events")
		return
	}

	var req UpdateConnectionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get existing event tournaments to validate
	eventTournaments, err := h.eventRepo.GetTournamentsByEvent(r.Context(), event.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch event tournaments")
		return
	}

	// Build lookup map of valid tournament IDs in this event
	validTournamentIDs := make(map[uint64]*domain.EventTournament)
	for _, et := range eventTournaments {
		validTournamentIDs[et.TournamentID] = et
	}

	// Validate DAG structure (no cycles, valid targets)
	if err := h.validateDAG(req.Connections, validTournamentIDs); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Apply updates
	for _, conn := range req.Connections {
		et, exists := validTournamentIDs[conn.TournamentID]
		if !exists {
			continue // Skip tournaments not in this event
		}

		// Update the event tournament
		et.TargetTournamentID = conn.TargetTournamentID
		et.AdvancingCount = conn.AdvancingCount
		et.PositionX = conn.PositionX
		et.PositionY = conn.PositionY

		if err := h.eventRepo.UpdateTournament(r.Context(), et); err != nil {
			log.Printf("Error updating tournament connection %d: %v", conn.TournamentID, err)
			writeError(w, http.StatusInternalServerError, "failed to update connections")
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateDAG checks for cycles and invalid connections
func (h *EventHandler) validateDAG(connections []ConnectionUpdate, validTournaments map[uint64]*domain.EventTournament) error {
	// Build adjacency list
	graph := make(map[uint64][]uint64)
	for _, conn := range connections {
		if conn.TargetTournamentID != nil {
			// Verify target is in this event
			if _, exists := validTournaments[*conn.TargetTournamentID]; !exists {
				return errors.New("target tournament is not part of this event")
			}
			// Verify not pointing to itself
			if conn.TournamentID == *conn.TargetTournamentID {
				return errors.New("tournament cannot advance to itself")
			}
			// Verify main event doesn't have outgoing connection
			if et := validTournaments[conn.TournamentID]; et != nil && et.Role == domain.EventRoleMain {
				return errors.New("main event cannot have outgoing connections")
			}
			graph[conn.TournamentID] = append(graph[conn.TournamentID], *conn.TargetTournamentID)
		}
	}

	// DFS cycle detection
	visited := make(map[uint64]bool)
	recStack := make(map[uint64]bool)

	var hasCycle func(node uint64) bool
	hasCycle = func(node uint64) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for node := range graph {
		if !visited[node] && hasCycle(node) {
			return errors.New("cycle detected in tournament connections")
		}
	}

	return nil
}

// TriggerAdvancementInternal is the internal endpoint for service-to-service advancement triggers
// Called by bracket service when a tournament completes
func (h *EventHandler) TriggerAdvancementInternal(w http.ResponseWriter, r *http.Request) {
	tournamentIDStr := chi.URLParam(r, "id")
	tournamentID, err := strconv.ParseUint(tournamentIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	// Check if tournament is part of an event as a qualifier
	et, err := h.eventRepo.GetEventTournamentByTournamentID(r.Context(), tournamentID)
	if err != nil {
		if errors.Is(err, repository.ErrEventTournamentNotFound) {
			// Tournament not part of an event, just return 200 OK
			w.WriteHeader(http.StatusOK)
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to check event tournament")
		return
	}

	// Only process if this is a qualifier with a target
	if et.Role != domain.EventRoleQualifier || et.TargetTournamentID == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	result, err := h.ProcessQualifierAdvancement(r.Context(), tournamentID)
	if err != nil {
		log.Printf("Auto-advancement failed for tournament %d: %v", tournamentID, err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	log.Printf("Auto-advancement completed for tournament %d: %d participants advanced", tournamentID, result.AdvancedCount)
	writeJSON(w, http.StatusOK, result)
}
