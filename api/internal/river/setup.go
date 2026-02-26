package river

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	riverlib "github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// NoOpArgs is a placeholder job type used to satisfy River's requirement
// of having at least one registered worker. It will be removed once real
// workers (e.g. GitHub sync, daily aggregation) are added.
type NoOpArgs struct{}

func (NoOpArgs) Kind() string { return "no_op" }

// NoOpWorker does nothing. It exists only so the River client can start.
type NoOpWorker struct {
	riverlib.WorkerDefaults[NoOpArgs]
}

func (w *NoOpWorker) Work(_ context.Context, _ *riverlib.Job[NoOpArgs]) error {
	return nil
}

// NewClient creates a River client with the given pool, workers, and periodic jobs.
// If workers is nil, a default Workers bundle with a no-op placeholder is used.
func NewClient(pool *pgxpool.Pool, workers *riverlib.Workers, periodicJobs []*riverlib.PeriodicJob) (*riverlib.Client[pgx.Tx], error) {
	if workers == nil {
		workers = riverlib.NewWorkers()
		riverlib.AddWorker(workers, &NoOpWorker{})
	}

	return riverlib.NewClient(riverpgxv5.New(pool), &riverlib.Config{
		Queues: map[string]riverlib.QueueConfig{
			riverlib.QueueDefault: {MaxWorkers: 2},
		},
		Workers:      workers,
		PeriodicJobs: periodicJobs,
	})
}
