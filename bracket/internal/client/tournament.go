package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type TournamentClient interface {
	GetTournament(ctx context.Context, id uint64) (*TournamentResponse, error)
	GetParticipant(ctx context.Context, id uint64) (*ParticipantResponse, error)

	// Stage reseed operations
	GetStageGroups(ctx context.Context, stageID uint64) ([]StageGroupResponse, error)
	GetStageParticipants(ctx context.Context, stageID uint64) ([]StageParticipantResponse, error)
	UpdateStageAssignments(ctx context.Context, stageID uint64, assignments []AssignmentUpdateRequest) error

	// Event advancement
	TriggerAdvancement(ctx context.Context, tournamentID uint64) error
}

// StageGroupResponse represents a group in a stage
type StageGroupResponse struct {
	ID         uint64 `json:"id"`
	GroupName  string `json:"group_name"`
	GroupOrder int    `json:"group_order"`
}

// StageParticipantResponse represents a participant in a stage with their global seed
type StageParticipantResponse struct {
	ParticipantID uint64  `json:"participant_id"`
	Name          string  `json:"name"`
	IconURL       *string `json:"icon_url,omitempty"`
	GlobalSeed    int     `json:"global_seed"`
}

// AssignmentUpdateRequest represents a single group assignment update
type AssignmentUpdateRequest struct {
	GroupID       uint64 `json:"group_id"`
	ParticipantID uint64 `json:"participant_id"`
	SeedInGroup   int    `json:"seed_in_group"`
}

type TournamentResponse struct {
	ID          uint64  `json:"id"`
	Slug        string  `json:"slug"`
	OrganizerID uint64  `json:"organizer_id"`
	CommunityID *uint64 `json:"community_id,omitempty"`
	EloSystemID *uint64 `json:"elo_system_id,omitempty"`
	PRSystemID  *uint64 `json:"pr_system_id,omitempty"`
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

// GetStageGroups fetches all groups for a stage from the tournament service
func (c *tournamentClient) GetStageGroups(ctx context.Context, stageID uint64) ([]StageGroupResponse, error) {
	url := fmt.Sprintf("%s/internal/stages/%d/groups", c.baseURL, stageID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call tournament service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tournament service returned status %d", resp.StatusCode)
	}

	var groups []StageGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return groups, nil
}

// GetStageParticipants fetches all participants for a stage with their global seeds
func (c *tournamentClient) GetStageParticipants(ctx context.Context, stageID uint64) ([]StageParticipantResponse, error) {
	url := fmt.Sprintf("%s/internal/stages/%d/participants", c.baseURL, stageID)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call tournament service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tournament service returned status %d", resp.StatusCode)
	}

	var participants []StageParticipantResponse
	if err := json.NewDecoder(resp.Body).Decode(&participants); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return participants, nil
}

// UpdateStageAssignments replaces all group assignments for a stage
func (c *tournamentClient) UpdateStageAssignments(ctx context.Context, stageID uint64, assignments []AssignmentUpdateRequest) error {
	url := fmt.Sprintf("%s/internal/stages/%d/assignments", c.baseURL, stageID)

	reqBody := struct {
		Assignments []AssignmentUpdateRequest `json:"assignments"`
	}{
		Assignments: assignments,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call tournament service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("tournament service returned status %d", resp.StatusCode)
	}

	return nil
}

// TriggerAdvancement triggers automatic advancement for a completed qualifier tournament
func (c *tournamentClient) TriggerAdvancement(ctx context.Context, tournamentID uint64) error {
	url := fmt.Sprintf("%s/internal/tournaments/%d/advance", c.baseURL, tournamentID)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call tournament service: %w", err)
	}
	defer resp.Body.Close()

	// 404 means tournament is not part of an event, which is fine
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("tournament service returned status %d", resp.StatusCode)
	}

	return nil
}
