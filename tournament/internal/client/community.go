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
	FindOrCreateGhostMember(ctx context.Context, communityID uint64, displayName string) (*MemberResponse, error)
	DeleteMembers(ctx context.Context, communityID uint64, memberIDs []uint64) (int64, error)
	GetBulkMemberRatings(ctx context.Context, eloSystemID uint64, memberIDs []uint64) (map[uint64]int, error)
	GetBulkMemberIcons(ctx context.Context, memberIDs []uint64) (map[uint64]string, error)
	GetBulkMemberData(ctx context.Context, memberIDs []uint64) (map[uint64]MemberDataResponse, error)
	SearchMembers(ctx context.Context, communityID uint64, query string, excludeIDs []uint64) ([]MemberSearchResult, error)
}

// MemberDataResponse holds icon and region data from bulk fetch
type MemberDataResponse struct {
	IconURL *string
	Region  *string
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
	Region      *string `json:"region"`
}

type MemberSearchResult struct {
	ID          uint64 `json:"id"`
	DisplayName string `json:"display_name"`
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

// FindOrCreateGhostMember finds an existing member by display_name or creates a new ghost member
func (c *communityClient) FindOrCreateGhostMember(ctx context.Context, communityID uint64, displayName string) (*MemberResponse, error) {
	url := fmt.Sprintf("%s/internal/communities/%d/members/find-or-create", c.baseURL, communityID)

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

// DeleteMembers deletes multiple community members (internal endpoint for cleanup)
func (c *communityClient) DeleteMembers(ctx context.Context, communityID uint64, memberIDs []uint64) (int64, error) {
	if len(memberIDs) == 0 {
		return 0, nil
	}

	url := fmt.Sprintf("%s/internal/communities/%d/members/delete", c.baseURL, communityID)

	reqBody := map[string][]uint64{"member_ids": memberIDs}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("failed to call community service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var result struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Deleted, nil
}

// GetBulkMemberRatings fetches ELO ratings for multiple members in a specific system
// Returns a map of memberID -> rating
func (c *communityClient) GetBulkMemberRatings(ctx context.Context, eloSystemID uint64, memberIDs []uint64) (map[uint64]int, error) {
	if len(memberIDs) == 0 {
		return make(map[uint64]int), nil
	}

	url := fmt.Sprintf("%s/internal/elo/bulk-ratings", c.baseURL)

	reqBody := map[string]any{
		"elo_system_id": eloSystemID,
		"member_ids":    memberIDs,
	}
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

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var ratings []struct {
		MemberID uint64 `json:"member_id"`
		Rating   int    `json:"rating"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ratings); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make(map[uint64]int, len(ratings))
	for _, r := range ratings {
		result[r.MemberID] = r.Rating
	}

	return result, nil
}

// GetBulkMemberIcons fetches icon URLs for multiple members
// Returns a map of memberID -> iconURL
func (c *communityClient) GetBulkMemberIcons(ctx context.Context, memberIDs []uint64) (map[uint64]string, error) {
	if len(memberIDs) == 0 {
		return make(map[uint64]string), nil
	}

	url := fmt.Sprintf("%s/internal/members/bulk-icons", c.baseURL)

	reqBody := map[string]any{
		"member_ids": memberIDs,
	}
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

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var icons []struct {
		MemberID uint64 `json:"member_id"`
		IconURL  string `json:"icon_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&icons); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make(map[uint64]string, len(icons))
	for _, i := range icons {
		result[i.MemberID] = i.IconURL
	}

	return result, nil
}

// GetBulkMemberData fetches icon URLs and regions for multiple members
// Returns a map of memberID -> MemberDataResponse
func (c *communityClient) GetBulkMemberData(ctx context.Context, memberIDs []uint64) (map[uint64]MemberDataResponse, error) {
	if len(memberIDs) == 0 {
		return make(map[uint64]MemberDataResponse), nil
	}

	url := fmt.Sprintf("%s/internal/members/bulk-data", c.baseURL)

	reqBody := map[string]any{
		"member_ids": memberIDs,
	}
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

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var data []struct {
		MemberID uint64  `json:"member_id"`
		IconURL  *string `json:"icon_url"`
		Region   *string `json:"region"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := make(map[uint64]MemberDataResponse, len(data))
	for _, d := range data {
		result[d.MemberID] = MemberDataResponse{
			IconURL: d.IconURL,
			Region:  d.Region,
		}
	}

	return result, nil
}

// SearchMembers searches for community members by display name prefix, excluding specified IDs
func (c *communityClient) SearchMembers(ctx context.Context, communityID uint64, query string, excludeIDs []uint64) ([]MemberSearchResult, error) {
	url := fmt.Sprintf("%s/internal/communities/%d/members/search", c.baseURL, communityID)

	reqBody := map[string]any{
		"query":       query,
		"exclude_ids": excludeIDs,
		"limit":       10,
	}
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

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("community service returned status %d", resp.StatusCode)
	}

	var results []MemberSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return results, nil
}
