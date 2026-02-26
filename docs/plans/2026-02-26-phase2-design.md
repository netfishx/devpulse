# DevPulse Phase 2 Design: First Data Source

> Goal: GitHub data flows in via scheduled sync, user sees commit timeline and daily stats on the web dashboard.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Data sync trigger | Cron only (no webhook) | GitHub Events API is near-realtime; hourly cron is sufficient for personal dashboard; no public URL needed |
| Task scheduler | River (PostgreSQL-native job queue) | Already using Supabase PostgreSQL; zero extra infra; transactional safety; built-in retry/dedup |
| Chart library | shadcn/ui Charts (Base UI version) | Recharts under the hood with shadcn design system; consistent styling with Tailwind |
| Dashboard scope | Summary cards + 30-day bar chart + activity timeline | High-impact views first; top repos/languages deferred to Phase 3 |

---

## 1. Backend Data Flow

```
GitHub Events API
       ↓ (hourly River periodic job)
  GitHubSyncWorker
       ↓ parse events → INSERT activities (ON CONFLICT DO NOTHING)
       ↓ (daily River periodic job)
  AggregateWorker
       ↓ SELECT + GROUP BY → UPSERT daily_summaries
       ↓
  REST API endpoints
    GET /api/activities   → paginated activity events
    GET /api/summaries    → last N days of daily stats
       ↓
  Web Dashboard
```

### GitHub Sync Logic

- Call `GET https://api.github.com/users/{username}/events` (max 300 events, 10 pages)
- Event types to process: `PushEvent`, `PullRequestEvent`, `PullRequestReviewEvent`, `CreateEvent`
- Deduplication: `UNIQUE (user_id, source, external_id)` constraint on activities table; `external_id` stores GitHub event ID; INSERT with `ON CONFLICT DO NOTHING`
- Token read from `data_sources` table; authenticated user rate limit is 5000 req/hour

### Daily Aggregation Logic

- SQL aggregation: `SELECT date(occurred_at), count(*) FILTER (WHERE type='push'), ...` grouped by day
- UPSERT into `daily_summaries` via `ON CONFLICT (user_id, date) DO UPDATE`
- Default: aggregate yesterday's data (runs at midnight); supports manual backfill for date range

### New API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/activities` | Bearer | Paginated activity list. Query: `?page=1&per_page=20&source=github` |
| GET | `/api/summaries` | Bearer | Daily stats. Query: `?days=30` |

---

## 2. Backend Project Structure

### New Files

```
api/
├── internal/
│   ├── github/              # GitHub data sync domain
│   │   ├── client.go        # GitHub API client (Events API)
│   │   ├── worker.go        # River GitHubSyncWorker
│   │   └── model.go         # GitHub event type mappings
│   ├── activity/            # Activity query domain
│   │   ├── handler.go       # GET /api/activities
│   │   └── service.go       # Paginated query logic
│   ├── summary/             # Aggregation domain
│   │   ├── handler.go       # GET /api/summaries
│   │   ├── service.go       # Aggregation logic
│   │   └── worker.go        # River AggregateWorker
│   └── river/               # River client init + worker registration
│       └── setup.go
├── db/
│   ├── migrations/
│   │   └── 002_activities_external_id.up.sql
│   └── queries/
│       ├── activity.sql      # Activity CRUD + paginated queries
│       └── summary.sql       # Summary UPSERT + range queries
└── cmd/api/main.go           # Add River client start/stop
```

### River Integration

River client initialized in `cmd/api/main.go` alongside Echo server:

- Two periodic jobs: hourly `GitHubSyncWorker`, daily `AggregateWorker`
- Max 2 workers (personal tool, minimal concurrency)
- Graceful shutdown: `riverClient.Stop(ctx)` in defer chain before pool close
- River uses its own migration tables in the same Supabase database

### Schema Change

```sql
-- 002_activities_external_id.up.sql
ALTER TABLE activities ADD COLUMN external_id text;
CREATE UNIQUE INDEX idx_activities_dedup ON activities (user_id, source, external_id);
```

`external_id` is nullable — non-GitHub sources may not have one. PostgreSQL UNIQUE indexes treat NULLs as distinct, so this doesn't block other sources.

---

## 3. Web Frontend Dashboard

### Tech

- shadcn/ui (Base UI version) — card, chart, button, badge components
- Dashboard route: `/dashboard` (auth required; login redirects here)

### Page Layout

```
┌─────────────────────────────────────┐
│  Summary Cards (3 cols)             │
│  ┌─────────┬──────────┬──────────┐  │
│  │ Commits │   PRs    │  Repos   │  │
│  └─────────┴──────────┴──────────┘  │
├─────────────────────────────────────┤
│  30-Day Bar Chart                   │
│  X: date, Y: commit count          │
├─────────────────────────────────────┤
│  Activity Timeline                  │
│  icon + description + repo + time   │
│  [ Load More ]                      │
└─────────────────────────────────────┘
```

### Data Fetching

- Summary cards + chart: Server Component SSR from `/api/summaries?days=30`; extract "today" from response
- Activity timeline: Client-side pagination via `/api/activities?page=N&per_page=20`; "Load More" button

### shadcn/ui Components (install only what's needed)

- `card` — summary cards
- `chart` — bar chart (Recharts wrapper)
- `button` — load more
- `badge` — event type labels

### API Client Extension

```typescript
// web/src/lib/api.ts additions
api.activities(params: { page?: number; perPage?: number })
api.summaries(params: { days?: number })
```

---

## Deferred to Phase 3

- GitHub Webhook receiver (real-time push events)
- Wakatime integration
- Top repos / top languages charts
- Push notifications
