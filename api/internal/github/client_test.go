package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient creates a Client pointed at the given test server URL.
func newTestClient(serverURL string) *Client {
	c := NewClient(nil)
	c.baseURL = serverURL
	return c
}

func TestFetchUserEvents_Success(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	events := []Event{
		{
			ID:        "1",
			Type:      "PushEvent",
			Repo:      Repo{Name: "user/repo-a"},
			CreatedAt: now,
			Payload: Payload{
				Commits: []Commit{
					{SHA: "abc123", Message: "initial commit"},
				},
				Size: 1,
			},
		},
		{
			ID:        "2",
			Type:      "PullRequestEvent",
			Repo:      Repo{Name: "user/repo-b"},
			CreatedAt: now,
			Payload: Payload{
				Action: "opened",
				PullRequest: &PullRequest{
					Title: "Add feature",
					State: "open",
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(events)
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	result, err := client.FetchUserEvents(context.Background(), "test-token")

	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Verify PushEvent parsing
	assert.Equal(t, "1", result[0].ID)
	assert.Equal(t, "PushEvent", result[0].Type)
	assert.Equal(t, "user/repo-a", result[0].Repo.Name)
	assert.Len(t, result[0].Payload.Commits, 1)
	assert.Equal(t, "abc123", result[0].Payload.Commits[0].SHA)
	assert.Equal(t, "initial commit", result[0].Payload.Commits[0].Message)
	assert.Equal(t, 1, result[0].Payload.Size)

	// Verify PullRequestEvent parsing
	assert.Equal(t, "2", result[1].ID)
	assert.Equal(t, "PullRequestEvent", result[1].Type)
	assert.Equal(t, "user/repo-b", result[1].Repo.Name)
	assert.Equal(t, "opened", result[1].Payload.Action)
	require.NotNil(t, result[1].Payload.PullRequest)
	assert.Equal(t, "Add feature", result[1].Payload.PullRequest.Title)
	assert.Equal(t, "open", result[1].Payload.PullRequest.State)
}

func TestFetchUserEvents_Pagination(t *testing.T) {
	// Build page 1: exactly 30 events (triggers fetching page 2)
	page1 := make([]Event, 30)
	for i := range page1 {
		page1[i] = Event{
			ID:   fmt.Sprintf("p1-%d", i),
			Type: "PushEvent",
			Repo: Repo{Name: "user/repo"},
		}
	}

	// Page 2: 5 events (less than 30, stops pagination)
	page2 := make([]Event, 5)
	for i := range page2 {
		page2[i] = Event{
			ID:   fmt.Sprintf("p2-%d", i),
			Type: "PushEvent",
			Repo: Repo{Name: "user/repo"},
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "1":
			json.NewEncoder(w).Encode(page1)
		case "2":
			json.NewEncoder(w).Encode(page2)
		default:
			json.NewEncoder(w).Encode([]Event{})
		}
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	result, err := client.FetchUserEvents(context.Background(), "test-token")

	require.NoError(t, err)
	assert.Len(t, result, 35) // 30 from page 1 + 5 from page 2

	// Verify first event is from page 1
	assert.Equal(t, "p1-0", result[0].ID)
	// Verify last event is from page 2
	assert.Equal(t, "p2-4", result[34].ID)
}

func TestFetchUserEvents_AuthHeader(t *testing.T) {
	var receivedAuth string
	var receivedAccept string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Event{})
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	_, err := client.FetchUserEvents(context.Background(), "ghp_my-secret-token")

	require.NoError(t, err)
	assert.Equal(t, "Bearer ghp_my-secret-token", receivedAuth)
	assert.Equal(t, "application/vnd.github+json", receivedAccept)
}

func TestFetchUserEvents_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Bad credentials"}`))
	}))
	defer srv.Close()

	client := newTestClient(srv.URL)
	result, err := client.FetchUserEvents(context.Background(), "bad-token")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "401")
}

func TestSupportedEventTypes(t *testing.T) {
	expected := []string{
		"PushEvent",
		"PullRequestEvent",
		"PullRequestReviewEvent",
		"CreateEvent",
	}

	assert.Len(t, SupportedEventTypes, len(expected))

	for _, eventType := range expected {
		assert.True(t, SupportedEventTypes[eventType], "expected %s to be supported", eventType)
	}

	// Verify unsupported types return false
	assert.False(t, SupportedEventTypes["WatchEvent"])
	assert.False(t, SupportedEventTypes["ForkEvent"])
}
