# DevPulse — 设计文档

## 一句话定义

开发者的个人数据仪表盘：把散落在 GitHub、Wakatime、CI 等平台的开发活动数据汇聚到一处，三端原生体验。

## 架构总览

```
┌──────────┐  ┌──────────┐  ┌──────────┐
│ Next.js  │  │ SwiftUI  │  │ Compose  │
│  Web     │  │  iOS     │  │ Android  │
└────┬─────┘  └────┬─────┘  └────┬─────┘
     │             │              │
     └──────┬──────┴──────────────┘
            │ REST (OpenAPI)
     ┌──────▼──────┐
     │   Go API    │  ← Echo + 定时任务
     │   Server    │  ← OAuth 代理 + Webhook 接收
     └──────┬──────┘
            │
     ┌──────▼──────┐
     │ PostgreSQL  │  ← 活动数据 + 聚合快照
     │ + Redis     │  ← 缓存 + 任务队列
     └─────────────┘
            │
     ┌──────▼──────┐
     │  外部 API   │  ← GitHub, Wakatime, ...
     └─────────────┘
```

## 核心功能（V1 范围）

| 功能 | 说明 |
|------|------|
| GitHub 活动流 | commits、PRs、reviews 的时间线 + 统计图表 |
| 编码时间 | Wakatime 数据：按项目/语言/日期的编码时长 |
| 每日摘要 | 自动生成"今日开发小结"，移动端推送 |
| 趋势图表 | 周/月维度的活动趋势、对比 |
| 多数据源管理 | OAuth 授权 + token 管理界面 |

**明确砍掉（YAGNI）：**

- 团队/协作功能 — 纯个人工具
- AI 分析/建议 — V1 不做
- CI/CD 数据源 — V2 再考虑

## 各端职责

### Go 后端（Echo）

- 用户认证（JWT）
- OAuth 代理（GitHub、Wakatime 的 token 交换和刷新）
- 定时拉取任务（cron：每小时同步一次外部数据）
- Webhook 接收（GitHub push/PR events 实时入库）
- 数据聚合（原始事件 → 日/周/月统计快照）
- OpenAPI schema 作为三端契约

### Web（Next.js）

- 主力数据浏览端：丰富的图表交互
- 数据源 OAuth 授权流程（redirect callback）
- 响应式但 PC 优先

### iOS（SwiftUI）

- 碎片化浏览：通勤时看每日摘要
- 推送通知（每日摘要、异常告警）
- Widget（桌面小组件显示今日编码时长）

### Android（Jetpack Compose）

- 功能对齐 iOS
- Widget（主屏小组件）
- 推送通知

## 数据模型（核心表）

```sql
-- 用户
users (
  id            BIGSERIAL PRIMARY KEY,
  email         TEXT UNIQUE NOT NULL,
  name          TEXT NOT NULL,
  avatar_url    TEXT,
  created_at    TIMESTAMPTZ DEFAULT now(),
  updated_at    TIMESTAMPTZ DEFAULT now()
);

-- 用户绑定的外部平台
data_sources (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT REFERENCES users(id),
  provider      TEXT NOT NULL,  -- github / wakatime / ...
  access_token  BYTEA NOT NULL, -- AES 加密
  refresh_token BYTEA,
  expires_at    TIMESTAMPTZ,
  created_at    TIMESTAMPTZ DEFAULT now()
);

-- 原始活动事件
activities (
  id            BIGSERIAL PRIMARY KEY,
  user_id       BIGINT REFERENCES users(id),
  source        TEXT NOT NULL,  -- github / wakatime
  type          TEXT NOT NULL,  -- commit / pr / review / coding
  payload       JSONB,
  occurred_at   TIMESTAMPTZ NOT NULL,
  created_at    TIMESTAMPTZ DEFAULT now()
);

-- 每日聚合快照
daily_summaries (
  id              BIGSERIAL PRIMARY KEY,
  user_id         BIGINT REFERENCES users(id),
  date            DATE NOT NULL,
  total_commits   INT DEFAULT 0,
  total_prs       INT DEFAULT 0,
  coding_minutes  INT DEFAULT 0,
  top_repos       JSONB,
  top_languages   JSONB,
  UNIQUE (user_id, date)
);
```

## 技术选型

| 层 | 选型 | 理由 |
|---|------|------|
| 后端框架 | Echo | 中间件链直观，集中式错误处理 |
| 数据库 | PostgreSQL | jsonb 存异构 payload，聚合查询强 |
| 缓存 | Redis | API 限流缓存 + 定时任务锁 |
| API 契约 | OpenAPI 3.1 | 三端各自手写客户端，schema 做对齐参考 |
| 认证 | JWT + refresh token | 无状态，移动端友好 |
| Web | Next.js 16 + Recharts | 已熟悉，图表库轻量 |
| iOS | SwiftUI + Swift 6 | 原生体验，有 orbit-ios 经验 |
| Android | Jetpack Compose + Kotlin | 原生体验，有 orbit-android 经验 |
| 部署 | Docker + fly.io | Go 单二进制部署，fly.io 免费额度够用 |
| CI | GitHub Actions | 按路径触发，各端独立流水线 |

## 仓库结构（Monorepo）

```
devpulse/
├── api/              ← Go (Echo) 后端
├── web/              ← Next.js
├── ios/              ← SwiftUI (Xcode project)
├── android/          ← Compose (Gradle project)
├── docs/
│   └── openapi.yaml  ← API 契约
├── .github/
│   └── workflows/    ← CI（按 paths 触发）
└── README.md
```

## 代码共享策略

三端只共享 API 契约（OpenAPI schema），各端独立实现网络层、UI、业务逻辑。

## 起步计划（三端同时）

### Phase 1（2 周）— 地基

- Go: 项目脚手架 + DB migration + JWT 认证 + GitHub OAuth
- Web: 项目脚手架 + 登录页 + OAuth callback
- iOS: 项目脚手架 + 登录流程
- Android: 项目脚手架 + 登录流程

### Phase 2（2 周）— 第一个数据源

- Go: GitHub 数据拉取 + webhook + 日聚合
- Web: GitHub 活动时间线 + 基础图表
- iOS: 活动列表 + 每日摘要卡片
- Android: 对齐 iOS

### Phase 3（2 周）— 体验完善

- Go: Wakatime 数据源 + 推送服务
- Web: 多数据源切换 + 趋势对比图
- iOS: Widget + 推送通知
- Android: Widget + 推送通知

## 风险与应对

| 风险 | 应对 |
|------|------|
| GitHub API 限流（5000 req/h） | Webhook 为主 + 定时拉取为辅 + Redis 缓存 |
| Go 不熟导致后端进度慢 | Phase 1 只做认证，复杂度渐进 |
| 三端同时开发精力分散 | 每个 phase 后端先行 1-2 天，前端跟进 |
| OAuth token 安全存储 | Go 端 AES 加密存储，移动端用 Keychain/KeyStore |
