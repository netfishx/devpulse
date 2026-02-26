package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SupportedEventTypes lists the event types we process.
var SupportedEventTypes = map[string]bool{
	"PushEvent":              true,
	"PullRequestEvent":       true,
	"PullRequestReviewEvent": true,
	"CreateEvent":            true,
}

// Client calls the GitHub API.
type Client struct {
	httpClient *http.Client
	baseURL    string // for testing; defaults to "https://api.github.com"
}

// NewClient creates a new GitHub API client.
// If httpClient is nil, a default http.Client is used.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{httpClient: httpClient, baseURL: "https://api.github.com"}
}

// FetchUserEvents fetches all recent events for the authenticated user.
// GitHub returns max 10 pages of 30 events (300 total).
func (c *Client) FetchUserEvents(ctx context.Context, token string) ([]Event, error) {
	var allEvents []Event

	for page := 1; page <= 10; page++ {
		url := fmt.Sprintf("%s/user/events?per_page=30&page=%d", c.baseURL, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch events page %d: %w", page, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("github api returned %d", resp.StatusCode)
		}

		var events []Event
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			return nil, fmt.Errorf("decode events: %w", err)
		}

		allEvents = append(allEvents, events...)

		if len(events) < 30 {
			break // No more pages
		}
	}

	return allEvents, nil
}
