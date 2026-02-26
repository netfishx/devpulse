package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapEventType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"PushEvent", "push"},
		{"PullRequestEvent", "pull_request"},
		{"PullRequestReviewEvent", "review"},
		{"CreateEvent", "create"},
		{"UnknownEvent", "UnknownEvent"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapEventType(tt.input))
		})
	}
}

func TestSyncArgsKind(t *testing.T) {
	args := SyncArgs{}
	assert.Equal(t, "github_sync", args.Kind())
}
