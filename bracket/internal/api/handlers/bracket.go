package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/braccet/bracket/internal/domain"
	"github.com/braccet/bracket/internal/engine"
	"github.com/braccet/bracket/internal/repository"
	"github.com/braccet/bracket/internal/service"
)

type BracketHandler struct {
	bracketSvc    service.BracketService
	matchSvc      service.MatchService
	swissSvc      service.SwissService
	tiebreakerSvc service.TiebreakerService
	repo          repository.MatchRepository
	setRepo       repository.SetRepository
	stageRepo     repository.StageRepository
}

func NewBracketHandler(bracketSvc service.BracketService, matchSvc service.MatchService, repo repository.MatchRepository, setRepo repository.SetRepository, stageRepo repository.StageRepository) *BracketHandler {
	return &BracketHandler{
		bracketSvc: bracketSvc,
		matchSvc:   matchSvc,
		repo:       repo,
		setRepo:    setRepo,
		stageRepo:  stageRepo,
	}
}

// NewBracketHandlerWithSwiss creates a bracket handler with Swiss support
func NewBracketHandlerWithSwiss(bracketSvc service.BracketService, matchSvc service.MatchService, swissSvc service.SwissService, repo repository.MatchRepository, setRepo repository.SetRepository, stageRepo repository.StageRepository) *BracketHandler {
	return &BracketHandler{
		bracketSvc: bracketSvc,
		matchSvc:   matchSvc,
		swissSvc:   swissSvc,
		repo:       repo,
		setRepo:    setRepo,
		stageRepo:  stageRepo,
	}
}

// NewBracketHandlerWithTiebreakers creates a bracket handler with Swiss and tiebreaker support
func NewBracketHandlerWithTiebreakers(bracketSvc service.BracketService, matchSvc service.MatchService, swissSvc service.SwissService, tiebreakerSvc service.TiebreakerService, repo repository.MatchRepository, setRepo repository.SetRepository, stageRepo repository.StageRepository) *BracketHandler {
	return &BracketHandler{
		bracketSvc:    bracketSvc,
		matchSvc:      matchSvc,
		swissSvc:      swissSvc,
		tiebreakerSvc: tiebreakerSvc,
		repo:          repo,
		setRepo:       setRepo,
		stageRepo:     stageRepo,
	}
}

type GenerateBracketRequest struct {
	TournamentID uint64               `json:"tournament_id"`
	Format       string               `json:"format"` // "single_elimination", "double_elimination", or "swiss"
	Participants []domain.Participant `json:"participants"`
	SwissRounds  *int                 `json:"swiss_rounds,omitempty"` // Optional override for Swiss round count
}

type BracketResponse struct {
	TournamentID  uint64           `json:"tournament_id"`
	Format        string           `json:"format"`
	TotalRounds   int              `json:"total_rounds"`
	WinnersRounds int              `json:"winners_rounds,omitempty"`
	LosersRounds  int              `json:"losers_rounds,omitempty"`
	CurrentRound  int              `json:"current_round"`
	IsComplete    bool             `json:"is_complete"`
	ChampionID    *uint64          `json:"champion_id,omitempty"`
	Matches       []*MatchResponse `json:"matches"`
	Stages        []*StageResponse `json:"stages"`
}

type StageResponse struct {
	TournamentID uint64 `json:"tournament_id"`
	BracketType  string `json:"bracket_type"`
	Round        int    `json:"round"`
	StageName    string `json:"stage_name"`
	BestOf       int    `json:"best_of"`
}

type SetResponse struct {
	SetNumber         int `json:"set_number"`
	Participant1Score int `json:"participant1_score"`
	Participant2Score int `json:"participant2_score"`
}

type MatchResponse struct {
	ID                  uint64        `json:"id"`
	Round               int           `json:"round"`
	Position            int           `json:"position"`
	BracketType         string        `json:"bracket_type"`
	Participant1ID      *uint64       `json:"participant1_id,omitempty"`
	Participant2ID      *uint64       `json:"participant2_id,omitempty"`
	Participant1Name    *string       `json:"participant1_name,omitempty"`
	Participant2Name    *string       `json:"participant2_name,omitempty"`
	Participant1IconURL *string       `json:"participant1_icon_url,omitempty"`
	Participant2IconURL *string       `json:"participant2_icon_url,omitempty"`
	Seed1               *int          `json:"seed1,omitempty"`
	Seed2               *int          `json:"seed2,omitempty"`
	Sets                []SetResponse `json:"sets"`
	Participant1Sets    int           `json:"participant1_sets"`
	Participant2Sets    int           `json:"participant2_sets"`
	WinnerID            *uint64       `json:"winner_id,omitempty"`
	ForfeitWinnerID     *uint64       `json:"forfeit_winner_id,omitempty"`
	Status              string        `json:"status"`
	NextMatchID         *uint64       `json:"next_match_id,omitempty"`
	LoserMatchID        *uint64       `json:"loser_match_id,omitempty"`
}

// PreviewRequest is similar to GenerateBracketRequest but for preview only (no persistence)
type PreviewRequest struct {
	TournamentID uint64               `json:"tournament_id"`
	Format       string               `json:"format"` // "single_elimination" or "double_elimination"
	Participants []domain.Participant `json:"participants"`
}

// Preview generates a bracket preview without persisting to the database.
// This uses the same engine logic as Generate, ensuring preview and actual bracket have identical BYE handling.
func (h *BracketHandler) Preview(w http.ResponseWriter, r *http.Request) {
	var req PreviewRequest
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

	var matches []*domain.Match
	var err error

	switch req.Format {
	case "single_elimination":
		matches, err = engine.SingleElimination(req.TournamentID, req.Participants)
	case "double_elimination":
		matches, err = engine.DoubleElimination(req.TournamentID, req.Participants)
	default:
		writeError(w, http.StatusBadRequest, "unsupported format: must be 'single_elimination' or 'double_elimination'")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Advance BYE winners to their next round matches (for preview display)
	advanceByeWinnersForPreview(matches)

	// Build response without DB IDs (preview only)
	resp := toPreviewResponse(req.TournamentID, matches, req.Format)
	json.NewEncoder(w).Encode(resp)
}

// advanceByeWinnersForPreview advances BYE winners to their next round matches in-memory.
// This is for preview display only - it shows where R1 BYE winners will appear in round 2.
// We only advance R1 BYEs because R2+ matches are real matches waiting for participants.
func advanceByeWinnersForPreview(matches []*domain.Match) {
	// Build a map of matches by bracket type, round, and position for quick lookup
	matchMap := make(map[string]*domain.Match)
	for _, m := range matches {
		key := fmt.Sprintf("%s-%d-%d", m.BracketType, m.Round, m.Position)
		matchMap[key] = m
	}

	// Find and process BYE matches in winners bracket round 1 only
	for _, m := range matches {
		if m.BracketType != domain.BracketWinners {
			continue
		}
		if m.Round != 1 {
			continue
		}
		if m.Status != domain.MatchCompleted {
			continue
		}
		// Check if this is a BYE match (one participant missing)
		if (m.Participant1ID == nil) == (m.Participant2ID == nil) {
			continue // Either both present or both missing - not a BYE
		}

		// Determine the winner (the non-nil participant)
		var winnerID *uint64
		var winnerName *string
		var winnerIconURL *string
		var winnerSeed *int

		if m.Participant1ID != nil {
			winnerID = m.Participant1ID
			winnerName = m.Participant1Name
			winnerIconURL = m.Participant1IconURL
			winnerSeed = m.Seed1
		} else {
			winnerID = m.Participant2ID
			winnerName = m.Participant2Name
			winnerIconURL = m.Participant2IconURL
			winnerSeed = m.Seed2
		}

		// Find next round match: position in R2 = ceil(position / 2)
		nextPosition := (m.Position + 1) / 2
		nextKey := fmt.Sprintf("%s-%d-%d", domain.BracketWinners, 2, nextPosition)
		nextMatch := matchMap[nextKey]
		if nextMatch == nil {
			continue
		}

		// Determine slot: odd position -> slot 1, even position -> slot 2
		if m.Position%2 == 1 {
			nextMatch.Participant1ID = winnerID
			nextMatch.Participant1Name = winnerName
			nextMatch.Participant1IconURL = winnerIconURL
			nextMatch.Seed1 = winnerSeed
		} else {
			nextMatch.Participant2ID = winnerID
			nextMatch.Participant2Name = winnerName
			nextMatch.Participant2IconURL = winnerIconURL
			nextMatch.Seed2 = winnerSeed
		}
	}
}

// toPreviewResponse converts generated matches to a preview response format.
// Preview matches don't have DB IDs but include all BYE information.
func toPreviewResponse(tournamentID uint64, matches []*domain.Match, format string) *BracketResponse {
	matchResponses := make([]*MatchResponse, len(matches))
	totalRounds := 0

	for i, m := range matches {
		matchResponses[i] = &MatchResponse{
			ID:                  0, // No DB ID for preview
			Round:               m.Round,
			Position:            m.Position,
			BracketType:         string(m.BracketType),
			Participant1ID:      m.Participant1ID,
			Participant2ID:      m.Participant2ID,
			Participant1Name:    m.Participant1Name,
			Participant2Name:    m.Participant2Name,
			Participant1IconURL: m.Participant1IconURL,
			Participant2IconURL: m.Participant2IconURL,
			Seed1:               m.Seed1,
			Seed2:               m.Seed2,
			Sets:                []SetResponse{},
			WinnerID:            m.WinnerID,
			Status:              string(m.Status),
		}

		// Track total rounds in winners bracket
		if m.BracketType == domain.BracketWinners && m.Round > totalRounds {
			totalRounds = m.Round
		}
	}

	// Generate preview stages with default names
	var stages []*StageResponse

	if format == "double_elimination" {
		losersRounds := engine.LosersRounds(totalRounds)

		// Winners bracket stages
		for round := 1; round <= totalRounds; round++ {
			stages = append(stages, &StageResponse{
				TournamentID: tournamentID,
				BracketType:  string(domain.BracketWinners),
				Round:        round,
				StageName:    domain.DefaultStageName(domain.BracketWinners, round, totalRounds),
				BestOf:       1,
			})
		}

		// Losers bracket stages
		for round := 1; round <= losersRounds; round++ {
			stages = append(stages, &StageResponse{
				TournamentID: tournamentID,
				BracketType:  string(domain.BracketLosers),
				Round:        round,
				StageName:    domain.DefaultStageName(domain.BracketLosers, round, losersRounds),
				BestOf:       1,
			})
		}

		// Grand final stage
		stages = append(stages, &StageResponse{
			TournamentID: tournamentID,
			BracketType:  string(domain.BracketGrandFinal),
			Round:        1,
			StageName:    domain.DefaultStageName(domain.BracketGrandFinal, 1, 1),
			BestOf:       1,
		})
	} else {
		// Single elimination - use empty bracket type for simple names (Final, Semifinals, etc.)
		for round := 1; round <= totalRounds; round++ {
			stages = append(stages, &StageResponse{
				TournamentID: tournamentID,
				BracketType:  string(domain.BracketWinners),
				Round:        round,
				StageName:    domain.DefaultStageName("", round, totalRounds),
				BestOf:       1,
			})
		}
	}

	resp := &BracketResponse{
		TournamentID: tournamentID,
		Format:       format,
		TotalRounds:  totalRounds,
		CurrentRound: 1,
		IsComplete:   false,
		Matches:      matchResponses,
		Stages:       stages,
	}

	// For double elimination, calculate winners and losers rounds
	if format == "double_elimination" {
		resp.WinnersRounds = totalRounds
		resp.LosersRounds = engine.LosersRounds(totalRounds)
	}

	return resp
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

	var state *service.BracketState
	var err error

	switch req.Format {
	case "single_elimination":
		state, err = h.bracketSvc.GenerateSingleElimination(r.Context(), req.TournamentID, req.Participants)
	case "double_elimination":
		state, err = h.bracketSvc.GenerateDoubleElimination(r.Context(), req.TournamentID, req.Participants)
	case "swiss":
		if h.swissSvc == nil {
			writeError(w, http.StatusInternalServerError, "swiss format not configured")
			return
		}
		swissState, swissErr := h.swissSvc.InitializeSwiss(r.Context(), req.TournamentID, req.Participants, req.SwissRounds)
		if swissErr != nil {
			writeError(w, http.StatusInternalServerError, swissErr.Error())
			return
		}
		resp := toSwissBracketResponse(swissState)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
		return
	default:
		writeError(w, http.StatusBadRequest, "unsupported format: must be 'single_elimination', 'double_elimination', or 'swiss'")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := toBracketResponseWithFormat(state, nil, req.Format)
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

	// Load stages
	stages, err := h.stageRepo.GetByTournament(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := toBracketResponseWithStages(state, stages)
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
	return toBracketResponseWithFormat(state, nil, "single_elimination")
}

func toBracketResponseWithStages(state *service.BracketState, stages []*domain.BracketStage) *BracketResponse {
	// Detect format from matches
	format := "single_elimination"
	for _, m := range state.Matches {
		if m.BracketType == domain.BracketLosers || m.BracketType == domain.BracketGrandFinal {
			format = "double_elimination"
			break
		}
	}
	return toBracketResponseWithFormat(state, stages, format)
}

func toBracketResponseWithFormat(state *service.BracketState, stages []*domain.BracketStage, format string) *BracketResponse {
	matches := make([]*MatchResponse, len(state.Matches))
	for i, m := range state.Matches {
		matches[i] = toMatchResponse(m)
	}

	stageResponses := make([]*StageResponse, len(stages))
	for i, s := range stages {
		stageName := ""
		if s.StageName != nil {
			stageName = *s.StageName
		}
		stageResponses[i] = &StageResponse{
			TournamentID: s.TournamentID,
			BracketType:  string(s.BracketType),
			Round:        s.Round,
			StageName:    stageName,
			BestOf:       s.BestOf,
		}
	}

	resp := &BracketResponse{
		TournamentID: state.TournamentID,
		Format:       format,
		TotalRounds:  state.TotalRounds,
		CurrentRound: state.CurrentRound,
		IsComplete:   state.IsComplete,
		ChampionID:   state.ChampionID,
		Matches:      matches,
		Stages:       stageResponses,
	}

	// For double elimination, calculate winners and losers rounds
	if format == "double_elimination" {
		resp.WinnersRounds = state.TotalRounds
		resp.LosersRounds = engine.LosersRounds(state.TotalRounds)
	}

	return resp
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
		ID:                  m.ID,
		Round:               m.Round,
		Position:            m.Position,
		BracketType:         string(m.BracketType),
		Participant1ID:      m.Participant1ID,
		Participant2ID:      m.Participant2ID,
		Participant1Name:    m.Participant1Name,
		Participant2Name:    m.Participant2Name,
		Participant1IconURL: m.Participant1IconURL,
		Participant2IconURL: m.Participant2IconURL,
		Seed1:               m.Seed1,
		Seed2:               m.Seed2,
		Sets:                sets,
		Participant1Sets:    p1Sets,
		Participant2Sets:    p2Sets,
		WinnerID:            m.WinnerID,
		ForfeitWinnerID:     m.ForfeitWinnerID,
		Status:              string(m.Status),
		NextMatchID:         m.NextMatchID,
		LoserMatchID:        m.LoserMatchID,
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Delete removes all bracket data for a tournament (matches, sets, stages) and reverts ELO.
// This is used when resetting a tournament back to registration phase.
func (h *BracketHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := strconv.ParseUint(chi.URLParam(r, "tournamentId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	if err := h.bracketSvc.DeleteBracket(r.Context(), tournamentID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Swiss-specific response types

type SwissBracketResponse struct {
	TournamentID        uint64                  `json:"tournament_id"`
	Format              string                  `json:"format"`
	TotalRounds         int                     `json:"total_rounds"`
	CurrentRound        int                     `json:"current_round"`
	IsComplete          bool                    `json:"is_complete"`
	HasTiebreakers      bool                    `json:"has_tiebreakers"`
	TiebreakersComplete bool                    `json:"tiebreakers_complete"`
	Standings           []SwissStandingResponse `json:"standings"`
	Matches             []*MatchResponse        `json:"matches"`
	Stages              []*StageResponse        `json:"stages"`
	Tiebreakers         []TiebreakerResponse    `json:"tiebreakers,omitempty"`
}

type TiebreakerResponse struct {
	ID             uint64           `json:"id"`
	TiePosition    int              `json:"tie_position"`
	ParticipantIDs []uint64         `json:"participant_ids"`
	IsComplete     bool             `json:"is_complete"`
	Matches        []*MatchResponse `json:"matches,omitempty"`
}

type SwissStandingResponse struct {
	Rank            int     `json:"rank"`
	ParticipantID   uint64  `json:"participant_id"`
	ParticipantName string  `json:"participant_name"`
	IconURL         *string `json:"icon_url,omitempty"`
	Wins            int     `json:"wins"`
	Losses          int     `json:"losses"`
	GameWins        int     `json:"game_wins"`
	GameLosses      int     `json:"game_losses"`
	OpponentWins    int     `json:"opponent_wins"`
}

// EliminationStandingResponse represents standings for single/double elimination
type EliminationStandingResponse struct {
	Rank            int     `json:"rank"`
	ParticipantID   uint64  `json:"participant_id"`
	ParticipantName string  `json:"participant_name"`
	IconURL         *string `json:"icon_url,omitempty"`
	Seed            int     `json:"seed"`
	Wins            int     `json:"wins"`
	Losses          int     `json:"losses"`
	GameWins        int     `json:"game_wins"`
	GameLosses      int     `json:"game_losses"`
	Placement       string  `json:"placement"`
}

// EliminationStandingsResponse is the API response for elimination standings
type EliminationStandingsResponse struct {
	TournamentID uint64                        `json:"tournament_id"`
	Format       string                        `json:"format"`
	IsComplete   bool                          `json:"is_complete"`
	Standings    []EliminationStandingResponse `json:"standings"`
}

func toSwissBracketResponse(state *domain.SwissBracketState) *SwissBracketResponse {
	resp := &SwissBracketResponse{
		TournamentID: state.TournamentID,
		Format:       "swiss",
		Standings:    make([]SwissStandingResponse, 0),
		Matches:      make([]*MatchResponse, 0),
		Stages:       make([]*StageResponse, 0),
		Tiebreakers:  make([]TiebreakerResponse, 0),
	}

	if state.Config != nil {
		resp.TotalRounds = state.Config.TotalRounds
		resp.CurrentRound = state.Config.CurrentRound
		resp.IsComplete = state.Config.IsComplete
		resp.HasTiebreakers = state.Config.HasTiebreakers
		resp.TiebreakersComplete = state.Config.TiebreakersComplete
	}

	for i, s := range state.Standings {
		resp.Standings = append(resp.Standings, SwissStandingResponse{
			Rank:            i + 1,
			ParticipantID:   s.ParticipantID,
			ParticipantName: s.ParticipantName,
			IconURL:         s.ParticipantIconURL,
			Wins:            s.Wins,
			Losses:          s.Losses,
			GameWins:        s.GameWins,
			GameLosses:      s.GameLosses,
			OpponentWins:    s.OpponentWins,
		})
	}

	// Separate tiebreaker matches from regular Swiss matches
	tiebreakerMatches := make(map[int][]*MatchResponse) // tiePosition -> matches
	for _, m := range state.Matches {
		matchResp := toMatchResponse(m)
		if m.BracketType == domain.BracketTiebreaker {
			tiebreakerMatches[m.Round] = append(tiebreakerMatches[m.Round], matchResp)
		} else {
			resp.Matches = append(resp.Matches, matchResp)
		}
	}

	for _, s := range state.Stages {
		stageName := ""
		if s.StageName != nil {
			stageName = *s.StageName
		}
		resp.Stages = append(resp.Stages, &StageResponse{
			TournamentID: s.TournamentID,
			BracketType:  string(s.BracketType),
			Round:        s.Round,
			StageName:    stageName,
			BestOf:       s.BestOf,
		})
	}

	// Add tiebreakers to response
	for _, tb := range state.Tiebreakers {
		tbResp := TiebreakerResponse{
			ID:             tb.ID,
			TiePosition:    tb.TiePosition,
			ParticipantIDs: tb.ParticipantIDs,
			IsComplete:     tb.IsComplete,
			Matches:        tiebreakerMatches[tb.TiePosition],
		}
		resp.Tiebreakers = append(resp.Tiebreakers, tbResp)
	}

	return resp
}

// GetSwissStandings returns the current Swiss standings for a tournament
func (h *BracketHandler) GetSwissStandings(w http.ResponseWriter, r *http.Request) {
	if h.swissSvc == nil {
		writeError(w, http.StatusInternalServerError, "swiss format not configured")
		return
	}

	tournamentID, err := strconv.ParseUint(chi.URLParam(r, "tournamentId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	state, err := h.swissSvc.GetState(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Load sets for all matches
	if len(state.Matches) > 0 {
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
	}

	resp := toSwissBracketResponse(state)
	json.NewEncoder(w).Encode(resp)
}

// AdvanceSwissRound manually triggers the next round of a Swiss tournament
func (h *BracketHandler) AdvanceSwissRound(w http.ResponseWriter, r *http.Request) {
	if h.swissSvc == nil {
		writeError(w, http.StatusInternalServerError, "swiss format not configured")
		return
	}

	tournamentID, err := strconv.ParseUint(chi.URLParam(r, "tournamentId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	matches, err := h.swissSvc.AdvanceRound(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if matches == nil {
		// Tournament is complete
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Return the new matches
	matchResponses := make([]*MatchResponse, len(matches))
	for i, m := range matches {
		matchResponses[i] = toMatchResponse(m)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"matches": matchResponses,
	})
}

// GetEliminationStandings returns standings for single/double elimination brackets
func (h *BracketHandler) GetEliminationStandings(w http.ResponseWriter, r *http.Request) {
	tournamentID, err := strconv.ParseUint(chi.URLParam(r, "tournamentId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tournament ID")
		return
	}

	standings, err := h.matchSvc.GetEliminationStandings(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get bracket state to determine format and completion status
	state, err := h.matchSvc.GetBracketState(r.Context(), tournamentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Determine format from matches
	format := "single_elimination"
	for _, m := range state.Matches {
		if m.BracketType == domain.BracketLosers || m.BracketType == domain.BracketGrandFinal {
			format = "double_elimination"
			break
		}
	}

	// Build response
	standingResponses := make([]EliminationStandingResponse, len(standings))
	for i, s := range standings {
		standingResponses[i] = EliminationStandingResponse{
			Rank:            s.Rank,
			ParticipantID:   s.ParticipantID,
			ParticipantName: s.ParticipantName,
			IconURL:         s.IconURL,
			Seed:            s.Seed,
			Wins:            s.Wins,
			Losses:          s.Losses,
			GameWins:        s.GameWins,
			GameLosses:      s.GameLosses,
			Placement:       s.Placement,
		}
	}

	resp := EliminationStandingsResponse{
		TournamentID: tournamentID,
		Format:       format,
		IsComplete:   state.IsComplete,
		Standings:    standingResponses,
	}

	json.NewEncoder(w).Encode(resp)
}
