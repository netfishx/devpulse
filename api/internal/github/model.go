package github

import "time"

// Event represents a GitHub event from the Events API.
// https://docs.github.com/en/rest/activity/events
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Repo      Repo      `json:"repo"`
	CreatedAt time.Time `json:"created_at"`
	Payload   Payload   `json:"payload"`
}

// Repo identifies the repository associated with an event.
type Repo struct {
	Name string `json:"name"`
}

// Payload contains event-type-specific data.
type Payload struct {
	// PushEvent
	Commits []Commit `json:"commits,omitempty"`
	Size    int      `json:"size,omitempty"`
	// PullRequestEvent
	Action      string       `json:"action,omitempty"`
	PullRequest *PullRequest `json:"pull_request,omitempty"`
}

// Commit represents a commit within a PushEvent payload.
type Commit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
}

// PullRequest represents a pull request within a PullRequestEvent payload.
type PullRequest struct {
	Title string `json:"title"`
	State string `json:"state"`
}
