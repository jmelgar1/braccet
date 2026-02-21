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
	setRepo  repository.SetRepository
}

func NewMatchHandler(matchSvc service.MatchService, repo repository.MatchRepository, setRepo repository.SetRepository) *MatchHandler {
	return &MatchHandler{
		matchSvc: matchSvc,
		repo:     repo,
		setRepo:  setRepo,
	}
}

type SetScoreRequest struct {
	SetNumber         int `json:"set_number"`
	Participant1Score int `json:"participant1_score"`
	Participant2Score int `json:"participant2_score"`
}

type ReportResultRequest struct {
	Sets []SetScoreRequest `json:"sets"`
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

	// Load sets for this match
	sets, err := h.setRepo.GetByMatchID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	match.Sets = sets

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

	if len(req.Sets) == 0 {
		writeError(w, http.StatusBadRequest, "at least one set is required")
		return
	}

	// Validate set numbers are sequential starting from 1
	for i, set := range req.Sets {
		if set.SetNumber != i+1 {
			writeError(w, http.StatusBadRequest, "set numbers must be sequential starting from 1")
			return
		}
	}

	// Convert request to domain model
	sets := make([]domain.SetScore, len(req.Sets))
	for i, s := range req.Sets {
		sets[i] = domain.SetScore{
			SetNumber:         s.SetNumber,
			Participant1Score: s.Participant1Score,
			Participant2Score: s.Participant2Score,
		}
	}

	result := domain.MatchResult{Sets: sets}

	err = h.matchSvc.ReportResult(r.Context(), id, result)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrMatchNotFound):
			writeError(w, http.StatusNotFound, "match not found")
		case errors.Is(err, service.ErrMatchNotReady):
			writeError(w, http.StatusBadRequest, "match is not ready for result reporting")
		case errors.Is(err, service.ErrSetsTied):
			writeError(w, http.StatusBadRequest, "sets are tied - there must be a clear winner")
		case errors.Is(err, service.ErrNoSets):
			writeError(w, http.StatusBadRequest, "at least one set is required")
		case errors.Is(err, service.ErrMatchAlreadyComplete):
			writeError(w, http.StatusBadRequest, "match has already been completed")
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Return updated match with sets
	match, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Load sets for the updated match
	matchSets, err := h.setRepo.GetByMatchID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	match.Sets = matchSets

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

	// Return updated match with sets
	match, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Load sets (will be empty for a just-started match, but keeps response consistent)
	sets, err := h.setRepo.GetByMatchID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	match.Sets = sets

	json.NewEncoder(w).Encode(toMatchResponse(match))
}
