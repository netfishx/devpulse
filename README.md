# DevPulse

Developer personal dashboard that aggregates coding activity from GitHub, Wakatime and other platforms into a unified view.

## Tech Stack

| Layer | Choice |
|-------|--------|
| Backend | Go + [Echo v5](https://echo.labstack.com/) |
| Database | PostgreSQL ([Supabase](https://supabase.com/)) + pgxpool |
| Query | [sqlc](https://sqlc.dev/) (SQL-first type-safe Go code generation) |
| Migration | [golang-migrate](https://github.com/golang-migrate/migrate) |
| API Contract | OpenAPI 3.1 |
| Web | Next.js 16 + React 19 + Tailwind v4 |

## Project Structure

```
devpulse/
├── api/              # Go REST API
│   ├── cmd/api/      # Entrypoint + dependency wiring
│   ├── internal/     # Domain-grouped business logic
│   │   ├── auth/     # Registration, login, JWT
│   │   └── oauth/    # GitHub OAuth flow
│   ├── db/           # Migrations, SQL queries, generated code
│   └── sqlc.yaml
├── web/              # Next.js frontend
│   └── src/
│       ├── app/      # App Router pages
│       └── lib/      # API client
├── docs/
│   └── openapi.yaml  # Shared API contract
├── Makefile
└── .env.example
```

## Getting Started

### Prerequisites

- Go 1.23+
- Bun
- PostgreSQL (or a [Supabase](https://supabase.com/) project)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI
- [sqlc](https://sqlc.dev/) CLI

### Setup

```bash
# 1. Clone
git clone https://github.com/netfishx/devpulse.git
cd devpulse

# 2. Environment variables
cp .env.example .env
# Edit .env with your Supabase credentials, JWT secret, and GitHub OAuth keys

# 3. Run database migrations
make db-migrate

# 4. Start the API server (port 8080)
make api-dev

# 5. Start the web frontend (port 3000)
cd web && bun install
make web-dev
```

### Common Commands

```bash
make api-dev        # Run Go API server
make api-test       # Run all Go tests
make api-lint       # Go vet
make web-dev        # Run Next.js dev server
make web-build      # Production build
make db-migrate     # Apply migrations
make db-sqlc        # Regenerate sqlc code
```

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | - | Health check |
| POST | `/api/register` | - | Create account |
| POST | `/api/login` | - | Get JWT token |
| GET | `/api/me` | Bearer | Current user profile |
| GET | `/api/github/redirect` | Bearer | GitHub OAuth URL |
| POST | `/api/github/callback` | Bearer | Exchange OAuth code |

Full spec: [`docs/openapi.yaml`](docs/openapi.yaml)

## License

MIT
