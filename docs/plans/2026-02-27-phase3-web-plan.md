# Phase 3 Web 体验完善 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Upgrade DevPulse Dashboard with trend comparison charts (daily/weekly/monthly), Top Repos ranking, contribution heatmap, data source filtering, and a Settings page for data source management.

**Architecture:** Backend-first approach — add 5 new API endpoints (weekly/monthly summaries, heatmap, top repos, data sources list) with sqlc-generated queries, then build frontend components using shadcn/ui Base UI (tabs, select, separator) + recharts ComposedChart. Source filtering via optional `source` query parameter on activities endpoints.

**Tech Stack:** Go + Echo v5 + sqlc + pgx/v5 (backend), Next.js 16 + React 19 + Tailwind v4 + shadcn/ui Base UI + recharts 2.15.4 (frontend)

---

## Context for implementer

**Key patterns in this codebase:**

- **Handler pattern** (`api/internal/*/handler.go`): Parse request params → call Service → return JSON. Handler never does business logic. See `api/internal/activity/handler.go` for reference.
- **Service pattern** (`api/internal/*/service.go`): Business logic + DB queries via `*dbgen.Queries`. Returns response structs. See `api/internal/activity/service.go` for reference.
- **Route registration**: Each handler has `RegisterRoutes(g *echo.Group)` method. Wired in `api/cmd/api/main.go` on the `protected` group.
- **Auth**: `mw.GetUserID(c)` extracts authenticated user ID from Echo context. All protected endpoints use this.
- **Error handling**: Services return `*apperror.AppError`. Handlers just `return err`. ErrorHandler middleware formats JSON response.
- **sqlc**: SQL queries in `api/db/queries/*.sql`, generated Go code in `api/db/generated/`. Regenerate with `cd api && sqlc generate`.
- **Frontend**: shadcn/ui Base UI (`"style": "base-vega"` in `web/components.json`). Components import from `@base-ui/react/*`. API client in `web/src/lib/api.ts`.
- **CSS rule**: Zero margin. Use `gap`, `padding`, flex/grid for spacing. NEVER use `m-*`, `mx-*`, `my-*`, `mt-*`, `space-x-*`, `space-y-*`.
- **Echo v5 handler signature**: `func(c *echo.Context) error` — note `*echo.Context` is already a pointer.
- **pgtype**: sqlc with pgx/v5 uses `pgtype.Date`, `pgtype.Int4`, `pgtype.Timestamptz`, `pgtype.Text` for nullable columns. Access values via `.Time`, `.Int32`, `.Valid`, etc.

---

### Task 1: New sqlc queries

**Files:**
- Create: `api/db/queries/summary_agg.sql`
- Create: `api/db/queries/activity_agg.sql`
- Modify: `api/db/queries/activity.sql`

**Step 1: Create weekly/monthly aggregation + heatmap queries**

Create `api/db/queries/summary_agg.sql`:

```sql
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
```

**Step 2: Create top repos query**

Create `api/db/queries/activity_agg.sql`:

```sql
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
```

**Step 3: Add source filter to existing activity queries**

Replace the contents of `api/db/queries/activity.sql` with:

```sql
-- name: InsertActivity :exec
INSERT INTO activities (user_id, source, type, payload, occurred_at, external_id)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, source, external_id) DO NOTHING;

-- name: ListActivitiesByUser :many
SELECT id, user_id, source, type, payload, occurred_at, external_id, created_at
FROM activities
WHERE user_id = $1
  AND ($4::text = '' OR source = $4)
ORDER BY occurred_at DESC
LIMIT $2 OFFSET $3;

-- name: CountActivitiesByUser :one
SELECT count(*) FROM activities
WHERE user_id = $1
  AND ($2::text = '' OR source = $2);

-- name: ListDistinctActivityUsers :many
SELECT DISTINCT user_id FROM activities;
```

Note: `ListActivitiesByUser` now has 4 params (userID, limit, offset, source). `CountActivitiesByUser` now has 2 params (userID, source). Pass `""` (empty string) for no source filter.

**Step 4: Regenerate sqlc**

Run: `cd api && sqlc generate`
Expected: Success, regenerated files in `api/db/generated/`

**Step 5: Verify generated code compiles**

Run: `cd api && go build ./...`
Expected: Build fails because existing callers of `ListActivitiesByUser` and `CountActivitiesByUser` need updating (param count changed). This is expected — Task 2 will fix callers.

**Step 6: Commit**

```bash
git add api/db/queries/summary_agg.sql api/db/queries/activity_agg.sql api/db/queries/activity.sql api/db/generated/
git commit -m "feat(api): add aggregation sqlc queries and source filter"
```

---

### Task 2: Weekly/Monthly summary + Heatmap endpoints

**Files:**
- Modify: `api/internal/summary/handler.go`
- Modify: `api/internal/summary/service.go`
- Modify: `api/internal/summary/handler_test.go`

**Step 1: Add response types and service methods**

Add to `api/internal/summary/service.go`:

```go
type PeriodSummary struct {
	Period        string `json:"period"`
	TotalCommits  int32  `json:"totalCommits"`
	TotalPrs      int32  `json:"totalPrs"`
	CodingMinutes int32  `json:"codingMinutes"`
}

type PeriodSummariesResponse struct {
	Summaries []PeriodSummary `json:"summaries"`
}

type HeatmapDay struct {
	Date  string `json:"date"`
	Level int    `json:"level"`
	Count int    `json:"count"`
}

type HeatmapResponse struct {
	Days []HeatmapDay `json:"days"`
}

func (s *Service) ListWeekly(ctx context.Context, userID int64, weeks int) (*PeriodSummariesResponse, error) {
	if weeks < 1 || weeks > 52 {
		weeks = 12
	}
	rows, err := s.q.ListWeeklySummaries(ctx, dbgen.ListWeeklySummariesParams{
		UserID:  userID,
		Column2: int32(weeks),
	})
	if err != nil {
		return nil, apperror.Internalf("list weekly summaries: %w", err)
	}
	summaries := make([]PeriodSummary, 0, len(rows))
	for _, r := range rows {
		// Format as ISO week: "2026-W08"
		year, week := r.Period.Time.ISOWeek()
		summaries = append(summaries, PeriodSummary{
			Period:        fmt.Sprintf("%d-W%02d", year, week),
			TotalCommits:  r.TotalCommits,
			TotalPrs:      r.TotalPrs,
			CodingMinutes: r.CodingMinutes,
		})
	}
	return &PeriodSummariesResponse{Summaries: summaries}, nil
}

func (s *Service) ListMonthly(ctx context.Context, userID int64, months int) (*PeriodSummariesResponse, error) {
	if months < 1 || months > 24 {
		months = 12
	}
	rows, err := s.q.ListMonthlySummaries(ctx, dbgen.ListMonthlySummariesParams{
		UserID:  userID,
		Column2: int32(months),
	})
	if err != nil {
		return nil, apperror.Internalf("list monthly summaries: %w", err)
	}
	summaries := make([]PeriodSummary, 0, len(rows))
	for _, r := range rows {
		summaries = append(summaries, PeriodSummary{
			Period:        r.Period.Time.Format("2006-01"),
			TotalCommits:  r.TotalCommits,
			TotalPrs:      r.TotalPrs,
			CodingMinutes: r.CodingMinutes,
		})
	}
	return &PeriodSummariesResponse{Summaries: summaries}, nil
}

func (s *Service) Heatmap(ctx context.Context, userID int64, days int) (*HeatmapResponse, error) {
	if days < 1 || days > 365 {
		days = 365
	}
	rows, err := s.q.ListDailySummariesForHeatmap(ctx, dbgen.ListDailySummariesForHeatmapParams{
		UserID:  userID,
		Column2: int32(days),
	})
	if err != nil {
		return nil, apperror.Internalf("list heatmap: %w", err)
	}
	result := make([]HeatmapDay, 0, len(rows))
	for _, r := range rows {
		count := int(r.TotalCommits)
		result = append(result, HeatmapDay{
			Date:  r.Date.Time.Format(time.DateOnly),
			Level: commitCountToLevel(count),
			Count: count,
		})
	}
	return &HeatmapResponse{Days: result}, nil
}

func commitCountToLevel(count int) int {
	switch {
	case count == 0:
		return 0
	case count <= 3:
		return 1
	case count <= 9:
		return 2
	case count <= 19:
		return 3
	default:
		return 4
	}
}
```

Note: The generated sqlc param struct field names (e.g., `Column2`) depend on how sqlc interprets the SQL. The implementer MUST read the generated code in `api/db/generated/summary_agg.sql.go` and use the actual field names. The names above (`Column2`) are best guesses.

**Step 2: Add handler methods**

Add to `api/internal/summary/handler.go`:

```go
func (h *Handler) ListWeekly(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}
	weeks, _ := strconv.Atoi(c.QueryParam("weeks"))
	resp, err := h.svc.ListWeekly(c.Request().Context(), userID, weeks)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListMonthly(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}
	months, _ := strconv.Atoi(c.QueryParam("months"))
	resp, err := h.svc.ListMonthly(c.Request().Context(), userID, months)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) Heatmap(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}
	days, _ := strconv.Atoi(c.QueryParam("days"))
	resp, err := h.svc.Heatmap(c.Request().Context(), userID, days)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}
```

**Step 3: Register routes**

Update `RegisterRoutes` in `api/internal/summary/handler.go`:

```go
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/summaries", h.List)
	g.GET("/summaries/weekly", h.ListWeekly)
	g.GET("/summaries/monthly", h.ListMonthly)
	g.GET("/summaries/heatmap", h.Heatmap)
}
```

**Step 4: Add tests**

Add to `api/internal/summary/handler_test.go`:

```go
func TestListWeekly_MissingAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/summaries/weekly", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := NewHandler(nil)
	err := h.ListWeekly(c)
	assert.Error(t, err)
}

func TestListMonthly_MissingAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/summaries/monthly", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := NewHandler(nil)
	err := h.ListMonthly(c)
	assert.Error(t, err)
}

func TestHeatmap_MissingAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/summaries/heatmap", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := NewHandler(nil)
	err := h.Heatmap(c)
	assert.Error(t, err)
}

func TestCommitCountToLevel(t *testing.T) {
	tests := []struct{ count, want int }{
		{0, 0}, {1, 1}, {3, 1}, {4, 2}, {9, 2}, {10, 3}, {19, 3}, {20, 4}, {100, 4},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, commitCountToLevel(tt.count), "count=%d", tt.count)
	}
}
```

**Step 5: Run tests**

Run: `cd api && go test ./internal/summary/ -v`
Expected: All pass

**Step 6: Commit**

```bash
git add api/internal/summary/
git commit -m "feat(api): add weekly/monthly summary and heatmap endpoints"
```

---

### Task 3: Top Repos endpoint

**Files:**
- Modify: `api/internal/activity/handler.go`
- Modify: `api/internal/activity/service.go`
- Modify: `api/internal/activity/handler_test.go`

**Step 1: Add service method**

Add to `api/internal/activity/service.go`:

```go
type RepoStats struct {
	Name       string `json:"name"`
	Count      int    `json:"count"`
	LastActive string `json:"lastActive"`
}

type TopReposResponse struct {
	Repos []RepoStats `json:"repos"`
}

func (s *Service) TopRepos(ctx context.Context, userID int64, days int, source string) (*TopReposResponse, error) {
	if days < 1 || days > 365 {
		days = 30
	}
	rows, err := s.q.ListTopRepos(ctx, dbgen.ListTopReposParams{
		UserID:  userID,
		Column2: int32(days),
		Column3: source, // "" = all sources
	})
	if err != nil {
		return nil, apperror.Internalf("list top repos: %w", err)
	}
	repos := make([]RepoStats, 0, len(rows))
	for _, r := range rows {
		name := ""
		if r.Name != nil {
			name = *r.Name
		}
		repos = append(repos, RepoStats{
			Name:       name,
			Count:      int(r.Count),
			LastActive: r.LastActive.Time.Format(time.DateOnly),
		})
	}
	return &TopReposResponse{Repos: repos}, nil
}
```

Note: `ListTopRepos` generated param names (`Column2`, `Column3`) and return field types (`r.Name` as `*string`) MUST be verified against the actual generated code. Read `api/db/generated/activity_agg.sql.go` first.

**Step 2: Add handler method**

Add to `api/internal/activity/handler.go`:

```go
func (h *Handler) TopRepos(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}
	days, _ := strconv.Atoi(c.QueryParam("days"))
	source := c.QueryParam("source")
	resp, err := h.svc.TopRepos(c.Request().Context(), userID, days, source)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}
```

**Step 3: Register route**

Update `RegisterRoutes` in `api/internal/activity/handler.go`:

```go
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/activities", h.List)
	g.GET("/activities/top-repos", h.TopRepos)
}
```

**Step 4: Add test**

Add to `api/internal/activity/handler_test.go`:

```go
func TestTopRepos_MissingAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/activities/top-repos", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := NewHandler(nil)
	err := h.TopRepos(c)
	assert.Error(t, err)
}
```

**Step 5: Run tests**

Run: `cd api && go test ./internal/activity/ -v`
Expected: All pass

**Step 6: Commit**

```bash
git add api/internal/activity/
git commit -m "feat(api): add top repos endpoint"
```

---

### Task 4: Data Sources list endpoint

**Files:**
- Create: `api/internal/datasource/handler.go`
- Create: `api/internal/datasource/service.go`
- Create: `api/internal/datasource/handler_test.go`

**Step 1: Create service**

Create `api/internal/datasource/service.go`:

```go
package datasource

import (
	"context"

	dbgen "github.com/ethanwang/devpulse/api/db/generated"
	"github.com/ethanwang/devpulse/api/internal/apperror"
)

type SourceInfo struct {
	ID          int64  `json:"id"`
	Provider    string `json:"provider"`
	Connected   bool   `json:"connected"`
	ConnectedAt string `json:"connectedAt"`
}

type ListResponse struct {
	Sources []SourceInfo `json:"sources"`
}

type Service struct {
	q *dbgen.Queries
}

func NewService(q *dbgen.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) List(ctx context.Context, userID int64) (*ListResponse, error) {
	rows, err := s.q.ListDataSourcesByUser(ctx, userID)
	if err != nil {
		return nil, apperror.Internalf("list data sources: %w", err)
	}
	sources := make([]SourceInfo, 0, len(rows))
	for _, r := range rows {
		connectedAt := ""
		if r.CreatedAt.Valid {
			connectedAt = r.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
		}
		sources = append(sources, SourceInfo{
			ID:          r.ID,
			Provider:    r.Provider,
			Connected:   true,
			ConnectedAt: connectedAt,
		})
	}
	return &ListResponse{Sources: sources}, nil
}
```

Note: `ListDataSourcesByUser` already exists in `api/db/queries/data_source.sql`. Check the generated return type fields in `api/db/generated/data_source.sql.go`.

**Step 2: Create handler**

Create `api/internal/datasource/handler.go`:

```go
package datasource

import (
	"net/http"

	"github.com/labstack/echo/v5"

	mw "github.com/ethanwang/devpulse/api/internal/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/data-sources", h.List)
}

func (h *Handler) List(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.List(c.Request().Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}
```

**Step 3: Create test**

Create `api/internal/datasource/handler_test.go`:

```go
package datasource

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestList_MissingAuth(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/data-sources", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := NewHandler(nil)
	err := h.List(c)
	assert.Error(t, err)
}
```

**Step 4: Run test**

Run: `cd api && go test ./internal/datasource/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add api/internal/datasource/
git commit -m "feat(api): add data sources list endpoint"
```

---

### Task 5: Source filter on activities + wire all in main.go

**Files:**
- Modify: `api/internal/activity/service.go` (update List to accept source param)
- Modify: `api/internal/activity/handler.go` (pass source param)
- Modify: `api/cmd/api/main.go` (wire datasource handler)

**Step 1: Update activity service to pass source filter**

In `api/internal/activity/service.go`, update the `List` method to accept a `source` parameter and pass it to the sqlc queries (which now require it after Task 1 changes):

```go
func (s *Service) List(ctx context.Context, userID int64, page, perPage int, source string) (*ListResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	rows, err := s.q.ListActivitiesByUser(ctx, dbgen.ListActivitiesByUserParams{
		UserID:  userID,
		Limit:   int32(perPage),
		Offset:  int32(offset),
		Column4: source, // "" = no filter
	})
	// ... rest same

	total, err := s.q.CountActivitiesByUser(ctx, dbgen.CountActivitiesByUserParams{
		UserID:  userID,
		Column2: source, // "" = no filter
	})
	// ... rest same
```

Note: sqlc-generated param names (`Column4`, `Column2`) MUST be verified against actual generated code.

**Step 2: Update activity handler to read source param**

In `api/internal/activity/handler.go`, update `List`:

```go
func (h *Handler) List(c *echo.Context) error {
	userID, err := mw.GetUserID(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	source := c.QueryParam("source")

	resp, err := h.svc.List(c.Request().Context(), userID, page, perPage, source)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}
```

**Step 3: Wire datasource handler in main.go**

Add import and wiring in `api/cmd/api/main.go`:

```go
import (
	// ... existing imports
	"github.com/ethanwang/devpulse/api/internal/datasource"
)

// After summaryHandler.RegisterRoutes(protected), add:
dsSvc := datasource.NewService(queries)
dsHandler := datasource.NewHandler(dsSvc)
dsHandler.RegisterRoutes(protected)
```

**Step 4: Verify build and tests**

Run: `cd api && go build ./... && go test ./... -v`
Expected: Build succeeds, all tests pass

**Step 5: Commit**

```bash
git add api/internal/activity/ api/cmd/api/main.go
git commit -m "feat(api): add source filter to activities and wire datasource endpoint"
```

---

### Task 6: Install shadcn/ui components + extend API client

**Files:**
- Modify: `web/src/lib/api.ts`
- New components installed by shadcn CLI

**Step 1: Install shadcn/ui components**

Run from `web/` directory:

```bash
cd web && bunx shadcn@latest add tabs select separator -y
```

Expected: 3 components added to `web/src/components/ui/`

**Step 2: Verify components use Base UI**

Check that the installed components import from `@base-ui/react/*` (not Radix). If they import from Radix, something is wrong with the config — verify `web/components.json` has `"style": "base-vega"`.

**Step 3: Extend API client**

Add new types and methods to `web/src/lib/api.ts`:

```typescript
export interface PeriodSummary {
  period: string;
  totalCommits: number;
  totalPrs: number;
  codingMinutes: number;
}

export interface PeriodSummariesResponse {
  summaries: PeriodSummary[];
}

export interface HeatmapDay {
  date: string;
  level: number;
  count: number;
}

export interface HeatmapResponse {
  days: HeatmapDay[];
}

export interface RepoStats {
  name: string;
  count: number;
  lastActive: string;
}

export interface TopReposResponse {
  repos: RepoStats[];
}

export interface DataSourceInfo {
  id: number;
  provider: string;
  connected: boolean;
  connectedAt: string;
}

export interface DataSourcesResponse {
  sources: DataSourceInfo[];
}
```

Add to the `api` object:

```typescript
weeklySummaries: (weeks = 12) =>
  request<PeriodSummariesResponse>(`/api/summaries/weekly?weeks=${weeks}`),

monthlySummaries: (months = 12) =>
  request<PeriodSummariesResponse>(`/api/summaries/monthly?months=${months}`),

heatmap: (days = 365) =>
  request<HeatmapResponse>(`/api/summaries/heatmap?days=${days}`),

topRepos: (days = 30, source = "") =>
  request<TopReposResponse>(
    `/api/activities/top-repos?days=${days}${source ? `&source=${source}` : ""}`
  ),

dataSources: () =>
  request<DataSourcesResponse>("/api/data-sources"),
```

Also update the existing `activities` method to accept source:

```typescript
activities: (page = 1, perPage = 20, source = "") =>
  request<ActivityListResponse>(
    `/api/activities?page=${page}&per_page=${perPage}${source ? `&source=${source}` : ""}`
  ),
```

**Step 4: Verify build**

Run: `cd web && bun run build`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add web/src/lib/api.ts web/src/components/ui/ web/package.json web/bun.lock
git commit -m "feat(web): install tabs/select/separator and extend API client"
```

---

### Task 7: Trend chart component

**Files:**
- Create: `web/src/components/trend-chart.tsx`

**Step 1: Create the trend chart component**

Create `web/src/components/trend-chart.tsx` with:

- Props: `dailySummaries: DailySummary[]`, `weeklySummaries: PeriodSummary[]`, `monthlySummaries: PeriodSummary[]`
- Tabs component with 3 tabs: Day / Week / Month
- Each tab renders a recharts `ComposedChart` (from recharts) inside `ChartContainer`:
  - `Bar` for current period commits + PRs (stacked, using `--color-chart-1` and `--color-chart-2`)
  - `Line` for previous period commits (dashed, `--color-chart-3`, `opacity: 0.5`) as comparison overlay
- For day tab: split 60-day data into current 30 days + previous 30 days
- For week tab: split data in half (current weeks vs previous weeks)
- For month tab: same split strategy
- `ChartTooltip` with `ChartTooltipContent`
- X-axis formatted: MM/DD (day), Wxx (week), MMM (month)
- Use `ChartConfig` to define color mappings

Import `ComposedChart, Bar, Line, XAxis, YAxis` from `recharts`.
Import `Tabs, TabsList, TabsTrigger, TabsContent` from shadcn/ui (verify actual export names from installed component).

Key detail: the "previous period" comparison line. For daily view with 60 days of data: index 0-29 = previous, index 30-59 = current. Align them by position (day-of-period index) so they overlay correctly. Transform data into `{ date, currentCommits, prevCommits, currentPrs, prevPrs }[]` shape before passing to chart.

**Step 2: Verify build**

Run: `cd web && bun run build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add web/src/components/trend-chart.tsx
git commit -m "feat(web): add trend chart component with day/week/month tabs"
```

---

### Task 8: Heatmap + Top Repos components

**Files:**
- Create: `web/src/components/heatmap.tsx`
- Create: `web/src/components/top-repos.tsx`

**Step 1: Create heatmap component**

Create `web/src/components/heatmap.tsx`:

- Props: `days: HeatmapDay[]`
- Pure div-based grid (no third-party lib), GitHub contribution graph style
- 52 columns x 7 rows (weeks x days), read right to left (most recent = right)
- Each cell is a small square div with Tailwind background colors based on `level`:
  - Level 0: `bg-muted`
  - Level 1: `bg-emerald-200 dark:bg-emerald-900`
  - Level 2: `bg-emerald-400 dark:bg-emerald-700`
  - Level 3: `bg-emerald-600 dark:bg-emerald-500`
  - Level 4: `bg-emerald-800 dark:bg-emerald-300`
- Grid layout: `grid grid-flow-col grid-rows-7 gap-1`
- Wrap in a horizontally scrollable container if needed
- Bottom row: month labels (Jan, Feb, ...) positioned at week boundaries
- Tooltip on hover showing date + count (use native `title` attribute for simplicity)

**Step 2: Create top repos component**

Create `web/src/components/top-repos.tsx`:

- Props: `repos: RepoStats[]`
- Numbered list inside a Card
- Each row: rank number, repo name (truncated), commit count badge
- Use `Badge` component for count
- Layout: `flex flex-col gap-2`, each row `flex items-center gap-3`
- If empty, show "No repository data yet."

**Step 3: Verify build**

Run: `cd web && bun run build`
Expected: Build succeeds

**Step 4: Commit**

```bash
git add web/src/components/heatmap.tsx web/src/components/top-repos.tsx
git commit -m "feat(web): add heatmap and top repos components"
```

---

### Task 9: Dashboard page upgrade

**Files:**
- Modify: `web/src/app/page.tsx`

This is the biggest frontend task. Replace the existing dashboard with the upgraded version.

**Step 1: Rewrite the dashboard page**

Key changes to `web/src/app/page.tsx`:

1. **Imports**: Add new api methods, new components (TrendChart, Heatmap, TopRepos), Tabs, Select from shadcn/ui

2. **State**: Add new state variables:
   ```typescript
   const [weeklySummaries, setWeeklySummaries] = useState<PeriodSummary[]>([]);
   const [monthlySummaries, setMonthlySummaries] = useState<PeriodSummary[]>([]);
   const [heatmapDays, setHeatmapDays] = useState<HeatmapDay[]>([]);
   const [topRepos, setTopRepos] = useState<RepoStats[]>([]);
   const [source, setSource] = useState("");
   ```

3. **Data fetching**: Expand `Promise.all` in useEffect:
   ```typescript
   const [userData, summaryData, activityData, weeklyData, monthlyData, heatmapData, reposData] =
     await Promise.all([
       api.me(),
       api.summaries(60), // 60 days for comparison (current 30 + previous 30)
       api.activities(1, 20, source),
       api.weeklySummaries(24), // 24 weeks for comparison
       api.monthlySummaries(24), // 24 months for comparison
       api.heatmap(365),
       api.topRepos(30, source),
     ]);
   ```

4. **Re-fetch on source change**: Add source to useEffect deps, or use a separate effect for source-dependent calls.

5. **Summary cards with comparison**: Calculate period-over-period change:
   ```typescript
   // For daily view (default): current 30 days vs previous 30 days
   const current30 = summaries.slice(0, 30);
   const prev30 = summaries.slice(30, 60);
   const currentCommits = current30.reduce((s, d) => s + d.totalCommits, 0);
   const prevCommits = prev30.reduce((s, d) => s + d.totalCommits, 0);
   const commitsDelta = prevCommits > 0
     ? Math.round(((currentCommits - prevCommits) / prevCommits) * 100)
     : 0;
   ```
   Show delta below each card number: green `↑12%` or red `↓5%`

6. **Header**: Add Select component for source filtering:
   ```tsx
   <Select value={source} onValueChange={setSource}>
     <SelectTrigger className="w-32">
       <SelectValue placeholder="All Sources" />
     </SelectTrigger>
     <SelectContent>
       <SelectItem value="">All</SelectItem>
       <SelectItem value="github">GitHub</SelectItem>
       <SelectItem value="wakatime" disabled>Wakatime</SelectItem>
     </SelectContent>
   </Select>
   ```
   Note: Verify the actual shadcn/ui Base UI Select component API. It may differ from Radix-based Select. Read `web/src/components/ui/select.tsx` first.

7. **Layout**: Replace the existing chart section with TrendChart, add TopRepos + Heatmap in a 2-column grid:
   ```tsx
   {/* Trend Chart (replaces old bar chart) */}
   <TrendChart
     dailySummaries={summaries}
     weeklySummaries={weeklySummaries}
     monthlySummaries={monthlySummaries}
   />

   {/* Top Repos + Heatmap side by side */}
   <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
     <Card>
       <CardHeader><CardTitle>Top Repos</CardTitle></CardHeader>
       <CardContent><TopRepos repos={topRepos} /></CardContent>
     </Card>
     <Card>
       <CardHeader><CardTitle>Contributions</CardTitle></CardHeader>
       <CardContent><Heatmap days={heatmapDays} /></CardContent>
     </Card>
   </div>
   ```

**Step 2: Verify build**

Run: `cd web && bun run build`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add web/src/app/page.tsx
git commit -m "feat(web): upgrade dashboard with trends, heatmap, top repos, source filter"
```

---

### Task 10: Settings page

**Files:**
- Create: `web/src/app/settings/page.tsx`

**Step 1: Create settings page**

Create `web/src/app/settings/page.tsx`:

```tsx
"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { ArrowLeft, Check, Link as LinkIcon } from "lucide-react";

import { api, type DataSourceInfo } from "@/lib/api";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";

const PROVIDERS = [
  { id: "github", name: "GitHub" },
  { id: "wakatime", name: "Wakatime" },
];

export default function SettingsPage() {
  const router = useRouter();
  const [sources, setSources] = useState<DataSourceInfo[]>([]);
  const [loading, setLoading] = useState(true);

  const redirectToLogin = useCallback(() => {
    localStorage.removeItem("token");
    router.replace("/login");
  }, [router]);

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) { router.replace("/login"); return; }
    api.dataSources()
      .then((data) => setSources(data.sources))
      .catch(() => redirectToLogin())
      .finally(() => setLoading(false));
  }, [router, redirectToLogin]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
      </div>
    );
  }

  const connectedProviders = new Set(sources.map((s) => s.provider));

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="flex items-center gap-4 border-b px-6 py-4">
        <Button variant="ghost" size="icon-sm" onClick={() => router.push("/")}>
          <ArrowLeft />
        </Button>
        <h1 className="text-xl font-bold tracking-tight">Settings</h1>
      </header>

      <main className="flex flex-1 flex-col gap-6 p-6">
        <Card>
          <CardHeader>
            <CardTitle>Data Sources</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-4">
              {PROVIDERS.map((provider, idx) => {
                const connected = connectedProviders.has(provider.id);
                const source = sources.find((s) => s.provider === provider.id);
                return (
                  <div key={provider.id}>
                    {idx > 0 && <Separator className="[margin-bottom:1rem]" />}
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <span className="font-medium">{provider.name}</span>
                        {connected ? (
                          <Badge variant="default">
                            <Check data-icon="inline-start" className="size-3" />
                            Connected
                          </Badge>
                        ) : (
                          <Badge variant="outline">Not Connected</Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        {connected && source && (
                          <span className="text-xs text-muted-foreground">
                            since {new Date(source.connectedAt).toLocaleDateString()}
                          </span>
                        )}
                        <Button
                          variant={connected ? "destructive" : "default"}
                          size="sm"
                          disabled={!connected}
                        >
                          {connected ? "Disconnect" : "Connect"}
                        </Button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      </main>
    </div>
  );
}
```

Note: The Separator inside the Settings page uses a `[margin-bottom:1rem]` Tailwind arbitrary value. This is an exception to the zero-margin rule since Separator is a visual element that needs spacing from the item below it. Alternatively, restructure to avoid it by using gap on parent — implementer should prefer the gap approach if possible.

**Step 2: Verify build**

Run: `cd web && bun run build`
Expected: Build succeeds, route `/settings` appears in output

**Step 3: Commit**

```bash
git add web/src/app/settings/
git commit -m "feat(web): add settings page with data source management"
```

---

## Summary

| Task | Description | Backend | Frontend |
|------|-------------|---------|----------|
| 1 | sqlc queries (aggregation + source filter) | ✓ | |
| 2 | Weekly/Monthly + Heatmap endpoints | ✓ | |
| 3 | Top Repos endpoint | ✓ | |
| 4 | Data Sources list endpoint | ✓ | |
| 5 | Source filter + wire main.go | ✓ | |
| 6 | Install shadcn/ui + extend API client | | ✓ |
| 7 | Trend chart component | | ✓ |
| 8 | Heatmap + Top Repos components | | ✓ |
| 9 | Dashboard page upgrade | | ✓ |
| 10 | Settings page | | ✓ |

**Dependencies:** Tasks 2-5 depend on Task 1. Tasks 7-10 depend on Task 6. Task 9 depends on Tasks 7+8.

**Backend tasks (1-5) and frontend foundation (6) can run early. Frontend components (7-8) can run in parallel. Task 9 integrates everything. Task 10 is independent of 7-9.**
