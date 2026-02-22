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

type EloHandler struct {
	eloService    service.EloService
	communityRepo repository.CommunityRepository
	memberRepo    repository.MemberRepository
}

func NewEloHandler(
	eloService service.EloService,
	communityRepo repository.CommunityRepository,
	memberRepo repository.MemberRepository,
) *EloHandler {
	return &EloHandler{
		eloService:    eloService,
		communityRepo: communityRepo,
		memberRepo:    memberRepo,
	}
}

// Request/Response types

type CreateEloSystemRequest struct {
	Name               string  `json:"name"`
	Description        *string `json:"description,omitempty"`
	StartingRating     *int    `json:"starting_rating,omitempty"`
	KFactor            *int    `json:"k_factor,omitempty"`
	FloorRating        *int    `json:"floor_rating,omitempty"`
	ProvisionalGames   *int    `json:"provisional_games,omitempty"`
	ProvisionalKFactor *int    `json:"provisional_k_factor,omitempty"`
	WinStreakEnabled   *bool   `json:"win_streak_enabled,omitempty"`
	WinStreakThreshold *int    `json:"win_streak_threshold,omitempty"`
	WinStreakBonus     *int    `json:"win_streak_bonus,omitempty"`
	DecayEnabled       *bool   `json:"decay_enabled,omitempty"`
	DecayDays          *int    `json:"decay_days,omitempty"`
	DecayAmount        *int    `json:"decay_amount,omitempty"`
	DecayFloor         *int    `json:"decay_floor,omitempty"`
	IsDefault          *bool   `json:"is_default,omitempty"`
}

type UpdateEloSystemRequest struct {
	Name               *string `json:"name,omitempty"`
	Description        *string `json:"description,omitempty"`
	StartingRating     *int    `json:"starting_rating,omitempty"`
	KFactor            *int    `json:"k_factor,omitempty"`
	FloorRating        *int    `json:"floor_rating,omitempty"`
	ProvisionalGames   *int    `json:"provisional_games,omitempty"`
	ProvisionalKFactor *int    `json:"provisional_k_factor,omitempty"`
	WinStreakEnabled   *bool   `json:"win_streak_enabled,omitempty"`
	WinStreakThreshold *int    `json:"win_streak_threshold,omitempty"`
	WinStreakBonus     *int    `json:"win_streak_bonus,omitempty"`
	DecayEnabled       *bool   `json:"decay_enabled,omitempty"`
	DecayDays          *int    `json:"decay_days,omitempty"`
	DecayAmount        *int    `json:"decay_amount,omitempty"`
	DecayFloor         *int    `json:"decay_floor,omitempty"`
	IsActive           *bool   `json:"is_active,omitempty"`
}

type EloSystemResponse struct {
	ID                 uint64  `json:"id"`
	CommunityID        uint64  `json:"community_id"`
	Name               string  `json:"name"`
	Description        *string `json:"description,omitempty"`
	StartingRating     int     `json:"starting_rating"`
	KFactor            int     `json:"k_factor"`
	FloorRating        int     `json:"floor_rating"`
	ProvisionalGames   int     `json:"provisional_games"`
	ProvisionalKFactor int     `json:"provisional_k_factor"`
	WinStreakEnabled   bool    `json:"win_streak_enabled"`
	WinStreakThreshold int     `json:"win_streak_threshold"`
	WinStreakBonus     int     `json:"win_streak_bonus"`
	DecayEnabled       bool    `json:"decay_enabled"`
	DecayDays          int     `json:"decay_days"`
	DecayAmount        int     `json:"decay_amount"`
	DecayFloor         int     `json:"decay_floor"`
	IsDefault          bool    `json:"is_default"`
	IsActive           bool    `json:"is_active"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

type MemberEloRatingResponse struct {
	ID               uint64  `json:"id"`
	MemberID         uint64  `json:"member_id"`
	MemberName       *string `json:"member_name,omitempty"`
	EloSystemID      uint64  `json:"elo_system_id"`
	Rating           int     `json:"rating"`
	GamesPlayed      int     `json:"games_played"`
	GamesWon         int     `json:"games_won"`
	CurrentWinStreak int     `json:"current_win_streak"`
	HighestRating    int     `json:"highest_rating"`
	LowestRating     int     `json:"lowest_rating"`
	LastGameAt       *string `json:"last_game_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type EloHistoryResponse struct {
	ID                   uint64   `json:"id"`
	MemberID             uint64   `json:"member_id"`
	EloSystemID          uint64   `json:"elo_system_id"`
	ChangeType           string   `json:"change_type"`
	RatingBefore         int      `json:"rating_before"`
	RatingChange         int      `json:"rating_change"`
	RatingAfter          int      `json:"rating_after"`
	MatchID              *uint64  `json:"match_id,omitempty"`
	TournamentID         *uint64  `json:"tournament_id,omitempty"`
	OpponentMemberID     *uint64  `json:"opponent_member_id,omitempty"`
	OpponentDisplayName  *string  `json:"opponent_display_name,omitempty"`
	OpponentRatingBefore *int     `json:"opponent_rating_before,omitempty"`
	IsWinner             *bool    `json:"is_winner,omitempty"`
	KFactorUsed          *int     `json:"k_factor_used,omitempty"`
	ExpectedScore        *float64 `json:"expected_score,omitempty"`
	WinStreakBonus       int      `json:"win_streak_bonus"`
	Notes                *string  `json:"notes,omitempty"`
	CreatedAt            string   `json:"created_at"`
}

type ProcessMatchEloRequest struct {
	EloSystemID    uint64 `json:"elo_system_id"`
	MatchID        uint64 `json:"match_id"`
	TournamentID   uint64 `json:"tournament_id"`
	WinnerMemberID uint64 `json:"winner_member_id"`
	LoserMemberID  uint64 `json:"loser_member_id"`
}

type ProcessMatchEloResponse struct {
	WinnerRatingBefore int `json:"winner_rating_before"`
	WinnerRatingAfter  int `json:"winner_rating_after"`
	WinnerChange       int `json:"winner_change"`
	LoserRatingBefore  int `json:"loser_rating_before"`
	LoserRatingAfter   int `json:"loser_rating_after"`
	LoserChange        int `json:"loser_change"`
}

// Helper functions

func toEloSystemResponse(s *domain.EloSystem) EloSystemResponse {
	return EloSystemResponse{
		ID:                 s.ID,
		CommunityID:        s.CommunityID,
		Name:               s.Name,
		Description:        s.Description,
		StartingRating:     s.StartingRating,
		KFactor:            s.KFactor,
		FloorRating:        s.FloorRating,
		ProvisionalGames:   s.ProvisionalGames,
		ProvisionalKFactor: s.ProvisionalKFactor,
		WinStreakEnabled:   s.WinStreakEnabled,
		WinStreakThreshold: s.WinStreakThreshold,
		WinStreakBonus:     s.WinStreakBonus,
		DecayEnabled:       s.DecayEnabled,
		DecayDays:          s.DecayDays,
		DecayAmount:        s.DecayAmount,
		DecayFloor:         s.DecayFloor,
		IsDefault:          s.IsDefault,
		IsActive:           s.IsActive,
		CreatedAt:          s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          s.UpdatedAt.Format(time.RFC3339),
	}
}

func toMemberEloRatingResponse(r *domain.MemberEloRating) MemberEloRatingResponse {
	resp := MemberEloRatingResponse{
		ID:               r.ID,
		MemberID:         r.MemberID,
		MemberName:       r.MemberDisplayName,
		EloSystemID:      r.EloSystemID,
		Rating:           r.Rating,
		GamesPlayed:      r.GamesPlayed,
		GamesWon:         r.GamesWon,
		CurrentWinStreak: r.CurrentWinStreak,
		HighestRating:    r.HighestRating,
		LowestRating:     r.LowestRating,
		CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        r.UpdatedAt.Format(time.RFC3339),
	}
	if r.LastGameAt != nil {
		formatted := r.LastGameAt.Format(time.RFC3339)
		resp.LastGameAt = &formatted
	}
	return resp
}

func toEloHistoryResponse(h *domain.EloHistory) EloHistoryResponse {
	return EloHistoryResponse{
		ID:                   h.ID,
		MemberID:             h.MemberID,
		EloSystemID:          h.EloSystemID,
		ChangeType:           string(h.ChangeType),
		RatingBefore:         h.RatingBefore,
		RatingChange:         h.RatingChange,
		RatingAfter:          h.RatingAfter,
		MatchID:              h.MatchID,
		TournamentID:         h.TournamentID,
		OpponentMemberID:     h.OpponentMemberID,
		OpponentDisplayName:  h.OpponentDisplayName,
		OpponentRatingBefore: h.OpponentRatingBefore,
		IsWinner:             h.IsWinner,
		KFactorUsed:          h.KFactorUsed,
		ExpectedScore:        h.ExpectedScore,
		WinStreakBonus:       h.WinStreakBonus,
		Notes:                h.Notes,
		CreatedAt:            h.CreatedAt.Format(time.RFC3339),
	}
}

// ListSystems returns all ELO systems for a community
func (h *EloHandler) ListSystems(w http.ResponseWriter, r *http.Request) {
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

	systems, err := h.eloService.GetSystemsByCommunity(r.Context(), community.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get ELO systems")
		return
	}

	var responses []EloSystemResponse
	for _, s := range systems {
		responses = append(responses, toEloSystemResponse(s))
	}

	writeJSON(w, http.StatusOK, responses)
}

// CreateSystem creates a new ELO system for a community
func (h *EloHandler) CreateSystem(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusForbidden, "Only owners and admins can create ELO systems")
		return
	}

	var req CreateEloSystemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Name is required")
		return
	}

	// Build system with defaults
	system := &domain.EloSystem{
		CommunityID:        community.ID,
		Name:               req.Name,
		Description:        req.Description,
		StartingRating:     1000,
		KFactor:            32,
		FloorRating:        100,
		ProvisionalGames:   10,
		ProvisionalKFactor: 64,
		WinStreakEnabled:   false,
		WinStreakThreshold: 3,
		WinStreakBonus:     5,
		DecayEnabled:       false,
		DecayDays:          30,
		DecayAmount:        10,
		DecayFloor:         800,
		IsDefault:          false,
		IsActive:           true,
	}

	// Override with provided values
	if req.StartingRating != nil {
		system.StartingRating = *req.StartingRating
	}
	if req.KFactor != nil {
		system.KFactor = *req.KFactor
	}
	if req.FloorRating != nil {
		system.FloorRating = *req.FloorRating
	}
	if req.ProvisionalGames != nil {
		system.ProvisionalGames = *req.ProvisionalGames
	}
	if req.ProvisionalKFactor != nil {
		system.ProvisionalKFactor = *req.ProvisionalKFactor
	}
	if req.WinStreakEnabled != nil {
		system.WinStreakEnabled = *req.WinStreakEnabled
	}
	if req.WinStreakThreshold != nil {
		system.WinStreakThreshold = *req.WinStreakThreshold
	}
	if req.WinStreakBonus != nil {
		system.WinStreakBonus = *req.WinStreakBonus
	}
	if req.DecayEnabled != nil {
		system.DecayEnabled = *req.DecayEnabled
	}
	if req.DecayDays != nil {
		system.DecayDays = *req.DecayDays
	}
	if req.DecayAmount != nil {
		system.DecayAmount = *req.DecayAmount
	}
	if req.DecayFloor != nil {
		system.DecayFloor = *req.DecayFloor
	}
	if req.IsDefault != nil {
		system.IsDefault = *req.IsDefault
	}

	if err := h.eloService.CreateSystem(r.Context(), system); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create ELO system")
		return
	}

	writeJSON(w, http.StatusCreated, toEloSystemResponse(system))
}

// GetSystem returns a single ELO system
func (h *EloHandler) GetSystem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "systemId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	system, err := h.eloService.GetSystem(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrEloSystemNotFound) {
			writeError(w, http.StatusNotFound, "ELO system not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get ELO system")
		return
	}

	writeJSON(w, http.StatusOK, toEloSystemResponse(system))
}

// UpdateSystem updates an ELO system
func (h *EloHandler) UpdateSystem(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusForbidden, "Only owners and admins can update ELO systems")
		return
	}

	system, err := h.eloService.GetSystem(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrEloSystemNotFound) {
			writeError(w, http.StatusNotFound, "ELO system not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get ELO system")
		return
	}

	// Verify system belongs to this community
	if system.CommunityID != community.ID {
		writeError(w, http.StatusNotFound, "ELO system not found")
		return
	}

	var req UpdateEloSystemRequest
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
	if req.StartingRating != nil {
		system.StartingRating = *req.StartingRating
	}
	if req.KFactor != nil {
		system.KFactor = *req.KFactor
	}
	if req.FloorRating != nil {
		system.FloorRating = *req.FloorRating
	}
	if req.ProvisionalGames != nil {
		system.ProvisionalGames = *req.ProvisionalGames
	}
	if req.ProvisionalKFactor != nil {
		system.ProvisionalKFactor = *req.ProvisionalKFactor
	}
	if req.WinStreakEnabled != nil {
		system.WinStreakEnabled = *req.WinStreakEnabled
	}
	if req.WinStreakThreshold != nil {
		system.WinStreakThreshold = *req.WinStreakThreshold
	}
	if req.WinStreakBonus != nil {
		system.WinStreakBonus = *req.WinStreakBonus
	}
	if req.DecayEnabled != nil {
		system.DecayEnabled = *req.DecayEnabled
	}
	if req.DecayDays != nil {
		system.DecayDays = *req.DecayDays
	}
	if req.DecayAmount != nil {
		system.DecayAmount = *req.DecayAmount
	}
	if req.DecayFloor != nil {
		system.DecayFloor = *req.DecayFloor
	}
	if req.IsActive != nil {
		system.IsActive = *req.IsActive
	}

	if err := h.eloService.UpdateSystem(r.Context(), system); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update ELO system")
		return
	}

	writeJSON(w, http.StatusOK, toEloSystemResponse(system))
}

// DeleteSystem deletes an ELO system
func (h *EloHandler) DeleteSystem(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusForbidden, "Only owners can delete ELO systems")
		return
	}

	system, err := h.eloService.GetSystem(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrEloSystemNotFound) {
			writeError(w, http.StatusNotFound, "ELO system not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get ELO system")
		return
	}

	// Verify system belongs to this community
	if system.CommunityID != community.ID {
		writeError(w, http.StatusNotFound, "ELO system not found")
		return
	}

	if err := h.eloService.DeleteSystem(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete ELO system")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetLeaderboard returns the top-rated members for an ELO system
func (h *EloHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
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

	ratings, err := h.eloService.GetLeaderboard(r.Context(), id, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get leaderboard")
		return
	}

	var responses []MemberEloRatingResponse
	for _, r := range ratings {
		responses = append(responses, toMemberEloRatingResponse(r))
	}

	writeJSON(w, http.StatusOK, responses)
}

// GetMemberRatings returns all ELO ratings for a member
func (h *EloHandler) GetMemberRatings(w http.ResponseWriter, r *http.Request) {
	memberIDStr := chi.URLParam(r, "memberId")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid member ID")
		return
	}

	ratings, err := h.eloService.GetMemberRatings(r.Context(), memberID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get member ratings")
		return
	}

	var responses []MemberEloRatingResponse
	for _, r := range ratings {
		responses = append(responses, toMemberEloRatingResponse(r))
	}

	writeJSON(w, http.StatusOK, responses)
}

// GetMemberHistory returns ELO history for a member in a system
func (h *EloHandler) GetMemberHistory(w http.ResponseWriter, r *http.Request) {
	memberIDStr := chi.URLParam(r, "memberId")
	memberID, err := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid member ID")
		return
	}

	systemIDStr := chi.URLParam(r, "systemId")
	systemID, err := strconv.ParseUint(systemIDStr, 10, 64)
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

	history, err := h.eloService.GetMemberHistory(r.Context(), memberID, systemID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get member history")
		return
	}

	var responses []EloHistoryResponse
	for _, h := range history {
		responses = append(responses, toEloHistoryResponse(h))
	}

	writeJSON(w, http.StatusOK, responses)
}

// ProcessMatch is an internal endpoint for processing match ELO updates
func (h *EloHandler) ProcessMatch(w http.ResponseWriter, r *http.Request) {
	var req ProcessMatchEloRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.EloSystemID == 0 || req.MatchID == 0 || req.TournamentID == 0 ||
		req.WinnerMemberID == 0 || req.LoserMemberID == 0 {
		writeError(w, http.StatusBadRequest, "All fields are required")
		return
	}

	result, err := h.eloService.ProcessMatchResult(r.Context(), service.ProcessMatchRequest{
		EloSystemID:    req.EloSystemID,
		MatchID:        req.MatchID,
		TournamentID:   req.TournamentID,
		WinnerMemberID: req.WinnerMemberID,
		LoserMemberID:  req.LoserMemberID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to process match ELO: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ProcessMatchEloResponse{
		WinnerRatingBefore: result.WinnerRatingBefore,
		WinnerRatingAfter:  result.WinnerRatingAfter,
		WinnerChange:       result.WinnerChange,
		LoserRatingBefore:  result.LoserRatingBefore,
		LoserRatingAfter:   result.LoserRatingAfter,
		LoserChange:        result.LoserChange,
	})
}

// GetSystemByID is an internal endpoint for getting a system by ID
func (h *EloHandler) GetSystemByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid system ID")
		return
	}

	system, err := h.eloService.GetSystem(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrEloSystemNotFound) {
			writeError(w, http.StatusNotFound, "ELO system not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to get ELO system")
		return
	}

	writeJSON(w, http.StatusOK, toEloSystemResponse(system))
}
