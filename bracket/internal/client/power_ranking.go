package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PowerRankingClient handles communication with the community service for power rankings
type PowerRankingClient interface {
	RecordPlacement(ctx context.Context, req RecordPlacementRequest) error
	RecordMatch(ctx context.Context, req RecordMatchRequest) error
	GetPRSystem(ctx context.Context, systemID uint64) (*PRSystemResponse, error)
	RevertTournamentPlacements(ctx context.Context, tournamentID uint64) error
}

type RecordPlacementRequest struct {
	PRSystemID        uint64   `json:"pr_system_id"`
	TournamentID      uint64   `json:"tournament_id"`
	MemberID          uint64   `json:"member_id"`
	Placement         int      `json:"placement"`
	ParticipantCount  int      `json:"participant_count"`
	TournamentClass   string   `json:"tournament_class"`
	PrizePoolUSD      *float64 `json:"prize_pool_usd,omitempty"`
	IsBo3Playoffs     bool     `json:"is_bo3_playoffs"`
	AvgOpponentRating *int     `json:"avg_opponent_rating,omitempty"`
	IsLAN             bool     `json:"is_lan"`
	CompletedAt       string   `json:"completed_at"`
}

type RecordMatchRequest struct {
	PRSystemID     uint64 `json:"pr_system_id"`
	MatchID        uint64 `json:"match_id"`
	TournamentID   uint64 `json:"tournament_id"`
	WinnerMemberID uint64 `json:"winner_member_id"`
	LoserMemberID  uint64 `json:"loser_member_id"`
	WinnerSetsWon  int    `json:"winner_sets_won"`
	LoserSetsWon   int    `json:"loser_sets_won"`
	WinnerRating   int    `json:"winner_rating"`
	LoserRating    int    `json:"loser_rating"`
	IsLAN          bool   `json:"is_lan"`
	CompletedAt    string `json:"completed_at"`
}

type PRSystemResponse struct {
	ID                      uint64 `json:"id"`
	CommunityID             uint64 `json:"community_id"`
	Name                    string `json:"name"`
	AchievementsWeight      int    `json:"achievements_weight"`
	FormWeight              int    `json:"form_weight"`
	LANWeight               int    `json:"lan_weight"`
	AchievementWindowMonths int    `json:"achievement_window_months"`
	FormWindowMonths        int    `json:"form_window_months"`
	LANResultsCount         int    `json:"lan_results_count"`
	IsActive                bool   `json:"is_active"`
}

type powerRankingClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPowerRankingClient(baseURL string) PowerRankingClient {
	return &powerRankingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// RecordPlacement records a tournament placement to the community service
func (c *powerRankingClient) RecordPlacement(ctx context.Context, req RecordPlacementRequest) error {
	url := fmt.Sprintf("%s/internal/power-rankings/placements", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call community service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	return nil
}

// RecordMatch records a match result for power ranking form tracking
func (c *powerRankingClient) RecordMatch(ctx context.Context, req RecordMatchRequest) error {
	url := fmt.Sprintf("%s/internal/power-rankings/matches", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call community service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	return nil
}

// GetPRSystem fetches a power ranking system by ID
func (c *powerRankingClient) GetPRSystem(ctx context.Context, systemID uint64) (*PRSystemResponse, error) {
	url := fmt.Sprintf("%s/internal/power-rankings/systems/%d", c.baseURL, systemID)
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
		return nil, fmt.Errorf("power ranking system not found")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var system PRSystemResponse
	if err := json.NewDecoder(resp.Body).Decode(&system); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &system, nil
}

// RevertTournamentPlacements reverts all power ranking data for a tournament
func (c *powerRankingClient) RevertTournamentPlacements(ctx context.Context, tournamentID uint64) error {
	url := fmt.Sprintf("%s/internal/power-rankings/tournament/%d", c.baseURL, tournamentID)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call community service: %w", err)
	}
	defer resp.Body.Close()

	// 204 No Content is success, 404 is also acceptable (no placements to revert)
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	return nil
}
