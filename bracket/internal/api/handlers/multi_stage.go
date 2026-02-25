package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/service"
)

type MultiStageHandler struct {
	multiStageSvc service.MultiStageService
}

func NewMultiStageHandler(multiStageSvc service.MultiStageService) *MultiStageHandler {
	return &MultiStageHandler{
		multiStageSvc: multiStageSvc,
	}
}

// CompleteStageRequest is the HTTP request body for completing a stage
type CompleteStageRequest struct {
	GroupIDs          []uint64 `json:"group_ids"`
	AdvancingPerGroup int      `json:"advancing_per_group"`
	RankingCriteria   []string `json:"ranking_criteria,omitempty"`
	Format            string   `json:"format"`
}

// CompleteStageResponse is the HTTP response for completing a stage
type CompleteStageResponseDTO struct {
	StageStandings        []StageStandingResponse   `json:"stage_standings"`
	AdvancingParticipants []AdvancingParticipantDTO `json:"advancing_participants"`
}

type StageStandingResponse struct {
	ParticipantID uint64 `json:"participant_id"`
	GroupID       uint64 `json:"group_id"`
	GroupRank     int    `json:"group_rank"`
	StageRank     *int   `json:"stage_rank,omitempty"`
	MatchWins     int    `json:"match_wins"`
	MatchLosses   int    `json:"match_losses"`
	SetWins       int    `json:"set_wins"`
	SetLosses     int    `json:"set_losses"`
	Advances      bool   `json:"advances"`
}

type AdvancingParticipantDTO struct {
	ParticipantID   uint64  `json:"participant_id"`
	ParticipantName string  `json:"participant_name"`
	IconURL         *string `json:"icon_url,omitempty"`
	StageRank       int     `json:"stage_rank"`
	GroupID         uint64  `json:"group_id"`
	GroupRank       int     `json:"group_rank"`
}

// CompleteStage finalizes a stage and returns advancing participants
// POST /brackets/{tournamentId}/stages/{stageId}/complete
func (h *MultiStageHandler) CompleteStage(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := strconv.ParseUint(chi.URLParam(r, "tournamentId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	stageID, err := strconv.ParseUint(chi.URLParam(r, "stageId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage ID")
		return
	}

	var req CompleteStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.GroupIDs) == 0 {
		writeError(w, http.StatusBadRequest, "group_ids is required")
		return
	}

	if req.AdvancingPerGroup <= 0 {
		writeError(w, http.StatusBadRequest, "advancing_per_group must be positive")
		return
	}

	// Convert string criteria to domain types
	var criteria []domain.RankingCriterion
	for _, c := range req.RankingCriteria {
		criteria = append(criteria, domain.RankingCriterion(c))
	}

	// Call service
	result, err := h.multiStageSvc.CompleteStage(r.Context(), service.CompleteStageRequest{
		TournamentID:      tournamentID,
		StageID:           stageID,
		GroupIDs:          req.GroupIDs,
		AdvancingPerGroup: req.AdvancingPerGroup,
		RankingCriteria:   criteria,
		Format:            req.Format,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build response
	standings := make([]StageStandingResponse, len(result.StageStandings))
	for i, ss := range result.StageStandings {
		standings[i] = StageStandingResponse{
			ParticipantID: ss.ParticipantID,
			GroupID:       ss.GroupID,
			GroupRank:     ss.GroupRank,
			StageRank:     ss.StageRank,
			MatchWins:     ss.MatchWins,
			MatchLosses:   ss.MatchLosses,
			SetWins:       ss.SetWins,
			SetLosses:     ss.SetLosses,
			Advances:      ss.Advances,
		}
	}

	advancing := make([]AdvancingParticipantDTO, len(result.AdvancingParticipants))
	for i, ap := range result.AdvancingParticipants {
		advancing[i] = AdvancingParticipantDTO{
			ParticipantID:   ap.ParticipantID,
			ParticipantName: ap.ParticipantName,
			IconURL:         ap.IconURL,
			StageRank:       ap.StageRank,
			GroupID:         ap.GroupID,
			GroupRank:       ap.GroupRank,
		}
	}

	resp := CompleteStageResponseDTO{
		StageStandings:        standings,
		AdvancingParticipants: advancing,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// GetStageAdvancingParticipants returns participants who advanced from a completed stage
// GET /brackets/{tournamentId}/stages/{stageId}/advancing
func (h *MultiStageHandler) GetStageAdvancingParticipants(w http.ResponseWriter, r *http.Request) {
	stageID, err := strconv.ParseUint(chi.URLParam(r, "stageId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stage ID")
		return
	}

	advancing, err := h.multiStageSvc.GetAdvancingParticipants(r.Context(), stageID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if advancing == nil {
		advancing = []service.AdvancingParticipant{}
	}

	// Convert to DTO
	result := make([]AdvancingParticipantDTO, len(advancing))
	for i, ap := range advancing {
		result[i] = AdvancingParticipantDTO{
			ParticipantID:   ap.ParticipantID,
			ParticipantName: ap.ParticipantName,
			IconURL:         ap.IconURL,
			StageRank:       ap.StageRank,
			GroupID:         ap.GroupID,
			GroupRank:       ap.GroupRank,
		}
	}

	resp := struct {
		AdvancingParticipants []AdvancingParticipantDTO `json:"advancing_participants"`
	}{
		AdvancingParticipants: result,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
