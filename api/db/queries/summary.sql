-- name: UpsertDailySummary :exec
INSERT INTO daily_summaries (user_id, date, total_commits, total_prs, coding_minutes, top_repos, top_languages)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id, date)
DO UPDATE SET
    total_commits = EXCLUDED.total_commits,
    total_prs = EXCLUDED.total_prs,
    coding_minutes = EXCLUDED.coding_minutes,
    top_repos = EXCLUDED.top_repos,
    top_languages = EXCLUDED.top_languages;

-- name: ListSummariesByUser :many
SELECT id, user_id, date, total_commits, total_prs, coding_minutes, top_repos, top_languages
FROM daily_summaries
WHERE user_id = $1
  AND date >= CURRENT_DATE - $2::int
ORDER BY date DESC;

-- name: AggregateDailySummary :one
SELECT
    count(*) FILTER (WHERE type = 'push')::int AS total_commits,
    count(*) FILTER (WHERE type = 'pull_request')::int AS total_prs
FROM activities
WHERE user_id = $1
  AND occurred_at >= $2::timestamptz
  AND occurred_at < $3::timestamptz;
