package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/braccet/community/internal/api/middleware"
	"github.com/braccet/community/internal/domain"
	"github.com/braccet/community/internal/repository"
	"github.com/braccet/community/internal/service"
	"github.com/go-chi/chi/v5"
)

type UploadHandler struct {
	storageService service.StorageService
	memberRepo     repository.MemberRepository
	communityRepo  repository.CommunityRepository
}

func NewUploadHandler(
	storageService service.StorageService,
	memberRepo repository.MemberRepository,
	communityRepo repository.CommunityRepository,
) *UploadHandler {
	return &UploadHandler{
		storageService: storageService,
		memberRepo:     memberRepo,
		communityRepo:  communityRepo,
	}
}

type GetUploadURLRequest struct {
	ContentType string `json:"content_type"`
}

// GetMemberIconUploadURL generates a presigned URL for uploading a member icon
// POST /communities/{slug}/members/{memberId}/icon/upload-url
func (h *UploadHandler) GetMemberIconUploadURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get community by slug
	slug := chi.URLParam(r, "slug")
	community, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch community")
		return
	}

	// Check if user is owner or admin
	if !h.isOwnerOrAdmin(r, community, userID) {
		writeError(w, http.StatusForbidden, "only owner or admin can upload member icons")
		return
	}

	// Get member by ID
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

	// Only ghost members can have icons uploaded
	if !member.IsGhost() {
		writeError(w, http.StatusForbidden, "only ghost members can have icons uploaded")
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
	response, err := h.storageService.GeneratePresignedUploadURL(r.Context(), community.ID, member.ID, req.ContentType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate upload URL")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// isOwnerOrAdmin checks if the user is an owner or admin of the community
func (h *UploadHandler) isOwnerOrAdmin(r *http.Request, community *domain.Community, userID uint64) bool {
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
