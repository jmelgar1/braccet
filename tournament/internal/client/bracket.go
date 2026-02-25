package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type BracketClient interface {
	ProcessWithdrawal(ctx context.Context, tournamentID, participantID uint64) error
	IsBracketComplete(ctx context.Context, tournamentID uint64) (bool, error)
	IsStageBracketComplete(ctx context.Context, tournamentID, stageID uint64) (bool, error)
	IsGroupBracketComplete(ctx context.Context, tournamentID, stageID, groupID uint64) (bool, error)
	DeleteBracket(ctx context.Context, tournamentID uint64) error
	CreateBracket(ctx context.Context, req CreateBracketRequest) error
	CreateGroupBracket(ctx context.Context, req CreateGroupBracketRequest) error
	CompleteStage(ctx context.Context, req CompleteStageRequest) (*CompleteStageResponse, error)
	GetStageAdvancingParticipants(ctx context.Context, tournamentID, stageID uint64) ([]AdvancingParticipant, error)
}

// CreateBracketRequest contains the parameters to create a single-stage bracket
type CreateBracketRequest struct {
	TournamentID uint64        `json:"tournament_id"`
	Format       string        `json:"format"` // "single_elimination", "double_elimination", or "swiss"
	Participants []Participant `json:"participants"`
	SwissRounds  *int          `json:"swiss_rounds,omitempty"` // Optional override for Swiss round count
}

// CreateGroupBracketRequest contains the parameters to create a bracket for a group
type CreateGroupBracketRequest struct {
	TournamentID      uint64        `json:"tournament_id"`
	StageID           uint64        `json:"stage_id"`
	GroupID           uint64        `json:"group_id"`
	Format            string        `json:"format"`
	Participants      []Participant `json:"participants"`
	SwissRounds       *int          `json:"swiss_rounds,omitempty"`
	WinsToAdvance     *int          `json:"wins_to_advance,omitempty"`     // Swiss threshold: wins needed to advance
	LossesToEliminate *int          `json:"losses_to_eliminate,omitempty"` // Swiss threshold: losses that eliminate
	SkipFinals        bool          `json:"skip_finals"`
}

// Participant represents a participant in a bracket request
type Participant struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"icon_url,omitempty"`
	Seed    int    `json:"seed"`
}

// CompleteStageRequest contains parameters to complete a stage and get advancing participants
type CompleteStageRequest struct {
	TournamentID      uint64   `json:"tournament_id"`
	StageID           uint64   `json:"stage_id"`
	GroupIDs          []uint64 `json:"group_ids"`
	AdvancingPerGroup int      `json:"advancing_per_group"`
	RankingCriteria   []string `json:"ranking_criteria,omitempty"`
	Format            string   `json:"format"`
	IsThresholdSwiss  bool     `json:"is_threshold_swiss,omitempty"` // True if this is a threshold-based Swiss stage
}

// CompleteStageResponse contains the results of completing a stage
type CompleteStageResponse struct {
	StageStandings        []StageStanding        `json:"stage_standings"`
	AdvancingParticipants []AdvancingParticipant `json:"advancing_participants"`
}

// StageStanding represents a participant's standing in a completed stage
type StageStanding struct {
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

// AdvancingParticipant represents a participant who advances to the next stage
type AdvancingParticipant struct {
	ParticipantID   uint64  `json:"participant_id"`
	ParticipantName string  `json:"participant_name"`
	IconURL         *string `json:"icon_url,omitempty"`
	StageRank       int     `json:"stage_rank"`
	GroupID         uint64  `json:"group_id"`
	GroupRank       int     `json:"group_rank"`
}

type bracketClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewBracketClient(baseURL string) BracketClient {
	return &bracketClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type forfeitRequest struct {
	TournamentID  uint64 `json:"tournament_id"`
	ParticipantID uint64 `json:"participant_id"`
}

// ProcessWithdrawal notifies the bracket service to forfeit matches for a withdrawn participant.
func (c *bracketClient) ProcessWithdrawal(ctx context.Context, tournamentID, participantID uint64) error {
	req := forfeitRequest{
		TournamentID:  tournamentID,
		ParticipantID: participantID,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/brackets/forfeit-participant", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call bracket service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("bracket service returned status %d", resp.StatusCode)
	}

	return nil
}

type bracketStateResponse struct {
	IsComplete bool `json:"is_complete"`
}

// IsBracketComplete checks if all matches in the bracket are completed.
func (c *bracketClient) IsBracketComplete(ctx context.Context, tournamentID uint64) (bool, error) {
	url := fmt.Sprintf("%s/brackets/%d", c.baseURL, tournamentID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("failed to call bracket service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, fmt.Errorf("bracket not found for tournament")
	}

	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("bracket service returned status %d", resp.StatusCode)
	}

	var state bracketStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return state.IsComplete, nil
}

// IsStageBracketComplete checks if all matches in a specific stage's bracket are completed.
// This is used for multi-stage tournaments to validate stage completion before advancing.
func (c *bracketClient) IsStageBracketComplete(ctx context.Context, tournamentID, stageID uint64) (bool, error) {
	url := fmt.Sprintf("%s/brackets/%d/stages/%d", c.baseURL, tournamentID, stageID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("failed to call bracket service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Stage bracket doesn't exist yet - not complete
		return false, nil
	}

	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("bracket service returned status %d", resp.StatusCode)
	}

	var state bracketStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return state.IsComplete, nil
}

// IsGroupBracketComplete checks if all matches in a specific group's bracket are completed.
// This is used for multi-stage tournaments with group stages.
func (c *bracketClient) IsGroupBracketComplete(ctx context.Context, tournamentID, stageID, groupID uint64) (bool, error) {
	url := fmt.Sprintf("%s/brackets/%d/stages/%d/groups/%d", c.baseURL, tournamentID, stageID, groupID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("failed to call bracket service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Group bracket doesn't exist yet - not complete
		return false, nil
	}

	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("bracket service returned status %d", resp.StatusCode)
	}

	var state bracketStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return state.IsComplete, nil
}

// DeleteBracket deletes all bracket data for a tournament (matches, sets, stages) and reverts ELO.
// This is used when resetting a tournament back to registration phase.
func (c *bracketClient) DeleteBracket(ctx context.Context, tournamentID uint64) error {
	url := fmt.Sprintf("%s/brackets/%d", c.baseURL, tournamentID)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call bracket service: %w", err)
	}
	defer resp.Body.Close()

	// 204 No Content is success, 404 is also acceptable (bracket may not exist)
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("bracket service returned status %d", resp.StatusCode)
	}

	return nil
}

// CreateBracket creates a bracket for a single-stage tournament.
// This is called when starting a non-multi-stage tournament.
func (c *bracketClient) CreateBracket(ctx context.Context, req CreateBracketRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/brackets", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call bracket service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Read error body for more details
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return fmt.Errorf("bracket service error: %s", errResp.Error)
		}
		return fmt.Errorf("bracket service returned status %d", resp.StatusCode)
	}

	return nil
}

// CreateGroupBracket creates a bracket for a specific group within a stage.
// This is called when starting a multi-stage tournament to generate matches for each group.
func (c *bracketClient) CreateGroupBracket(ctx context.Context, req CreateGroupBracketRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/brackets/groups", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call bracket service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("bracket service returned status %d", resp.StatusCode)
	}

	return nil
}

// CompleteStage finalizes a stage and returns the advancing participants.
// This is called when advancing to the next stage in a multi-stage tournament.
func (c *bracketClient) CompleteStage(ctx context.Context, req CompleteStageRequest) (*CompleteStageResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/brackets/%d/stages/%d/complete", c.baseURL, req.TournamentID, req.StageID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call bracket service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("bracket service returned status %d", resp.StatusCode)
	}

	var result CompleteStageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetStageAdvancingParticipants returns participants who advanced from a completed stage.
// This queries the stage_standings table for participants with advances=true.
func (c *bracketClient) GetStageAdvancingParticipants(ctx context.Context, tournamentID, stageID uint64) ([]AdvancingParticipant, error) {
	url := fmt.Sprintf("%s/brackets/%d/stages/%d/advancing", c.baseURL, tournamentID, stageID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call bracket service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Stage not completed yet or no standings
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("bracket service returned status %d", resp.StatusCode)
	}

	var result struct {
		AdvancingParticipants []AdvancingParticipant `json:"advancing_participants"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.AdvancingParticipants, nil
}
