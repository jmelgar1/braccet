package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/service"
)

type StageHandler struct {
	stageSvc service.StageService
}

func NewStageHandler(stageSvc service.StageService) *StageHandler {
	return &StageHandler{stageSvc: stageSvc}
}

type UpdateStageRequest struct {
	StageName   *string `json:"stage_name,omitempty"`
	BestOf      *int    `json:"best_of,omitempty"`
	BracketType *string `json:"bracket_type,omitempty"` // Optional: "winners", "losers", "grand_final", "swiss"
}

func (h *StageHandler) GetStages(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := strconv.ParseUint(chi.URLParam(r, "tournamentId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	stages, err := h.stageSvc.GetStages(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]*StageResponse, len(stages))
	for i, s := range stages {
		stageName := ""
		if s.StageName != nil {
			stageName = *s.StageName
		}
		resp[i] = &StageResponse{
			TournamentID: s.TournamentID,
			BracketType:  string(s.BracketType),
			Round:        s.Round,
			StageName:    stageName,
			BestOf:       s.BestOf,
		}
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *StageHandler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := strconv.ParseUint(chi.URLParam(r, "tournamentId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	round, err := strconv.Atoi(chi.URLParam(r, "round"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid round number")
		return
	}

	var req UpdateStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Default to winners bracket for single elimination
	bracketType := domain.BracketWinners
	if req.BracketType != nil {
		switch *req.BracketType {
		case "winners":
			bracketType = domain.BracketWinners
		case "losers":
			bracketType = domain.BracketLosers
		case "grand_final":
			bracketType = domain.BracketGrandFinal
		case "swiss":
			bracketType = domain.BracketSwiss
		default:
			writeError(w, http.StatusBadRequest, "invalid bracket_type")
			return
		}
	}

	svcReq := service.UpdateStageRequest{
		StageName: req.StageName,
		BestOf:    req.BestOf,
	}

	stage, err := h.stageSvc.UpdateStage(r.Context(), tournamentID, bracketType, round, svcReq)
	if err != nil {
		if err == service.ErrInvalidBestOf {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stageName := ""
	if stage.StageName != nil {
		stageName = *stage.StageName
	}
	resp := &StageResponse{
		TournamentID: stage.TournamentID,
		BracketType:  string(stage.BracketType),
		Round:        stage.Round,
		StageName:    stageName,
		BestOf:       stage.BestOf,
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *StageHandler) GetGroupStages(w http.ResponseWriter, r *http.Request) {
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

	groupID, err := strconv.ParseUint(chi.URLParam(r, "groupId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid group ID")
		return
	}

	stages, err := h.stageSvc.GetGroupStages(r.Context(), tournamentID, stageID, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]*StageResponse, len(stages))
	for i, s := range stages {
		stageName := ""
		if s.StageName != nil {
			stageName = *s.StageName
		}
		resp[i] = &StageResponse{
			TournamentID: s.TournamentID,
			BracketType:  string(s.BracketType),
			Round:        s.Round,
			StageName:    stageName,
			BestOf:       s.BestOf,
		}
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *StageHandler) UpdateGroupStage(w http.ResponseWriter, r *http.Request) {
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

	groupID, err := strconv.ParseUint(chi.URLParam(r, "groupId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid group ID")
		return
	}

	round, err := strconv.Atoi(chi.URLParam(r, "round"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid round number")
		return
	}

	var req UpdateStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Default to Swiss bracket type for group stages
	bracketType := domain.BracketSwiss
	if req.BracketType != nil {
		switch *req.BracketType {
		case "winners":
			bracketType = domain.BracketWinners
		case "losers":
			bracketType = domain.BracketLosers
		case "swiss":
			bracketType = domain.BracketSwiss
		default:
			writeError(w, http.StatusBadRequest, "invalid bracket_type")
			return
		}
	}

	svcReq := service.UpdateStageRequest{
		StageName: req.StageName,
		BestOf:    req.BestOf,
	}

	stage, err := h.stageSvc.UpdateGroupStage(r.Context(), tournamentID, stageID, groupID, bracketType, round, svcReq)
	if err != nil {
		if err == service.ErrInvalidBestOf {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stageName := ""
	if stage.StageName != nil {
		stageName = *stage.StageName
	}
	resp := &StageResponse{
		TournamentID: stage.TournamentID,
		BracketType:  string(stage.BracketType),
		Round:        stage.Round,
		StageName:    stageName,
		BestOf:       stage.BestOf,
	}
	json.NewEncoder(w).Encode(resp)
}
