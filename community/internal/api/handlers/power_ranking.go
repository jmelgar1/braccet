package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/braccet/community/internal/api/middleware"
	"github.com/braccet/community/internal/domain"
	"github.com/braccet/community/internal/repository"
	"github.com/braccet/community/internal/service"
	"github.com/go-chi/chi/v5"
)

type PowerRankingHandler struct {
	prService     service.PowerRankingService
	communityRepo repository.CommunityRepository
	memberRepo    repository.MemberRepository
}

func NewPowerRankingHandler(
	prService service.PowerRankingService,
	communityRepo repository.CommunityRepository,
	memberRepo repository.MemberRepository,
) *PowerRankingHandler {
	return &PowerRankingHandler{
		prService:     prService,
		communityRepo: communityRepo,
		memberRepo:    memberRepo,
	}
}

// =============================================================================
// Request/Response types
// =============================================================================

type CreatePRSystemRequest struct {
	Name                    string  `json:"name"`
	Description             *string `json:"description,omitempty"`
	AchievementsWeight      *int    `json:"achievements_weight,omitempty"`
	FormWeight              *int    `json:"form_weight,omitempty"`
	LANWeight               *int    `json:"lan_weight,omitempty"`
	AchievementDecayMonths  *int    `json:"achievement_decay_months,omitempty"`
	AchievementWindowMonths *int    `json:"achievement_window_months,omitempty"`
	FormWindowMonths        *int    `json:"form_window_months,omitempty"`
	LANResultsCount         *int    `json:"lan_results_count,omitempty"`
	IsDefault               *bool   `json:"is_default,omitempty"`
}

type UpdatePRSystemRequest struct {
	Name                    *string `json:"name,omitempty"`
	Description             *string `json:"description,omitempty"`
	AchievementsWeight      *int    `json:"achievements_weight,omitempty"`
	FormWeight              *int    `json:"form_weight,omitempty"`
	LANWeight               *int    `json:"lan_weight,omitempty"`
	AchievementDecayMonths  *int    `json:"achievement_decay_months,omitempty"`
	AchievementWindowMonths *int    `json:"achievement_window_months,omitempty"`
	FormWindowMonths        *int    `json:"form_window_months,omitempty"`
	LANResultsCount         *int    `json:"lan_results_count,omitempty"`
	IsActive                *bool   `json:"is_active,omitempty"`
}

type PRSystemResponse struct {
	ID                      uint64  `json:"id"`
	CommunityID             uint64  `json:"community_id"`
	Name                    string  `json:"name"`
	Description             *string `json:"description,omitempty"`
	AchievementsWeight      int     `json:"achievements_weight"`
	FormWeight              int     `json:"form_weight"`
	LANWeight               int     `json:"lan_weight"`
	AchievementDecayMonths  int     `json:"achievement_decay_months"`
	AchievementWindowMonths int     `json:"achievement_window_months"`
	FormWindowMonths        int     `json:"form_window_months"`
	LANResultsCount         int     `json:"lan_results_count"`
	IsDefault               bool    `json:"is_default"`
	IsActive                bool    `json:"is_active"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
}

type MemberPowerRankingResponse struct {
	ID                    uint64   `json:"id"`
	MemberID              uint64   `json:"member_id"`
	MemberName            *string  `json:"member_name,omitempty"`
	MemberRegion          *string  `json:"member_region,omitempty"`
	MemberIconURL         *string  `json:"member_icon_url,omitempty"`
	PRSystemID            uint64   `json:"pr_system_id"`
	TotalPoints           float64  `json:"total_points"`
	Rank                  *int     `json:"rank,omitempty"`
	AchievementsScore     float64  `json:"achievements_score"`
	FormScore             float64  `json:"form_score"`
	LANScore              float64  `json:"lan_score"`
	FormWins              int      `json:"form_wins"`
	FormSetWins           int      `json:"form_set_wins"`
	FormSetLosses         int      `json:"form_set_losses"`
	FormAvgOpponentRating *int     `json:"form_avg_opponent_rating,omitempty"`
	FormBestWinRating     *int     `json:"form_best_win_rating,omitempty"`
	LANPlacementsCount    int      `json:"lan_placements_count"`
	LastCalculatedAt      *string  `json:"last_calculated_at,omitempty"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
}

type TournamentPlacementResponse struct {
	ID                uint64   `json:"id"`
	MemberID          uint64   `json:"member_id"`
	PRSystemID        uint64   `json:"pr_system_id"`
	TournamentID      uint64   `json:"tournament_id"`
	Placement         int      `json:"placement"`
	ParticipantCount  int      `json:"participant_count"`
	TournamentClass   string   `json:"tournament_class"`
	PrizePoolUSD      *float64 `json:"prize_pool_usd,omitempty"`
	IsBo3Playoffs     bool     `json:"is_bo3_playoffs"`
	AvgOpponentRating *int     `json:"avg_opponent_rating,omitempty"`
	IsLAN             bool     `json:"is_lan"`
	RawPoints         int      `json:"raw_points"`
	CompletedAt       string   `json:"completed_at"`
	CreatedAt         string   `json:"created_at"`
}

type ProcessPlacementRequest struct {
	PRSystemID        uint64   `json:"pr_system_id"`
	TournamentID      uint64   `json:"tournament_id"`
	MemberID          uint64   `json:"member_id"`
	Placement         int      `json:"placement"`
	ParticipantCount  int      `json:"participant_count"`
	TournamentClass   string   `json:"tournament_class"`
	PrizePoolUSD      *float64 `json:"prize_pool_usd,omitempty"`
	IsBo3Playoffs     bool     `json:"is_bo3_playoffs"`
	AvgOpponentRating *int     `json:"avg_opponent_rating,omitempty"`
	IsLAN             bool     `json:"is_lan"`
	CompletedAt       string   `json:"completed_at"`
}

type ProcessMatchRequest struct {
	PRSystemID     uint64 `json:"pr_system_id"`
	MatchID        uint64 `json:"match_id"`
	TournamentID   uint64 `json:"tournament_id"`
	WinnerMemberID uint64 `json:"winner_member_id"`
	LoserMemberID  uint64 `json:"loser_member_id"`
	WinnerSetsWon  int    `json:"winner_sets_won"`
	LoserSetsWon   int    `json:"loser_sets_won"`
	WinnerRating   int    `json:"winner_rating"`
	LoserRating    int    `json:"loser_rating"`
	IsLAN          bool   `json:"is_lan"`
	CompletedAt    string `json:"completed_at"`
}

// =============================================================================
// Helper functions
// =============================================================================

func toPRSystemResponse(s *domain.PowerRankingSystem) PRSystemResponse {
	return PRSystemResponse{
		ID:                      s.ID,
		CommunityID:             s.CommunityID,
		Name:                    s.Name,
		Description:             s.Description,
		AchievementsWeight:      s.AchievementsWeight,
		FormWeight:              s.FormWeight,
		LANWeight:               s.LANWeight,
		AchievementDecayMonths:  s.AchievementDecayMonths,
		AchievementWindowMonths: s.AchievementWindowMonths,
		FormWindowMonths:        s.FormWindowMonths,
		LANResultsCount:         s.LANResultsCount,
		IsDefault:               s.IsDefault,
		IsActive:                s.IsActive,
		CreatedAt:               s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               s.UpdatedAt.Format(time.RFC3339),
	}
}

func toMemberPowerRankingResponse(r *domain.MemberPowerRanking) MemberPowerRankingResponse {
	resp := MemberPowerRankingResponse{
		ID:                    r.ID,
		MemberID:              r.MemberID,
		MemberName:            r.MemberDisplayName,
		MemberRegion:          r.MemberRegion,
		MemberIconURL:         r.MemberIconURL,
		PRSystemID:            r.PRSystemID,
		TotalPoints:           r.TotalPoints,
		Rank:                  r.Rank,
		AchievementsScore:     r.AchievementsScore,
		FormScore:             r.FormScore,
		LANScore:              r.LANScore,
		FormWins:              r.FormWins,
		FormSetWins:           r.FormSetWins,
		FormSetLosses:         r.FormSetLosses,
		FormAvgOpponentRating: r.FormAvgOpponentRating,
		FormBestWinRating:     r.FormBestWinRating,
		LANPlacementsCount:    r.LANPlacementsCount,
		CreatedAt:             r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             r.UpdatedAt.Format(time.RFC3339),
	}
	if r.LastCalculatedAt != nil {
		formatted := r.LastCalculatedAt.Format(time.RFC3339)
		resp.LastCalculatedAt = &formatted
	}
	return resp
}

func toTournamentPlacementResponse(p *domain.TournamentPlacement) TournamentPlacementResponse {
	return TournamentPlacementResponse{
		ID:                p.ID,
		MemberID:          p.MemberID,
		PRSystemID:        p.PRSystemID,
		TournamentID:      p.TournamentID,
		Placement:         p.Placement,
		ParticipantCount:  p.ParticipantCount,
		TournamentClass:   string(p.TournamentClass),
		PrizePoolUSD:      p.PrizePoolUSD,
		IsBo3Playoffs:     p.IsBo3Playoffs,
		AvgOpponentRating: p.AvgOpponentRating,
		IsLAN:             p.IsLAN,
		RawPoints:         p.RawPoints,
		CompletedAt:       p.CompletedAt.Format(time.RFC3339),
		CreatedAt:         p.CreatedAt.Format(time.RFC3339),
	}
}

// =============================================================================
// Public Handlers - System CRUD
// =============================================================================

// ListSystems returns all PR systems for a community
func (h *PowerRankingHandler) ListSystems(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	community, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "Community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get community")
		return
	}

	systems, err := h.prService.GetSystemsByCommunity(r.Context(), community.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get power ranking systems")
		return
	}

	var responses []PRSystemResponse
	for _, s := range systems {
		responses = append(responses, toPRSystemResponse(s))
	}

	writeJSON(w, http.StatusOK, responses)
}

// CreateSystem creates a new PR system for a community
func (h *PowerRankingHandler) CreateSystem(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	userID, _ := middleware.GetUserID(r.Context())

	community, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "Community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get community")
		return
	}

	// Check if user is owner or admin
	member, err := h.memberRepo.GetByCommunityAndUser(r.Context(), community.ID, userID)
	if err != nil || (member.Role != domain.RoleOwner && member.Role != domain.RoleAdmin) {
		writeError(w, http.StatusForbidden, "Only owners and admins can create power ranking systems")
		return
	}

	var req CreatePRSystemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}

	// Build system with defaults
	system := &domain.PowerRankingSystem{
		CommunityID:            community.ID,
		Name:                   req.Name,
		Description:            req.Description,
		AchievementsWeight:     50,
		FormWeight:             30,
		LANWeight:              20,
		AchievementDecayMonths: 6,
		AchievementWindowMonths: 12,
		FormWindowMonths:       3,
		LANResultsCount:        10,
		IsDefault:              false,
		IsActive:               true,
	}

	// Override with provided values
	if req.AchievementsWeight != nil {
		system.AchievementsWeight = *req.AchievementsWeight
	}
	if req.FormWeight != nil {
		system.FormWeight = *req.FormWeight
	}
	if req.LANWeight != nil {
		system.LANWeight = *req.LANWeight
	}
	if req.AchievementDecayMonths != nil {
		system.AchievementDecayMonths = *req.AchievementDecayMonths
	}
	if req.AchievementWindowMonths != nil {
		system.AchievementWindowMonths = *req.AchievementWindowMonths
	}
	if req.FormWindowMonths != nil {
		system.FormWindowMonths = *req.FormWindowMonths
	}
	if req.LANResultsCount != nil {
		system.LANResultsCount = *req.LANResultsCount
	}
	if req.IsDefault != nil {
		system.IsDefault = *req.IsDefault
	}

	// Validate weights sum to 100
	if system.AchievementsWeight+system.FormWeight+system.LANWeight != 100 {
		writeError(w, http.StatusBadRequest, "Component weights must sum to 100")
		return
	}

	if err := h.prService.CreateSystem(r.Context(), system); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create power ranking system")
		return
	}

	writeJSON(w, http.StatusCreated, toPRSystemResponse(system))
}

// GetSystem returns a single PR system
func (h *PowerRankingHandler) GetSystem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "systemId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	system, err := h.prService.GetSystem(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrPRSystemNotFound) {
			writeError(w, http.StatusNotFound, "Power ranking system not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get power ranking system")
		return
	}

	writeJSON(w, http.StatusOK, toPRSystemResponse(system))
}

// UpdateSystem updates a PR system
func (h *PowerRankingHandler) UpdateSystem(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "systemId")
	userID, _ := middleware.GetUserID(r.Context())

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	community, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "Community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get community")
		return
	}

	// Check if user is owner or admin
	member, err := h.memberRepo.GetByCommunityAndUser(r.Context(), community.ID, userID)
	if err != nil || (member.Role != domain.RoleOwner && member.Role != domain.RoleAdmin) {
		writeError(w, http.StatusForbidden, "Only owners and admins can update power ranking systems")
		return
	}

	system, err := h.prService.GetSystem(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrPRSystemNotFound) {
			writeError(w, http.StatusNotFound, "Power ranking system not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get power ranking system")
		return
	}

	// Verify system belongs to this community
	if system.CommunityID != community.ID {
		writeError(w, http.StatusNotFound, "Power ranking system not found")
		return
	}

	var req UpdatePRSystemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update fields
	if req.Name != nil {
		system.Name = *req.Name
	}
	if req.Description != nil {
		system.Description = req.Description
	}
	if req.AchievementsWeight != nil {
		system.AchievementsWeight = *req.AchievementsWeight
	}
	if req.FormWeight != nil {
		system.FormWeight = *req.FormWeight
	}
	if req.LANWeight != nil {
		system.LANWeight = *req.LANWeight
	}
	if req.AchievementDecayMonths != nil {
		system.AchievementDecayMonths = *req.AchievementDecayMonths
	}
	if req.AchievementWindowMonths != nil {
		system.AchievementWindowMonths = *req.AchievementWindowMonths
	}
	if req.FormWindowMonths != nil {
		system.FormWindowMonths = *req.FormWindowMonths
	}
	if req.LANResultsCount != nil {
		system.LANResultsCount = *req.LANResultsCount
	}
	if req.IsActive != nil {
		system.IsActive = *req.IsActive
	}

	// Validate weights sum to 100
	if system.AchievementsWeight+system.FormWeight+system.LANWeight != 100 {
		writeError(w, http.StatusBadRequest, "Component weights must sum to 100")
		return
	}

	if err := h.prService.UpdateSystem(r.Context(), system); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update power ranking system")
		return
	}

	writeJSON(w, http.StatusOK, toPRSystemResponse(system))
}

// DeleteSystem deletes a PR system
func (h *PowerRankingHandler) DeleteSystem(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	idStr := chi.URLParam(r, "systemId")
	userID, _ := middleware.GetUserID(r.Context())

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	community, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "Community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get community")
		return
	}

	// Only owner can delete
	member, err := h.memberRepo.GetByCommunityAndUser(r.Context(), community.ID, userID)
	if err != nil || member.Role != domain.RoleOwner {
		writeError(w, http.StatusForbidden, "Only owners can delete power ranking systems")
		return
	}

	system, err := h.prService.GetSystem(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrPRSystemNotFound) {
			writeError(w, http.StatusNotFound, "Power ranking system not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get power ranking system")
		return
	}

	// Verify system belongs to this community
	if system.CommunityID != community.ID {
		writeError(w, http.StatusNotFound, "Power ranking system not found")
		return
	}

	if err := h.prService.DeleteSystem(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete power ranking system")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// =============================================================================
// Public Handlers - Leaderboard & Member Rankings
// =============================================================================

// GetLeaderboard returns the power ranking leaderboard (triggers calculation)
func (h *PowerRankingHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "systemId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	rankings, err := h.prService.GetLeaderboard(r.Context(), id, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get leaderboard")
		return
	}

	var responses []MemberPowerRankingResponse
	for _, r := range rankings {
		responses = append(responses, toMemberPowerRankingResponse(r))
	}

	writeJSON(w, http.StatusOK, responses)
}

// GetMemberRankings returns all PR rankings for a member
func (h *PowerRankingHandler) GetMemberRankings(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	memberIDStr := chi.URLParam(r, "memberId")

	memberID, err := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid member ID")
		return
	}

	community, err := h.communityRepo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			writeError(w, http.StatusNotFound, "Community not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get community")
		return
	}

	// Get all PR systems for this community
	systems, err := h.prService.GetSystemsByCommunity(r.Context(), community.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get power ranking systems")
		return
	}

	var responses []MemberPowerRankingResponse
	for _, system := range systems {
		ranking, err := h.prService.GetMemberRanking(r.Context(), memberID, system.ID)
		if err != nil {
			continue // Skip systems where member has no ranking
		}
		responses = append(responses, toMemberPowerRankingResponse(ranking))
	}

	writeJSON(w, http.StatusOK, responses)
}

// GetMemberPlacements returns all placements for a member in a system
func (h *PowerRankingHandler) GetMemberPlacements(w http.ResponseWriter, r *http.Request) {
	memberIDStr := chi.URLParam(r, "memberId")
	systemIDStr := chi.URLParam(r, "systemId")

	memberID, err := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid member ID")
		return
	}

	systemID, err := strconv.ParseUint(systemIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	placements, err := h.prService.GetMemberPlacements(r.Context(), memberID, systemID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get member placements")
		return
	}

	var responses []TournamentPlacementResponse
	for _, p := range placements {
		responses = append(responses, toTournamentPlacementResponse(p))
	}

	writeJSON(w, http.StatusOK, responses)
}

// =============================================================================
// Internal Handlers - Service-to-Service
// =============================================================================

// ProcessPlacement records a tournament placement (called from bracket service)
func (h *PowerRankingHandler) ProcessPlacement(w http.ResponseWriter, r *http.Request) {
	var req ProcessPlacementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PRSystemID == 0 || req.TournamentID == 0 || req.MemberID == 0 ||
		req.Placement == 0 || req.ParticipantCount == 0 {
		writeError(w, http.StatusBadRequest, "Required fields: pr_system_id, tournament_id, member_id, placement, participant_count")
		return
	}

	// Parse completed_at
	completedAt := time.Now()
	if req.CompletedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.CompletedAt)
		if err == nil {
			completedAt = parsed
		}
	}

	domainReq := &domain.ProcessPlacementRequest{
		PRSystemID:        req.PRSystemID,
		TournamentID:      req.TournamentID,
		MemberID:          req.MemberID,
		Placement:         req.Placement,
		ParticipantCount:  req.ParticipantCount,
		TournamentClass:   domain.TournamentClass(req.TournamentClass),
		PrizePoolUSD:      req.PrizePoolUSD,
		IsBo3Playoffs:     req.IsBo3Playoffs,
		AvgOpponentRating: req.AvgOpponentRating,
		IsLAN:             req.IsLAN,
		CompletedAt:       completedAt,
	}

	// Default to "online" if no class specified
	if domainReq.TournamentClass == "" {
		domainReq.TournamentClass = domain.TournamentClassOnline
	}

	if err := h.prService.ProcessPlacement(r.Context(), domainReq); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to process placement: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// ProcessMatch records a match result for form tracking (called from bracket service)
func (h *PowerRankingHandler) ProcessMatch(w http.ResponseWriter, r *http.Request) {
	var req ProcessMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.PRSystemID == 0 || req.MatchID == 0 || req.TournamentID == 0 ||
		req.WinnerMemberID == 0 || req.LoserMemberID == 0 {
		writeError(w, http.StatusBadRequest, "Required fields: pr_system_id, match_id, tournament_id, winner_member_id, loser_member_id")
		return
	}

	// Parse completed_at
	completedAt := time.Now()
	if req.CompletedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.CompletedAt)
		if err == nil {
			completedAt = parsed
		}
	}

	domainReq := &domain.ProcessPRMatchRequest{
		PRSystemID:     req.PRSystemID,
		MatchID:        req.MatchID,
		TournamentID:   req.TournamentID,
		WinnerMemberID: req.WinnerMemberID,
		LoserMemberID:  req.LoserMemberID,
		WinnerSetsWon:  req.WinnerSetsWon,
		LoserSetsWon:   req.LoserSetsWon,
		WinnerRating:   req.WinnerRating,
		LoserRating:    req.LoserRating,
		IsLAN:          req.IsLAN,
		CompletedAt:    completedAt,
	}

	if err := h.prService.ProcessMatch(r.Context(), domainReq); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to process match: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetSystemByID is an internal endpoint for getting a system by ID
func (h *PowerRankingHandler) GetSystemByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	system, err := h.prService.GetSystem(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrPRSystemNotFound) {
			writeError(w, http.StatusNotFound, "Power ranking system not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get power ranking system")
		return
	}

	writeJSON(w, http.StatusOK, toPRSystemResponse(system))
}

// RevertTournamentPlacements reverts all placements from a tournament
// DELETE /internal/power-rankings/tournament/{tournamentId}
func (h *PowerRankingHandler) RevertTournamentPlacements(w http.ResponseWriter, r *http.Request) {
	tournamentIDStr := chi.URLParam(r, "tournamentId")
	tournamentID, err := strconv.ParseUint(tournamentIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid tournament ID")
		return
	}

	if err := h.prService.RevertTournamentPlacements(r.Context(), tournamentID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to revert tournament placements: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
