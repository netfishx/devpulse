-- name: ListTopRepos :many
SELECT payload->>'repo' AS name,
       COUNT(*)::int AS count,
       MAX(occurred_at) AS last_active
FROM activities
WHERE user_id = $1
  AND occurred_at >= CURRENT_DATE - $2::int
  AND payload->>'repo' IS NOT NULL
GROUP BY payload->>'repo'
ORDER BY count DESC
LIMIT 10;
