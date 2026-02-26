# Phase 3 Web 体验完善 — 设计文档

## 目标

升级 Dashboard 体验：趋势对比图（日/周/月三档 + 环比叠加）、Top Repos 排行、编码热力图、数据源筛选器、Settings 数据源管理页。

## 方案

后端聚合 + 前端展示分离（方案 A）。所有聚合在 SQL/Go 层完成，前端只做展示。三端共享 API。

## 后端新增端点

全部在 protected group (`/api`) 下，需 Bearer token。

| 端点 | 说明 |
|------|------|
| `GET /api/summaries/weekly?weeks=12&source=` | 最近 N 周聚合（DATE_TRUNC week） |
| `GET /api/summaries/monthly?months=12&source=` | 最近 N 月聚合（DATE_TRUNC month） |
| `GET /api/activities/top-repos?days=30&source=` | 按 commit 数排名的仓库 TOP 10 |
| `GET /api/summaries/heatmap?days=365&source=` | 每日活动强度 level 0-4 |
| `GET /api/data-sources` | 当前用户已绑定的数据源列表 |

### source 参数过滤

所有 summaries/activities 端点（含现有端点）加可选 `?source=github` 参数。sqlc 查询加 `WHERE ($source::text IS NULL OR source = $source)`。

### 周/月聚合 SQL

```sql
-- weekly
SELECT DATE_TRUNC('week', date) AS period,
       SUM(total_commits), SUM(total_prs), SUM(coding_minutes)
FROM daily_summaries
WHERE user_id = $1 AND date >= NOW() - ($2 || ' weeks')::interval
GROUP BY period ORDER BY period;

-- monthly
SELECT DATE_TRUNC('month', date) AS period,
       SUM(total_commits), SUM(total_prs), SUM(coding_minutes)
FROM daily_summaries
WHERE user_id = $1 AND date >= NOW() - ($2 || ' months')::interval
GROUP BY period ORDER BY period;
```

### Top Repos SQL

```sql
SELECT payload->>'repo' AS name, COUNT(*) AS count,
       MAX(occurred_at) AS last_active
FROM activities
WHERE user_id = $1 AND occurred_at >= NOW() - ($2 || ' days')::interval
GROUP BY name ORDER BY count DESC LIMIT 10;
```

### 热力图

复用 daily_summaries，返回 `{date, level, count}`。level 在 Go Service 层按 total_commits 分桶：0=0, 1=1-3, 2=4-9, 3=10-19, 4=20+。

### 响应结构

```json
// weekly/monthly
{ "summaries": [{ "period": "2026-W08", "totalCommits": 45, "totalPrs": 8, "codingMinutes": 0 }] }

// top-repos
{ "repos": [{ "name": "user/repo", "count": 45, "lastActive": "2026-02-26" }] }

// heatmap
{ "days": [{ "date": "2026-02-26", "level": 3, "count": 15 }] }

// data-sources
{ "sources": [{ "id": 1, "provider": "github", "connected": true, "connectedAt": "2026-02-20T..." }] }
```

## 前端页面结构

### Dashboard (`/`) 改造

```
Header: DevPulse    [Source Filter ▾] [User / Logout]
─────────────────────────────────────────────
Summary Cards (3 张) + 底部环比 ↑12% / ↓5%
─────────────────────────────────────────────
趋势图 [日 | 周 | 月] Tab
  BarChart 当前周期 + 半透明 Line 上一周期
─────────────────────────────────────────────
Top Repos (左)  |  编码热力图 (右)
─────────────────────────────────────────────
Recent Activity Timeline
```

**数据源筛选器**：Select 组件，选项 All / GitHub / Wakatime。选中后所有 API 请求附带 `?source=xxx`。

**环比指标**：日=前30天vs再前30天，周=本周vs上周，月=本月vs上月。绿 ↑ 红 ↓。

**趋势对比图**：ComposedChart = BarChart（当前周期）+ Line（上一周期，半透明虚线）。

**热力图**：纯 div + Tailwind，52x7 格子矩阵，4 级绿色深浅。无第三方库。

### Settings (`/settings`) 新增

数据源管理页，列出已绑定数据源的状态。Connect 按钮暂时 disabled（OAuth 未实现）。

## 新增 shadcn/ui 组件

- Tabs（日/周/月切换）
- Select（数据源筛选）
- Separator（Settings 分隔）

## 新建文件

| 文件 | 用途 |
|------|------|
| `web/src/app/settings/page.tsx` | 数据源管理页 |
| `web/src/components/trend-chart.tsx` | 趋势对比图 |
| `web/src/components/heatmap.tsx` | 编码热力图 |
| `web/src/components/top-repos.tsx` | Top Repos 排行 |
| `api/internal/summary/heatmap.go` | 热力图 handler + service |
| `api/internal/activity/top_repos.go` | Top Repos handler + service |
| `api/internal/datasource/handler.go` | 数据源列表 handler |
| `api/internal/datasource/service.go` | 数据源列表 service |
| `api/db/queries/summary_agg.sql` | 周/月聚合 + 热力图查询 |
| `api/db/queries/activity_agg.sql` | Top Repos 查询 |

## 环比对比策略

不需要新端点。前端请求两次同一聚合端点：

- 日视图：`?days=30` + `?days=60`（取后 30 天作为对比）
- 周视图：`?weeks=12` 返回 12 周数据，前端拆分当前 vs 上一周期
- 月视图：`?months=12` 同理

Summary Cards 环比同理，基于当前选中的时间粒度计算。
