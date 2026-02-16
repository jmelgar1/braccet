package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/repository"
	"github.com/braccet/bracket/internal/service"
)

type MatchHandler struct {
	matchSvc service.MatchService
	repo     repository.MatchRepository
}

func NewMatchHandler(matchSvc service.MatchService, repo repository.MatchRepository) *MatchHandler {
	return &MatchHandler{
		matchSvc: matchSvc,
		repo:     repo,
	}
}

type ReportResultRequest struct {
	WinnerID          uint64 `json:"winner_id"`
	Participant1Score int    `json:"participant1_score"`
	Participant2Score int    `json:"participant2_score"`
}

func (h *MatchHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid match ID")
		return
	}

	match, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrMatchNotFound) {
			writeError(w, http.StatusNotFound, "match not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	json.NewEncoder(w).Encode(toMatchResponse(match))
}

func (h *MatchHandler) ReportResult(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid match ID")
		return
	}

	var req ReportResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.WinnerID == 0 {
		writeError(w, http.StatusBadRequest, "winner_id is required")
		return
	}

	result := domain.MatchResult{
		WinnerID:          req.WinnerID,
		Participant1Score: req.Participant1Score,
		Participant2Score: req.Participant2Score,
	}

	err = h.matchSvc.ReportResult(r.Context(), id, result)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrMatchNotFound):
			writeError(w, http.StatusNotFound, "match not found")
		case errors.Is(err, service.ErrMatchNotReady):
			writeError(w, http.StatusBadRequest, "match is not ready for result reporting")
		case errors.Is(err, service.ErrInvalidWinner):
			writeError(w, http.StatusBadRequest, "winner must be a participant in the match")
		case errors.Is(err, service.ErrMatchAlreadyComplete):
			writeError(w, http.StatusBadRequest, "match has already been completed")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Return updated match
	match, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	json.NewEncoder(w).Encode(toMatchResponse(match))
}

func (h *MatchHandler) Start(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid match ID")
		return
	}

	err = h.matchSvc.StartMatch(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrMatchNotFound):
			writeError(w, http.StatusNotFound, "match not found")
		case errors.Is(err, service.ErrMatchNotReady):
			writeError(w, http.StatusBadRequest, "match is not ready to start")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Return updated match
	match, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	json.NewEncoder(w).Encode(toMatchResponse(match))
}
