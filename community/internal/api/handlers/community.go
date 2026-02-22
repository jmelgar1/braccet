package handlers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/braccet/community/internal/api/middleware"
	"github.com/braccet/community/internal/domain"
	"github.com/braccet/community/internal/repository"
	"github.com/go-chi/chi/v5"
)

type CommunityHandler struct {
	communityRepo repository.CommunityRepository
	memberRepo    repository.MemberRepository
}

func NewCommunityHandler(communityRepo repository.CommunityRepository, memberRepo repository.MemberRepository) *CommunityHandler {
	return &CommunityHandler{
		communityRepo: communityRepo,
		memberRepo:    memberRepo,
	}
}

// Request/Response types

type CreateCommunityRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Game        *string `json:"game,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

type UpdateCommunityRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Game        *string `json:"game,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

type CommunityResponse struct {
	ID          uint64  `json:"id"`
	Slug        string  `json:"slug"`
	OwnerID     uint64  `json:"owner_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Game        *string `json:"game,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
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

func toCommunityResponse(c *domain.Community) CommunityResponse {
	return CommunityResponse{
		ID:          c.ID,
		Slug:        c.Slug,
		OwnerID:     c.OwnerID,
		Name:        c.Name,
		Description: c.Description,
		Game:        c.Game,
		AvatarURL:   c.AvatarURL,
		CreatedAt:   c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   c.UpdatedAt.Format(time.RFC3339),
	}
}

// List returns all communities for the authenticated user
func (h *CommunityHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	communities, err := h.communityRepo.ListByOwner(r.Context(), userID)
	if err != nil {
		log.Printf("Error fetching communities for user %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch communities")
		return
	}

	response := make([]CommunityResponse, len(communities))
	for i, c := range communities {
		response[i] = toCommunityResponse(c)
	}

	writeJSON(w, http.StatusOK, response)
}

// Create creates a new community
func (h *CommunityHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateCommunityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	community := &domain.Community{
		Slug:        generateSlug(),
		OwnerID:     userID,
		Name:        req.Name,
		Description: req.Description,
		Game:        req.Game,
		AvatarURL:   req.AvatarURL,
		Settings:    json.RawMessage(`{}`),
	}

	if err := h.communityRepo.Create(r.Context(), community); err != nil {
		log.Printf("Error creating community: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create community")
		return
	}

	// Add the owner as a member with owner role
	owner := &domain.CommunityMember{
		CommunityID: community.ID,
		UserID:      &userID,
		DisplayName: "Owner", // Will be updated with actual display name from auth service
		Role:        domain.RoleOwner,
	}
	if err := h.memberRepo.Create(r.Context(), owner); err != nil {
		log.Printf("Error adding owner as member: %v", err)
		// Don't fail the creation, but log the error
	}

	writeJSON(w, http.StatusCreated, toCommunityResponse(community))
}

// Get returns a single community by slug
func (h *CommunityHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid community slug")
		return
	}

	community, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	writeJSON(w, http.StatusOK, toCommunityResponse(community))
}

// GetByID returns a single community by ID (internal endpoint for service-to-service calls)
func (h *CommunityHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid community ID")
		return
	}

	community, err := h.communityRepo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	writeJSON(w, http.StatusOK, toCommunityResponse(community))
}

// Update updates a community
func (h *CommunityHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid community slug")
		return
	}

	community, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	// Only owner can update
	if community.OwnerID != userID {
		writeError(w, http.StatusForbidden, "you can only update your own communities")
		return
	}

	var req UpdateCommunityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Apply updates
	if req.Name != nil {
		community.Name = *req.Name
	}
	if req.Description != nil {
		community.Description = req.Description
	}
	if req.Game != nil {
		community.Game = req.Game
	}
	if req.AvatarURL != nil {
		community.AvatarURL = req.AvatarURL
	}

	if err := h.communityRepo.Update(r.Context(), community); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update community")
		return
	}

	// Fetch updated community
	updated, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated community")
		return
	}

	writeJSON(w, http.StatusOK, toCommunityResponse(updated))
}

// Delete deletes a community
func (h *CommunityHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid community slug")
		return
	}

	community, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	// Only owner can delete
	if community.OwnerID != userID {
		writeError(w, http.StatusForbidden, "you can only delete your own communities")
		return
	}

	if err := h.communityRepo.Delete(r.Context(), community.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete community")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
