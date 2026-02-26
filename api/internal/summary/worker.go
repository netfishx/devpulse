package summary

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	riverlib "github.com/riverqueue/river"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
)

// AggregateArgs are the arguments for the daily aggregation job.
type AggregateArgs struct{}

func (AggregateArgs) Kind() string { return "daily_aggregate" }

// AggregateWorker aggregates activities into daily summaries.
type AggregateWorker struct {
	riverlib.WorkerDefaults[AggregateArgs]
	q *dbgen.Queries
}

func NewAggregateWorker(q *dbgen.Queries) *AggregateWorker {
	return &AggregateWorker{q: q}
}

func (w *AggregateWorker) Work(ctx context.Context, job *riverlib.Job[AggregateArgs]) error {
	yesterday := time.Now().AddDate(0, 0, -1)
	startOfDay := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.AddDate(0, 0, 1)

	users, err := w.q.ListDistinctActivityUsers(ctx)
	if err != nil {
		return err
	}

	for _, userID := range users {
		if err := w.aggregateUser(ctx, userID, startOfDay, endOfDay); err != nil {
			slog.Error("aggregation failed for user", "user_id", userID, "error", err)
		}
	}
	return nil
}

func (w *AggregateWorker) aggregateUser(ctx context.Context, userID int64, start, end time.Time) error {
	row, err := w.q.AggregateDailySummary(ctx, dbgen.AggregateDailySummaryParams{
		UserID:  userID,
		Column2: pgtype.Timestamptz{Time: start, Valid: true},
		Column3: pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil {
		return err
	}

	err = w.q.UpsertDailySummary(ctx, dbgen.UpsertDailySummaryParams{
		UserID:        userID,
		Date:          pgtype.Date{Time: start, Valid: true},
		TotalCommits:  pgtype.Int4{Int32: row.TotalCommits, Valid: true},
		TotalPrs:      pgtype.Int4{Int32: row.TotalPrs, Valid: true},
		CodingMinutes: pgtype.Int4{Int32: 0, Valid: true},
		TopRepos:      json.RawMessage("[]"),
		TopLanguages:  json.RawMessage("[]"),
	})
	if err != nil {
		return err
	}

	slog.Info("daily summary aggregated", "user_id", userID, "date", start.Format("2006-01-02"))
	return nil
}
