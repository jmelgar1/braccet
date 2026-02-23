package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/braccet/community/internal/api/middleware"
	"github.com/braccet/community/internal/domain"
	"github.com/braccet/community/internal/repository"
	"github.com/go-chi/chi/v5"
)

type MemberHandler struct {
	memberRepo    repository.MemberRepository
	communityRepo repository.CommunityRepository
}

func NewMemberHandler(memberRepo repository.MemberRepository, communityRepo repository.CommunityRepository) *MemberHandler {
	return &MemberHandler{
		memberRepo:    memberRepo,
		communityRepo: communityRepo,
	}
}

// Request/Response types

type AddMemberRequest struct {
	UserID      *uint64 `json:"user_id,omitempty"` // NULL for ghost members
	DisplayName string  `json:"display_name"`
	Role        *string `json:"role,omitempty"` // Defaults to "member"
}

type UpdateMemberRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	IconURL     *string `json:"icon_url,omitempty"`
	Region      *string `json:"region,omitempty"`
}

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

type MemberResponse struct {
	ID            uint64  `json:"id"`
	CommunityID   uint64  `json:"community_id"`
	UserID        *uint64 `json:"user_id,omitempty"`
	DisplayName   string  `json:"display_name"`
	Role          string  `json:"role"`
	IsGhost       bool    `json:"is_ghost"`
	IconURL       *string `json:"icon_url,omitempty"`
	Region        *string `json:"region,omitempty"`
	EloRating     *int    `json:"elo_rating,omitempty"`
	MatchesPlayed int     `json:"matches_played"`
	MatchesWon    int     `json:"matches_won"`
	JoinedAt      string  `json:"joined_at"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

func toMemberResponse(m *domain.CommunityMember) MemberResponse {
	return MemberResponse{
		ID:            m.ID,
		CommunityID:   m.CommunityID,
		UserID:        m.UserID,
		DisplayName:   m.DisplayName,
		Role:          string(m.Role),
		IsGhost:       m.IsGhost(),
		IconURL:       m.IconURL,
		Region:        m.Region,
		EloRating:     m.EloRating,
		MatchesPlayed: m.MatchesPlayed,
		MatchesWon:    m.MatchesWon,
		JoinedAt:      m.JoinedAt.Format(time.RFC3339),
		CreatedAt:     m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     m.UpdatedAt.Format(time.RFC3339),
	}
}

// getCommunityFromSlug is a helper to get community by slug from URL
func (h *MemberHandler) getCommunityFromSlug(r *http.Request) (*domain.Community, error) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		return nil, errors.New("invalid community slug")
	}
	return h.communityRepo.GetBySlug(r.Context(), slug)
}

// isOwnerOrAdmin checks if the user is an owner or admin of the community
func (h *MemberHandler) isOwnerOrAdmin(r *http.Request, community *domain.Community, userID uint64) bool {
	// Check if user is owner
	if community.OwnerID == userID {
		return true
	}

	// Check if user is admin
	member, err := h.memberRepo.GetByCommunityAndUser(r.Context(), community.ID, userID)
	if err != nil {
		return false
	}
	return member.Role == domain.RoleOwner || member.Role == domain.RoleAdmin
}

// List returns all members of a community
func (h *MemberHandler) List(w http.ResponseWriter, r *http.Request) {
	community, err := h.getCommunityFromSlug(r)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	members, err := h.memberRepo.GetByCommunity(r.Context(), community.ID)
	if err != nil {
		log.Printf("Error fetching members for community %d: %v", community.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch members")
		return
	}

	response := make([]MemberResponse, len(members))
	for i, m := range members {
		response[i] = toMemberResponse(m)
	}

	writeJSON(w, http.StatusOK, response)
}

// Add adds a new member to the community (user or ghost)
func (h *MemberHandler) Add(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	community, err := h.getCommunityFromSlug(r)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	// Only owner or admin can add members
	if !h.isOwnerOrAdmin(r, community, userID) {
		writeError(w, http.StatusForbidden, "only owner or admin can add members")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}

	// If adding a real user, check they're not already a member
	if req.UserID != nil {
		existing, err := h.memberRepo.GetByCommunityAndUser(r.Context(), community.ID, *req.UserID)
		if err == nil && existing != nil {
			writeError(w, http.StatusConflict, "user is already a member of this community")
			return
		}
	}

	role := domain.RoleMember
	if req.Role != nil {
		r := domain.MemberRole(*req.Role)
		if r != domain.RoleMember && r != domain.RoleAdmin {
			writeError(w, http.StatusBadRequest, "role must be 'member' or 'admin'")
			return
		}
		role = r
	}

	member := &domain.CommunityMember{
		CommunityID: community.ID,
		UserID:      req.UserID,
		DisplayName: req.DisplayName,
		Role:        role,
	}

	if err := h.memberRepo.Create(r.Context(), member); err != nil {
		log.Printf("Error adding member: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	writeJSON(w, http.StatusCreated, toMemberResponse(member))
}

// Get returns a single member by ID
func (h *MemberHandler) Get(w http.ResponseWriter, r *http.Request) {
	memberIDStr := chi.URLParam(r, "memberId")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member ID")
		return
	}

	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch member")
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(member))
}

// GetByID returns a single member by ID (internal endpoint for service-to-service calls)
func (h *MemberHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	memberIDStr := chi.URLParam(r, "memberId")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member ID")
		return
	}

	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch member")
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(member))
}

// Update updates a member's display name, icon, and/or region
func (h *MemberHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	community, err := h.getCommunityFromSlug(r)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	memberIDStr := chi.URLParam(r, "memberId")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member ID")
		return
	}

	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch member")
		return
	}

	// Verify member belongs to this community
	if member.CommunityID != community.ID {
		writeError(w, http.StatusNotFound, "member not found in this community")
		return
	}

	var req UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Determine permissions
	isGhost := member.IsGhost()
	isSelf := member.UserID != nil && *member.UserID == userID
	isAdmin := h.isOwnerOrAdmin(r, community, userID)

	// DisplayName and IconURL can only be edited on ghost members by admins
	if req.DisplayName != nil || req.IconURL != nil {
		if !isGhost {
			writeError(w, http.StatusForbidden, "only ghost members can have name/icon edited")
			return
		}
		if !isAdmin {
			writeError(w, http.StatusForbidden, "only owner or admin can edit ghost members")
			return
		}
	}

	// Region can be edited by: admins (for any member) or users (for themselves)
	if req.Region != nil {
		if !isAdmin && !isSelf {
			writeError(w, http.StatusForbidden, "cannot edit another member's region")
			return
		}
		if !isValidRegionFormat(*req.Region) {
			writeError(w, http.StatusBadRequest, "region must be exactly 2 uppercase letters")
			return
		}
	}

	// Apply updates
	if req.DisplayName != nil {
		member.DisplayName = *req.DisplayName
	}
	if req.IconURL != nil {
		member.IconURL = req.IconURL
	}
	if req.Region != nil {
		if *req.Region == "" {
			member.Region = nil
		} else {
			member.Region = req.Region
		}
	}

	if err := h.memberRepo.Update(r.Context(), member); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update member")
		return
	}

	// Fetch updated member
	updated, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated member")
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(updated))
}

// UpdateRole updates a member's role
func (h *MemberHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	community, err := h.getCommunityFromSlug(r)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	// Only owner can change roles
	if community.OwnerID != userID {
		writeError(w, http.StatusForbidden, "only owner can change member roles")
		return
	}

	memberIDStr := chi.URLParam(r, "memberId")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member ID")
		return
	}

	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch member")
		return
	}

	// Verify member belongs to this community
	if member.CommunityID != community.ID {
		writeError(w, http.StatusNotFound, "member not found in this community")
		return
	}

	// Can't change owner's role
	if member.Role == domain.RoleOwner {
		writeError(w, http.StatusBadRequest, "cannot change owner's role")
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role := domain.MemberRole(req.Role)
	if role != domain.RoleMember && role != domain.RoleAdmin {
		writeError(w, http.StatusBadRequest, "role must be 'member' or 'admin'")
		return
	}

	if err := h.memberRepo.UpdateRole(r.Context(), memberID, role); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update role")
		return
	}

	// Fetch updated member
	updated, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch updated member")
		return
	}

	writeJSON(w, http.StatusOK, toMemberResponse(updated))
}

// Remove removes a member from the community
func (h *MemberHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	community, err := h.getCommunityFromSlug(r)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	// Only owner or admin can remove members
	if !h.isOwnerOrAdmin(r, community, userID) {
		writeError(w, http.StatusForbidden, "only owner or admin can remove members")
		return
	}

	memberIDStr := chi.URLParam(r, "memberId")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid member ID")
		return
	}

	member, err := h.memberRepo.GetByID(r.Context(), memberID)
	if err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "member not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch member")
		return
	}

	// Verify member belongs to this community
	if member.CommunityID != community.ID {
		writeError(w, http.StatusNotFound, "member not found in this community")
		return
	}

	// Can't remove owner
	if member.Role == domain.RoleOwner {
		writeError(w, http.StatusBadRequest, "cannot remove owner from community")
		return
	}

	if err := h.memberRepo.Delete(r.Context(), memberID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListGhosts returns all ghost members of a community
func (h *MemberHandler) ListGhosts(w http.ResponseWriter, r *http.Request) {
	community, err := h.getCommunityFromSlug(r)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	members, err := h.memberRepo.ListGhostMembers(r.Context(), community.ID)
	if err != nil {
		log.Printf("Error fetching ghost members for community %d: %v", community.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch ghost members")
		return
	}

	response := make([]MemberResponse, len(members))
	for i, m := range members {
		response[i] = toMemberResponse(m)
	}

	writeJSON(w, http.StatusOK, response)
}

// CreateGhostMember creates a ghost member for internal service-to-service calls
func (h *MemberHandler) CreateGhostMember(w http.ResponseWriter, r *http.Request) {
	communityIDStr := chi.URLParam(r, "id")
	communityID, err := strconv.ParseUint(communityIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid community ID")
		return
	}

	community, err := h.communityRepo.GetByID(r.Context(), communityID)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}

	// Force ghost member (no user_id)
	member := &domain.CommunityMember{
		CommunityID: community.ID,
		UserID:      nil,
		DisplayName: req.DisplayName,
		Role:        domain.RoleMember,
	}

	if err := h.memberRepo.Create(r.Context(), member); err != nil {
		log.Printf("Error creating ghost member: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create ghost member")
		return
	}

	writeJSON(w, http.StatusCreated, toMemberResponse(member))
}

// FindOrCreateGhostMember finds an existing member by display_name or creates a new ghost member
// POST /internal/communities/{id}/members/find-or-create
func (h *MemberHandler) FindOrCreateGhostMember(w http.ResponseWriter, r *http.Request) {
	communityIDStr := chi.URLParam(r, "id")
	communityID, err := strconv.ParseUint(communityIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid community ID")
		return
	}

	community, err := h.communityRepo.GetByID(r.Context(), communityID)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "display_name is required")
		return
	}

	// Try to find existing member by display_name
	existing, err := h.memberRepo.GetByCommunityAndDisplayName(r.Context(), community.ID, req.DisplayName)
	if err == nil {
		// Found existing member - return it
		writeJSON(w, http.StatusOK, toMemberResponse(existing))
		return
	}

	if !errors.Is(err, repository.ErrMemberNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to search for member")
		return
	}

	// Not found - create new ghost member
	member := &domain.CommunityMember{
		CommunityID: community.ID,
		UserID:      nil,
		DisplayName: req.DisplayName,
		Role:        domain.RoleMember,
	}

	if err := h.memberRepo.Create(r.Context(), member); err != nil {
		log.Printf("Error creating ghost member: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create ghost member")
		return
	}

	writeJSON(w, http.StatusCreated, toMemberResponse(member))
}

// DeleteMembersInternal handles internal service-to-service bulk delete requests
// POST /internal/communities/{id}/members/delete
func (h *MemberHandler) DeleteMembersInternal(w http.ResponseWriter, r *http.Request) {
	communityIDStr := chi.URLParam(r, "id")
	communityID, err := strconv.ParseUint(communityIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid community ID")
		return
	}

	// Verify community exists
	_, err = h.communityRepo.GetByID(r.Context(), communityID)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	var req struct {
		MemberIDs []uint64 `json:"member_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.MemberIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]int64{"deleted": 0})
		return
	}

	deleted, err := h.memberRepo.DeleteMany(r.Context(), req.MemberIDs)
	if err != nil {
		log.Printf("Error deleting members: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete members")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"deleted": deleted})
}

// SearchMembersRequest for internal search endpoint
type SearchMembersRequest struct {
	Query      string   `json:"query"`
	ExcludeIDs []uint64 `json:"exclude_ids,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

// SearchMemberResponse is a simplified response for autocomplete
type SearchMemberResponse struct {
	ID          uint64 `json:"id"`
	DisplayName string `json:"display_name"`
}

// SearchMembers handles internal search requests for autocomplete
// POST /internal/communities/{id}/members/search
func (h *MemberHandler) SearchMembers(w http.ResponseWriter, r *http.Request) {
	communityIDStr := chi.URLParam(r, "id")
	communityID, err := strconv.ParseUint(communityIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid community ID")
		return
	}

	var req SearchMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Default limit
	limit := 10
	if req.Limit > 0 && req.Limit <= 50 {
		limit = req.Limit
	}

	// Fetch extra results to account for exclusions
	fetchLimit := limit + len(req.ExcludeIDs)
	members, err := h.memberRepo.SearchByDisplayNamePrefix(r.Context(), communityID, req.Query, fetchLimit)
	if err != nil {
		log.Printf("Error searching members: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to search members")
		return
	}

	// Build exclude set for O(1) lookup
	excludeSet := make(map[uint64]bool, len(req.ExcludeIDs))
	for _, id := range req.ExcludeIDs {
		excludeSet[id] = true
	}

	// Filter and convert to response
	response := make([]SearchMemberResponse, 0, limit)
	for _, m := range members {
		if excludeSet[m.ID] {
			continue
		}
		response = append(response, SearchMemberResponse{
			ID:          m.ID,
			DisplayName: m.DisplayName,
		})
		if len(response) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// GetBulkMemberIcons returns icon URLs for multiple members
// POST /internal/members/bulk-icons
func (h *MemberHandler) GetBulkMemberIcons(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MemberIDs []uint64 `json:"member_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.MemberIDs) == 0 {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	icons, err := h.memberRepo.GetBulkIconURLs(r.Context(), req.MemberIDs)
	if err != nil {
		log.Printf("Error fetching member icons: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch member icons")
		return
	}

	// Return as array of {member_id, icon_url}
	type IconResult struct {
		MemberID uint64 `json:"member_id"`
		IconURL  string `json:"icon_url"`
	}

	response := make([]IconResult, 0, len(icons))
	for memberID, iconURL := range icons {
		response = append(response, IconResult{
			MemberID: memberID,
			IconURL:  iconURL,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// GetBulkMemberData returns icon URLs and regions for multiple members
// POST /internal/members/bulk-data
func (h *MemberHandler) GetBulkMemberData(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MemberIDs []uint64 `json:"member_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.MemberIDs) == 0 {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	data, err := h.memberRepo.GetBulkMemberData(r.Context(), req.MemberIDs)
	if err != nil {
		log.Printf("Error fetching member data: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch member data")
		return
	}

	// Return as array of {member_id, icon_url, region}
	type DataResult struct {
		MemberID uint64  `json:"member_id"`
		IconURL  *string `json:"icon_url,omitempty"`
		Region   *string `json:"region,omitempty"`
	}

	response := make([]DataResult, 0, len(data))
	for memberID, d := range data {
		response = append(response, DataResult{
			MemberID: memberID,
			IconURL:  d.IconURL,
			Region:   d.Region,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// Leaderboard returns the ELO leaderboard for a community
func (h *MemberHandler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	community, err := h.getCommunityFromSlug(r)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	// Default limit of 50
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	members, err := h.memberRepo.GetLeaderboard(r.Context(), community.ID, limit)
	if err != nil {
		log.Printf("Error fetching leaderboard for community %d: %v", community.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch leaderboard")
		return
	}

	response := make([]MemberResponse, len(members))
	for i, m := range members {
		response[i] = toMemberResponse(m)
	}

	writeJSON(w, http.StatusOK, response)
}

// isValidRegionFormat validates that a region code is exactly 2 uppercase letters
func isValidRegionFormat(region string) bool {
	if region == "" {
		return true // Allow clearing
	}
	if len(region) != 2 {
		return false
	}
	for _, c := range region {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

// GetRegions returns distinct region codes used in the community
func (h *MemberHandler) GetRegions(w http.ResponseWriter, r *http.Request) {
	community, err := h.getCommunityFromSlug(r)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	regions, err := h.memberRepo.GetDistinctRegions(r.Context(), community.ID)
	if err != nil {
		log.Printf("Error fetching regions for community %d: %v", community.ID, err)
		writeError(w, http.StatusInternalServerError, "failed to fetch regions")
		return
	}

	writeJSON(w, http.StatusOK, regions)
}
