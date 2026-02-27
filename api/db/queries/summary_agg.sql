-- name: ListWeeklySummaries :many
SELECT DATE_TRUNC('week', date)::date AS period,
       COALESCE(SUM(total_commits), 0)::int AS total_commits,
       COALESCE(SUM(total_prs), 0)::int AS total_prs,
       COALESCE(SUM(coding_minutes), 0)::int AS coding_minutes
FROM daily_summaries
WHERE user_id = $1
  AND date >= CURRENT_DATE - ($2::int * 7)
GROUP BY DATE_TRUNC('week', date)
ORDER BY period;

-- name: ListMonthlySummaries :many
SELECT DATE_TRUNC('month', date)::date AS period,
       COALESCE(SUM(total_commits), 0)::int AS total_commits,
       COALESCE(SUM(total_prs), 0)::int AS total_prs,
       COALESCE(SUM(coding_minutes), 0)::int AS coding_minutes
FROM daily_summaries
WHERE user_id = $1
  AND date >= CURRENT_DATE - ($2::int * 30)
GROUP BY DATE_TRUNC('month', date)
ORDER BY period;

-- name: ListDailySummariesForHeatmap :many
SELECT date, COALESCE(total_commits, 0)::int AS total_commits
FROM daily_summaries
WHERE user_id = $1
  AND date >= CURRENT_DATE - $2::int
ORDER BY date;
