# DevPulse API

Go (Echo v5) 后端服务，为 Web/iOS/Android 三端提供 REST API。

## 技术栈

- **框架**: Echo v5
- **数据库**: PostgreSQL (Supabase) + pgxpool
- **查询**: sqlc (SQL → 类型安全 Go 代码)
- **迁移**: golang-migrate
- **认证**: JWT + bcrypt

## 项目结构

```
api/
├── cmd/api/main.go          ← 入口，手动依赖注入
├── internal/
│   ├── auth/                ← 认证领域 (handler / service / model)
│   ├── apperror/            ← 统一错误类型 + Echo 错误中间件
│   ├── validate/            ← 共享请求验证 (go-playground/validator)
│   ├── config/              ← 环境变量配置
│   └── middleware/          ← 通用中间件 (JWT 等)
├── db/
│   ├── migrations/          ← SQL 迁移文件 (golang-migrate)
│   ├── queries/             ← sqlc 查询文件 (.sql)
│   └── generated/           ← sqlc 生成的代码 (git tracked)
└── sqlc.yaml
```

### 分层约定

- **Handler** — HTTP 关注点：请求绑定、验证、调用 Service、返回 JSON
- **Service** — 业务逻辑：密码哈希、JWT、OAuth，不知道 HTTP 的存在
- **sqlc generated** — 数据访问：类型安全的 SQL 查询函数

错误流：Service 返回 `*apperror.AppError` → Handler 直接 `return err` → `apperror.ErrorHandler` 统一格式化 JSON 响应。

## 开发

### 前置条件

- Go 1.22+
- [sqlc](https://sqlc.dev) (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)
- PostgreSQL (本地或 Supabase)

### 启动

```bash
# 设置环境变量（或创建 .env）
export DATABASE_URL="postgres://..."
export JWT_SECRET="your-secret"

# 启动服务
go run ./cmd/api
# → http://localhost:8080/health
```

### 常用命令

```bash
go run ./cmd/api              # 启动开发服务器
go test ./...                 # 全量测试
go test ./internal/auth/ -v   # 单包测试
go build -o devpulse-api ./cmd/api  # 构建二进制

sqlc generate                 # 从 db/queries/*.sql 重新生成代码
sqlc vet                      # 检查 SQL 查询语法
```

### 数据库迁移

```bash
# 安装 migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 执行迁移
migrate -path db/migrations -database "$DATABASE_URL" up

# 回滚一步
migrate -path db/migrations -database "$DATABASE_URL" down 1

# 创建新迁移
migrate create -ext sql -dir db/migrations -seq <name>
```

### 添加新查询

1. 在 `db/queries/` 下写 SQL 文件
2. 运行 `sqlc generate`
3. 在 Service 层调用生成的函数

## API 端点

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | `/health` | 健康检查 | 无 |
| POST | `/api/register` | 用户注册 | 无 |
| POST | `/api/login` | 用户登录 | 无 |

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DATABASE_URL` | `postgres://localhost:5432/devpulse_dev?sslmode=disable` | PostgreSQL 连接字符串 |
| `JWT_SECRET` | `devpulse-dev-secret-change-me` | JWT 签名密钥 |
| `PORT` | `8080` | 服务端口 |
