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
