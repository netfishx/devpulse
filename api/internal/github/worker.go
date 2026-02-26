package github

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	riverlib "github.com/riverqueue/river"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
)

// SyncArgs are the arguments for the GitHub sync job.
type SyncArgs struct{}

func (SyncArgs) Kind() string { return "github_sync" }

// SyncWorker syncs GitHub events for all users with a GitHub data source.
type SyncWorker struct {
	riverlib.WorkerDefaults[SyncArgs]
	q      *dbgen.Queries
	client *Client
}

func NewSyncWorker(q *dbgen.Queries, client *Client) *SyncWorker {
	return &SyncWorker{q: q, client: client}
}

func (w *SyncWorker) Work(ctx context.Context, job *riverlib.Job[SyncArgs]) error {
	sources, err := w.q.ListDataSourcesByProvider(ctx, "github")
	if err != nil {
		return err
	}

	for _, src := range sources {
		if err := w.syncUser(ctx, src); err != nil {
			slog.Error("github sync failed for user", "user_id", src.UserID, "error", err)
			// Continue with other users, don't fail the whole job
		}
	}
	return nil
}

func (w *SyncWorker) syncUser(ctx context.Context, src dbgen.ListDataSourcesByProviderRow) error {
	ds, err := w.q.GetDataSourceByUserAndProvider(ctx, dbgen.GetDataSourceByUserAndProviderParams{
		UserID:   src.UserID,
		Provider: "github",
	})
	if err != nil {
		return err
	}

	events, err := w.client.FetchUserEvents(ctx, string(ds.AccessToken))
	if err != nil {
		return err
	}

	var inserted int
	for _, evt := range events {
		if !SupportedEventTypes[evt.Type] {
			continue
		}

		payload, _ := json.Marshal(map[string]any{
			"repo":    evt.Repo.Name,
			"payload": evt.Payload,
		})

		eventType := mapEventType(evt.Type)

		err := w.q.InsertActivity(ctx, dbgen.InsertActivityParams{
			UserID:     src.UserID,
			Source:     "github",
			Type:       eventType,
			Payload:    payload,
			OccurredAt: pgtype.Timestamptz{Time: evt.CreatedAt, Valid: true},
			ExternalID: pgtype.Text{String: evt.ID, Valid: true},
		})
		if err != nil {
			slog.Error("insert activity failed", "event_id", evt.ID, "error", err)
			continue
		}
		inserted++
	}

	slog.Info("github sync complete", "user_id", src.UserID, "events", len(events), "inserted", inserted)
	return nil
}

func mapEventType(ghType string) string {
	switch ghType {
	case "PushEvent":
		return "push"
	case "PullRequestEvent":
		return "pull_request"
	case "PullRequestReviewEvent":
		return "review"
	case "CreateEvent":
		return "create"
	default:
		return ghType
	}
}
