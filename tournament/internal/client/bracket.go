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
	DeleteBracket(ctx context.Context, tournamentID uint64) error
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
