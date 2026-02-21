package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/braccet/bracket/internal/service"
)

type ForfeitHandler struct {
	forfeitSvc service.ForfeitService
}

func NewForfeitHandler(forfeitSvc service.ForfeitService) *ForfeitHandler {
	return &ForfeitHandler{forfeitSvc: forfeitSvc}
}

type ForfeitParticipantRequest struct {
	TournamentID  uint64 `json:"tournament_id"`
	ParticipantID uint64 `json:"participant_id"`
}

type ForfeitResponse struct {
	ForfeitedMatches []uint64 `json:"forfeited_matches"`
	AdvancedWinners  []uint64 `json:"advanced_winners"`
}

// ForfeitParticipant processes a participant withdrawal by forfeiting their matches.
func (h *ForfeitHandler) ForfeitParticipant(w http.ResponseWriter, r *http.Request) {
	var req ForfeitParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TournamentID == 0 || req.ParticipantID == 0 {
		writeError(w, http.StatusBadRequest, "tournament_id and participant_id required")
		return
	}

	summary, err := h.forfeitSvc.ProcessWithdrawal(r.Context(), req.TournamentID, req.ParticipantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := ForfeitResponse{
		ForfeitedMatches: summary.ForfeitedMatches,
		AdvancedWinners:  summary.AdvancedWinners,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
