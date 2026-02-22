package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CommunityClient interface {
	ProcessMatchElo(ctx context.Context, req ProcessMatchEloRequest) (*ProcessMatchEloResponse, error)
	GetEloSystem(ctx context.Context, systemID uint64) (*EloSystemResponse, error)
}

type ProcessMatchEloRequest struct {
	EloSystemID    uint64 `json:"elo_system_id"`
	MatchID        uint64 `json:"match_id"`
	TournamentID   uint64 `json:"tournament_id"`
	WinnerMemberID uint64 `json:"winner_member_id"`
	LoserMemberID  uint64 `json:"loser_member_id"`
}

type ProcessMatchEloResponse struct {
	WinnerRatingBefore int `json:"winner_rating_before"`
	WinnerRatingAfter  int `json:"winner_rating_after"`
	WinnerChange       int `json:"winner_change"`
	LoserRatingBefore  int `json:"loser_rating_before"`
	LoserRatingAfter   int `json:"loser_rating_after"`
	LoserChange        int `json:"loser_change"`
}

type EloSystemResponse struct {
	ID             uint64 `json:"id"`
	CommunityID    uint64 `json:"community_id"`
	Name           string `json:"name"`
	StartingRating int    `json:"starting_rating"`
	KFactor        int    `json:"k_factor"`
	IsActive       bool   `json:"is_active"`
}

type communityClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewCommunityClient(baseURL string) CommunityClient {
	return &communityClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ProcessMatchElo sends a match result to the community service for ELO processing
func (c *communityClient) ProcessMatchElo(ctx context.Context, req ProcessMatchEloRequest) (*ProcessMatchEloResponse, error) {
	url := fmt.Sprintf("%s/internal/elo/process-match", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call community service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var result ProcessMatchEloResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetEloSystem fetches an ELO system by ID
func (c *communityClient) GetEloSystem(ctx context.Context, systemID uint64) (*EloSystemResponse, error) {
	url := fmt.Sprintf("%s/internal/elo/systems/%d", c.baseURL, systemID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call community service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("elo system not found")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var system EloSystemResponse
	if err := json.NewDecoder(resp.Body).Decode(&system); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &system, nil
}
