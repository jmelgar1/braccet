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
	GetCommunity(ctx context.Context, communityID uint64) (*CommunityResponse, error)
	GetMember(ctx context.Context, communityID, memberID uint64) (*MemberResponse, error)
	CreateGhostMember(ctx context.Context, communityID uint64, displayName string) (*MemberResponse, error)
}

type CommunityResponse struct {
	ID      uint64 `json:"id"`
	OwnerID uint64 `json:"owner_id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
}

type MemberResponse struct {
	ID          uint64  `json:"id"`
	CommunityID uint64  `json:"community_id"`
	UserID      *uint64 `json:"user_id"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
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

// GetCommunity fetches a community by ID from the community service (internal endpoint)
func (c *communityClient) GetCommunity(ctx context.Context, communityID uint64) (*CommunityResponse, error) {
	url := fmt.Sprintf("%s/internal/communities/%d", c.baseURL, communityID)
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
		return nil, fmt.Errorf("community not found")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var community CommunityResponse
	if err := json.NewDecoder(resp.Body).Decode(&community); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &community, nil
}

// GetMember fetches a community member by ID from the community service (internal endpoint)
func (c *communityClient) GetMember(ctx context.Context, communityID, memberID uint64) (*MemberResponse, error) {
	url := fmt.Sprintf("%s/internal/communities/%d/members/%d", c.baseURL, communityID, memberID)
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
		return nil, fmt.Errorf("member not found")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var member MemberResponse
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &member, nil
}

// CreateGhostMember creates a ghost member (no user_id) in the community service
func (c *communityClient) CreateGhostMember(ctx context.Context, communityID uint64, displayName string) (*MemberResponse, error) {
	url := fmt.Sprintf("%s/internal/communities/%d/members", c.baseURL, communityID)

	reqBody := map[string]string{"display_name": displayName}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call community service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("community not found")
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var member MemberResponse
	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &member, nil
}
