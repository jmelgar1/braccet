package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/repository"
	"github.com/braccet/bracket/internal/service"
)

type BracketHandler struct {
	bracketSvc service.BracketService
	matchSvc   service.MatchService
	repo       repository.MatchRepository
	setRepo    repository.SetRepository
}

func NewBracketHandler(bracketSvc service.BracketService, matchSvc service.MatchService, repo repository.MatchRepository, setRepo repository.SetRepository) *BracketHandler {
	return &BracketHandler{
		bracketSvc: bracketSvc,
		matchSvc:   matchSvc,
		repo:       repo,
		setRepo:    setRepo,
	}
}

type GenerateBracketRequest struct {
	TournamentID uint64               `json:"tournament_id"`
	Format       string               `json:"format"` // "single_elimination" or "double_elimination"
	Participants []domain.Participant `json:"participants"`
}

type BracketResponse struct {
	TournamentID uint64           `json:"tournament_id"`
	TotalRounds  int              `json:"total_rounds"`
	CurrentRound int              `json:"current_round"`
	IsComplete   bool             `json:"is_complete"`
	ChampionID   *uint64          `json:"champion_id,omitempty"`
	Matches      []*MatchResponse `json:"matches"`
}

type SetResponse struct {
	SetNumber         int `json:"set_number"`
	Participant1Score int `json:"participant1_score"`
	Participant2Score int `json:"participant2_score"`
}

type MatchResponse struct {
	ID               uint64        `json:"id"`
	Round            int           `json:"round"`
	Position         int           `json:"position"`
	BracketType      string        `json:"bracket_type"`
	Participant1ID   *uint64       `json:"participant1_id,omitempty"`
	Participant2ID   *uint64       `json:"participant2_id,omitempty"`
	Participant1Name *string       `json:"participant1_name,omitempty"`
	Participant2Name *string       `json:"participant2_name,omitempty"`
	Sets             []SetResponse `json:"sets"`
	Participant1Sets int           `json:"participant1_sets"`
	Participant2Sets int           `json:"participant2_sets"`
	WinnerID         *uint64       `json:"winner_id,omitempty"`
	ForfeitWinnerID  *uint64       `json:"forfeit_winner_id,omitempty"`
	Status           string        `json:"status"`
	NextMatchID      *uint64       `json:"next_match_id,omitempty"`
}

func (h *BracketHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req GenerateBracketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Participants) < 2 {
		writeError(w, http.StatusBadRequest, "at least 2 participants required")
		return
	}

	// Default to single elimination
	if req.Format == "" {
		req.Format = "single_elimination"
	}

	if req.Format != "single_elimination" {
		writeError(w, http.StatusBadRequest, "only single_elimination format is currently supported")
		return
	}

	state, err := h.bracketSvc.GenerateSingleElimination(r.Context(), req.TournamentID, req.Participants)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := toBracketResponse(state)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *BracketHandler) GetState(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := strconv.ParseUint(chi.URLParam(r, "tournamentId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	state, err := h.matchSvc.GetBracketState(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(state.Matches) == 0 {
		writeError(w, http.StatusNotFound, "bracket not found")
		return
	}

	// Load sets for all matches
	matchIDs := make([]uint64, len(state.Matches))
	for i, m := range state.Matches {
		matchIDs[i] = m.ID
	}
	setsMap, err := h.setRepo.GetByMatchIDs(r.Context(), matchIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Attach sets to matches
	for _, m := range state.Matches {
		if sets, ok := setsMap[m.ID]; ok {
			m.Sets = sets
		}
	}

	resp := toBracketResponse(state)
	json.NewEncoder(w).Encode(resp)
}

func (h *BracketHandler) ListMatches(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := strconv.ParseUint(chi.URLParam(r, "tournamentId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	matches, err := h.repo.GetByTournament(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Load sets for all matches
	matchIDs := make([]uint64, len(matches))
	for i, m := range matches {
		matchIDs[i] = m.ID
	}
	setsMap, err := h.setRepo.GetByMatchIDs(r.Context(), matchIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Attach sets to matches
	for _, m := range matches {
		if sets, ok := setsMap[m.ID]; ok {
			m.Sets = sets
		}
	}

	resp := make([]*MatchResponse, len(matches))
	for i, m := range matches {
		resp[i] = toMatchResponse(m)
	}

	json.NewEncoder(w).Encode(resp)
}

func toBracketResponse(state *service.BracketState) *BracketResponse {
	matches := make([]*MatchResponse, len(state.Matches))
	for i, m := range state.Matches {
		matches[i] = toMatchResponse(m)
	}

	return &BracketResponse{
		TournamentID: state.TournamentID,
		TotalRounds:  state.TotalRounds,
		CurrentRound: state.CurrentRound,
		IsComplete:   state.IsComplete,
		ChampionID:   state.ChampionID,
		Matches:      matches,
	}
}

func toMatchResponse(m *domain.Match) *MatchResponse {
	// Convert sets
	sets := make([]SetResponse, len(m.Sets))
	var p1Sets, p2Sets int
	for i, s := range m.Sets {
		sets[i] = SetResponse{
			SetNumber:         s.SetNumber,
			Participant1Score: s.Participant1Score,
			Participant2Score: s.Participant2Score,
		}
		if s.Participant1Score > s.Participant2Score {
			p1Sets++
		} else if s.Participant2Score > s.Participant1Score {
			p2Sets++
		}
	}

	return &MatchResponse{
		ID:               m.ID,
		Round:            m.Round,
		Position:         m.Position,
		BracketType:      string(m.BracketType),
		Participant1ID:   m.Participant1ID,
		Participant2ID:   m.Participant2ID,
		Participant1Name: m.Participant1Name,
		Participant2Name: m.Participant2Name,
		Sets:             sets,
		Participant1Sets: p1Sets,
		Participant2Sets: p2Sets,
		WinnerID:         m.WinnerID,
		ForfeitWinnerID:  m.ForfeitWinnerID,
		Status:           string(m.Status),
		NextMatchID:      m.NextMatchID,
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
