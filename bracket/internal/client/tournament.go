package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TournamentClient interface {
	GetTournament(ctx context.Context, id uint64) (*TournamentResponse, error)
	GetParticipant(ctx context.Context, id uint64) (*ParticipantResponse, error)
}

type TournamentResponse struct {
	ID          uint64  `json:"id"`
	Slug        string  `json:"slug"`
	OrganizerID uint64  `json:"organizer_id"`
	CommunityID *uint64 `json:"community_id,omitempty"`
	EloSystemID *uint64 `json:"elo_system_id,omitempty"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
}

type ParticipantResponse struct {
	ID                uint64  `json:"id"`
	TournamentID      uint64  `json:"tournament_id"`
	UserID            *uint64 `json:"user_id,omitempty"`
	CommunityMemberID *uint64 `json:"community_member_id,omitempty"`
	DisplayName       string  `json:"display_name"`
	Status            string  `json:"status"`
}

type tournamentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewTournamentClient(baseURL string) TournamentClient {
	return &tournamentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetTournament fetches a tournament by ID from the tournament service
func (c *tournamentClient) GetTournament(ctx context.Context, id uint64) (*TournamentResponse, error) {
	url := fmt.Sprintf("%s/internal/tournaments/%d", c.baseURL, id)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call tournament service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("tournament not found")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tournament service returned status %d", resp.StatusCode)
	}

	var tournament TournamentResponse
	if err := json.NewDecoder(resp.Body).Decode(&tournament); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tournament, nil
}

// GetParticipant fetches a participant by ID from the tournament service
func (c *tournamentClient) GetParticipant(ctx context.Context, id uint64) (*ParticipantResponse, error) {
	url := fmt.Sprintf("%s/internal/participants/%d", c.baseURL, id)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call tournament service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("participant not found")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tournament service returned status %d", resp.StatusCode)
	}

	var participant ParticipantResponse
	if err := json.NewDecoder(resp.Body).Decode(&participant); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &participant, nil
}
