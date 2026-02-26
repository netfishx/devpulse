# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

DevPulse — 开发者个人数据仪表盘，汇聚 GitHub、Wakatime 等平台的开发活动数据，提供 Web/iOS/Android 三端原生体验。纯个人工具，不涉及团队协作功能。

## 架构

Monorepo，四个独立子项目通过 OpenAPI 契约对齐，无共享代码库：

```
Go API (Echo v5) ← 唯一后端，所有客户端通过 REST 通信
    ├── JWT 认证（HS256, 7天过期）
    ├── OAuth 代理（GitHub/Wakatime token 交换和刷新）
    ├── 定时拉取（cron 每小时同步外部数据）
    ├── Webhook 接收（GitHub 实时事件入库）
    └── 数据聚合（原始事件 → 日/周/月统计快照）

三端客户端各自独立实现网络层、UI、业务逻辑
    ├── Web (Next.js 16) — PC 优先，图表交互主力端
    ├── iOS (SwiftUI + Swift 6) — 碎片化浏览 + Widget + 推送（未开始）
    └── Android (Jetpack Compose + Kotlin) — 功能对齐 iOS（未开始）
```

## 技术栈

| 层 | 选型 |
|---|------|
| 后端框架 | Go + Echo v5 |
| 数据库 | PostgreSQL (Supabase) + pgxpool |
| 查询 | sqlc（SQL → 类型安全 Go 代码）|
| 迁移 | golang-migrate |
| API 契约 | OpenAPI 3.1 (`docs/openapi.yaml`) |
| Web | Next.js 16 (App Router) + React 19 + Tailwind v4 + ESLint |
| 部署 | Docker + fly.io |
| CI | GitHub Actions（按 paths 触发，各端独立流水线）|

## 仓库结构

```
devpulse/
├── api/                    ← Go 后端（详见 api/README.md）
│   ├── cmd/api/main.go     ← 入口，手动依赖注入
│   ├── internal/
│   │   ├── auth/           ← 认证领域（handler/service/model）
│   │   ├── oauth/          ← OAuth 领域（handler/service）
│   │   ├── apperror/       ← 统一错误类型 + Echo 错误中间件
│   │   ├── jwtutil/        ← JWT 生成/解析
│   │   ├── middleware/     ← JWT 认证中间件
│   │   ├── validate/       ← 共享请求验证
│   │   └── config/         ← 环境变量配置
│   ├── db/
│   │   ├── migrations/     ← SQL 迁移文件
│   │   ├── queries/        ← sqlc 查询文件
│   │   └── generated/      ← sqlc 生成代码（git tracked）
│   └── sqlc.yaml
├── web/                    ← Next.js 16
│   └── src/
│       ├── app/            ← App Router 页面
│       └── lib/api.ts      ← Go API 客户端
├── docs/
│   ├── openapi.yaml        ← 三端共享 API 契约
│   └── plans/              ← 设计文档和实施计划
├── Makefile                ← 开发快捷命令
└── .env.example            ← 环境变量模板
```

## 后端分层约定

```
请求 → Echo → Handler (绑定+验证) → Service (业务逻辑) → sqlc Queries (DB)
                                          ↓
错误流: Service 返回 *apperror.AppError → Handler return err → ErrorHandler 统一 JSON
```

- **Handler** — HTTP 关注点：请求绑定、验证、调用 Service、返回 JSON
- **Service** — 业务逻辑：密码哈希、JWT、OAuth，不知道 HTTP 的存在
- **sqlc generated** — 数据访问：类型安全 SQL 函数，由 `sqlc generate` 从 `db/queries/*.sql` 生成

## 开发命令

```bash
# Makefile 快捷方式（从根目录执行）
make api-dev          # 启动 Go API → :8080
make api-test         # Go 全量测试
make web-dev          # 启动 Next.js → :3000
make web-build        # Next.js 生产构建
make db-migrate       # 执行数据库迁移（需要 DATABASE_URL）
make db-sqlc          # 重新生成 sqlc 代码

# 或直接进入子目录
cd api && go run ./cmd/api
cd api && go test ./internal/auth/ -v    # 单包测试
cd web && bun run dev
```

## 核心数据模型

四张核心表，`bigint GENERATED ALWAYS AS IDENTITY` 主键，所有 FK 列有索引：

- **users** — 用户账户（email 唯一）
- **data_sources** — 外部平台绑定（`user_id + provider` 唯一，token 加密存储）
- **activities** — 原始活动事件（JSONB payload + GIN 索引，按 `user_id + occurred_at DESC` 索引）
- **daily_summaries** — 每日聚合快照（`user_id + date` 唯一约束）

## 关键设计决策

- **代码共享策略**：三端只共享 OpenAPI schema，各端独立实现
- **数据库选型**：sqlc + pgx/v5，SQL-first 编译时类型安全，零运行时开销
- **依赖注入**：手动构造函数注入（cmd/api/main.go 组装 repo → service → handler）
- **错误处理**：`apperror.AppError` + 统一 ErrorHandler 中间件，Service 层映射错误，Handler 只 `return err`
- **数据同步**：Webhook 为主（实时），定时拉取为辅（兜底）
- **V1 范围**：GitHub 活动流、Wakatime 编码时间、每日摘要、趋势图表、多数据源管理。不做团队功能、AI 分析、CI/CD 数据源

## 参考文档

- 设计文档：`docs/plans/2026-02-26-devpulse-design.md`
- 实施计划：`docs/plans/2026-02-26-devpulse-plan.md`
- API 详情：`api/README.md`
