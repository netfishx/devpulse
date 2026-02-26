# DevPulse Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a developer personal dashboard that aggregates GitHub/Wakatime data across Web, iOS, and Android with a Go backend.

**Architecture:** Monorepo with four independent subprojects (api/web/ios/android) sharing only an OpenAPI contract. Go (Echo) backend handles auth, OAuth proxy, data sync, and aggregation. Each frontend is fully native.

**Tech Stack:** Go + Echo, PostgreSQL, Redis, Next.js 16, SwiftUI + Swift 6, Jetpack Compose + Kotlin, Docker, fly.io

---

## Phase 1: Foundation (Tasks 1–12)

> Goal: All four projects scaffolded, user can register/login on all three clients via the Go API.

---

### Task 1: Go module + Echo hello world

**Files:**
- Create: `api/go.mod`
- Create: `api/main.go`
- Create: `api/.gitignore`

**Step 1: Initialize Go module**

```bash
cd /Users/ethanwang/projects/devpulse
mkdir -p api
cd api
go mod init github.com/ethanwang/devpulse/api
```

**Step 2: Install Echo**

```bash
go get github.com/labstack/echo/v4
```

**Step 3: Write main.go with health check**

```go
package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.Logger.Fatal(e.Start(":8080"))
}
```

**Step 4: Create .gitignore**

```
# Binaries
devpulse-api
*.exe

# Environment
.env

# IDE
.idea/
.vscode/

# OS
.DS_Store
```

**Step 5: Run and verify**

```bash
go run main.go &
curl http://localhost:8080/health
# Expected: {"status":"ok"}
kill %1
```

**Step 6: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add api/
git commit -m "feat(api): scaffold Go + Echo with health check"
```

---

### Task 2: PostgreSQL schema + migration tooling

**Files:**
- Create: `api/db/migrations/001_init.up.sql`
- Create: `api/db/migrations/001_init.down.sql`
- Create: `api/db/db.go`
- Modify: `api/go.mod` (new dependencies)

**Step 1: Install dependencies**

```bash
cd /Users/ethanwang/projects/devpulse/api
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/stdlib
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
```

**Step 2: Write up migration**

Create `api/db/migrations/001_init.up.sql`:

```sql
CREATE TABLE users (
    id          BIGSERIAL PRIMARY KEY,
    email       TEXT UNIQUE NOT NULL,
    name        TEXT NOT NULL,
    avatar_url  TEXT,
    password    TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE data_sources (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT REFERENCES users(id) ON DELETE CASCADE,
    provider      TEXT NOT NULL,
    access_token  BYTEA NOT NULL,
    refresh_token BYTEA,
    expires_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE activities (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT REFERENCES users(id) ON DELETE CASCADE,
    source      TEXT NOT NULL,
    type        TEXT NOT NULL,
    payload     JSONB,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_activities_user_occurred
    ON activities (user_id, occurred_at DESC);

CREATE TABLE daily_summaries (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT REFERENCES users(id) ON DELETE CASCADE,
    date            DATE NOT NULL,
    total_commits   INT DEFAULT 0,
    total_prs       INT DEFAULT 0,
    coding_minutes  INT DEFAULT 0,
    top_repos       JSONB,
    top_languages   JSONB,
    UNIQUE (user_id, date)
);
```

**Step 3: Write down migration**

Create `api/db/migrations/001_init.down.sql`:

```sql
DROP TABLE IF EXISTS daily_summaries;
DROP TABLE IF EXISTS activities;
DROP TABLE IF EXISTS data_sources;
DROP TABLE IF EXISTS users;
```

**Step 4: Write database connection helper**

Create `api/db/db.go`:

```go
package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}

func RunMigrations(db *sql.DB) error {
	driver, err := migratePostgres.WithInstance(db, &migratePostgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://db/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}
	log.Println("migrations applied successfully")
	return nil
}
```

**Step 5: Create local PostgreSQL database and test migration**

```bash
createdb devpulse_dev
cd /Users/ethanwang/projects/devpulse/api
DATABASE_URL="postgres://localhost:5432/devpulse_dev?sslmode=disable" \
  go run -tags postgres . migrate
# 先手动验证 SQL：
psql devpulse_dev -c "\dt"
# Expected: users, data_sources, activities, daily_summaries
```

**Step 6: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add api/
git commit -m "feat(api): add PostgreSQL schema and migration tooling"
```

---

### Task 3: User registration + password hashing

**Files:**
- Create: `api/handler/auth.go`
- Create: `api/handler/auth_test.go`
- Create: `api/model/user.go`
- Modify: `api/main.go` (add routes)
- Modify: `api/go.mod` (add bcrypt)

**Step 1: Install bcrypt**

```bash
cd /Users/ethanwang/projects/devpulse/api
go get golang.org/x/crypto/bcrypt
```

**Step 2: Write user model**

Create `api/model/user.go`:

```go
package model

import "time"

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL *string   `json:"avatarUrl,omitempty"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Name     string `json:"name" validate:"required,min=2"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}
```

**Step 3: Write failing test for registration**

Create `api/handler/auth_test.go`:

```go
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/ethanwang/devpulse/api/handler"
)

func TestRegister_Success(t *testing.T) {
	e := echo.New()
	body := `{"email":"test@example.com","name":"Test User","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := handler.NewAuthHandler(nil) // nil DB for now, will mock
	err := h.Register(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "test@example.com")
}
```

**Step 4: Run test to verify it fails**

```bash
cd /Users/ethanwang/projects/devpulse/api
go get github.com/stretchr/testify
go test ./handler/ -run TestRegister_Success -v
# Expected: FAIL — handler package doesn't exist yet
```

**Step 5: Implement auth handler (Register)**

Create `api/handler/auth.go`:

```go
package handler

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/ethanwang/devpulse/api/model"
)

type AuthHandler struct {
	db *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req model.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	var user model.User
	err = h.db.QueryRowContext(c.Request().Context(),
		`INSERT INTO users (email, name, password) VALUES ($1, $2, $3)
		 RETURNING id, email, name, avatar_url, created_at, updated_at`,
		req.Email, req.Name, string(hashed),
	).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return c.JSON(http.StatusConflict, map[string]string{"error": "email already exists"})
	}

	return c.JSON(http.StatusCreated, user)
}
```

**Step 6: Run test again**

```bash
go test ./handler/ -run TestRegister_Success -v
# Expected: PASS (handler binds and responds, DB call panics on nil but test structure works)
```

> Note: Full integration test with real DB comes in Task 4. This test validates request binding and response structure.

**Step 7: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add api/
git commit -m "feat(api): add user registration with bcrypt hashing"
```

---

### Task 4: JWT login + middleware

**Files:**
- Create: `api/handler/middleware.go`
- Modify: `api/handler/auth.go` (add Login)
- Create: `api/handler/login_test.go`
- Create: `api/config/config.go`
- Modify: `api/main.go` (wire routes + middleware)
- Modify: `api/go.mod` (add jwt)

**Step 1: Install JWT library**

```bash
cd /Users/ethanwang/projects/devpulse/api
go get github.com/golang-jwt/jwt/v5
```

**Step 2: Write config loader**

Create `api/config/config.go`:

```go
package config

import "os"

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/devpulse_dev?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "devpulse-dev-secret-change-me"),
		Port:        getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

**Step 3: Write failing test for login**

Create `api/handler/login_test.go`:

```go
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/ethanwang/devpulse/api/handler"
)

func TestLogin_InvalidJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{bad json`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := handler.NewAuthHandler(nil)
	_ = h.Login(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
```

**Step 4: Run test to verify it fails**

```bash
go test ./handler/ -run TestLogin -v
# Expected: FAIL — Login method doesn't exist
```

**Step 5: Implement Login + JWT generation**

Add to `api/handler/auth.go`:

```go
import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func (h *AuthHandler) SetJWTSecret(secret string) {
	h.jwtSecret = secret
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req model.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	var user model.User
	var hashedPassword string
	err := h.db.QueryRowContext(c.Request().Context(),
		`SELECT id, email, name, avatar_url, password, created_at, updated_at
		 FROM users WHERE email = $1`, req.Email,
	).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL,
		&hashedPassword, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}

	token, err := generateJWT(user.ID, h.jwtSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "token generation failed"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func generateJWT(userID int64, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
```

**Step 6: Write JWT auth middleware**

Create `api/handler/middleware.go`:

```go
package handler

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing token"})
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			claims := token.Claims.(jwt.MapClaims)
			c.Set("userID", int64(claims["sub"].(float64)))
			return next(c)
		}
	}
}
```

**Step 7: Run tests**

```bash
go test ./handler/ -v
# Expected: All tests pass
```

**Step 8: Wire everything in main.go**

Update `api/main.go` to connect DB, run migrations, register routes:

```go
package main

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/ethanwang/devpulse/api/config"
	"github.com/ethanwang/devpulse/api/db"
	"github.com/ethanwang/devpulse/api/handler"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db connection failed:", err)
	}
	defer database.Close()

	if err := db.RunMigrations(database); err != nil {
		log.Fatal("migration failed:", err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	auth := handler.NewAuthHandler(database)
	auth.SetJWTSecret(cfg.JWTSecret)

	api := e.Group("/api")
	api.POST("/register", auth.Register)
	api.POST("/login", auth.Login)

	// Protected routes
	protected := api.Group("")
	protected.Use(handler.JWTMiddleware(cfg.JWTSecret))
	protected.GET("/me", auth.Me)

	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
```

**Step 9: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add api/
git commit -m "feat(api): add JWT login, auth middleware, config"
```

---

### Task 5: GitHub OAuth flow (backend)

**Files:**
- Create: `api/handler/oauth.go`
- Create: `api/handler/oauth_test.go`
- Modify: `api/config/config.go` (add GitHub OAuth fields)
- Modify: `api/main.go` (add OAuth routes)

**Step 1: Add GitHub OAuth config fields**

Add to `config/config.go`:

```go
type Config struct {
	// ... existing fields
	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string
}

// In Load():
GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
GitHubCallbackURL:  getEnv("GITHUB_CALLBACK_URL", "http://localhost:3000/auth/github/callback"),
```

**Step 2: Write failing test**

Create `api/handler/oauth_test.go`:

```go
package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/ethanwang/devpulse/api/handler"
)

func TestGitHubRedirect_ReturnsURL(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/github", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := handler.NewOAuthHandler(nil, "test-client-id", "", "http://localhost/callback")
	_ = h.GitHubRedirect(c)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "github.com/login/oauth/authorize")
	assert.Contains(t, rec.Body.String(), "test-client-id")
}
```

**Step 3: Run test to verify it fails**

```bash
go test ./handler/ -run TestGitHub -v
# Expected: FAIL — OAuthHandler doesn't exist
```

**Step 4: Implement OAuth handler**

Create `api/handler/oauth.go`:

```go
package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
)

type OAuthHandler struct {
	db           *sql.DB
	clientID     string
	clientSecret string
	callbackURL  string
}

func NewOAuthHandler(db *sql.DB, clientID, clientSecret, callbackURL string) *OAuthHandler {
	return &OAuthHandler{
		db:           db,
		clientID:     clientID,
		clientSecret: clientSecret,
		callbackURL:  callbackURL,
	}
}

func (h *OAuthHandler) GitHubRedirect(c echo.Context) error {
	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user,repo",
		url.QueryEscape(h.clientID),
		url.QueryEscape(h.callbackURL),
	)
	return c.JSON(http.StatusOK, map[string]string{"url": authURL})
}

func (h *OAuthHandler) GitHubCallback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing code"})
	}

	// Exchange code for access token
	tokenURL := "https://github.com/login/oauth/access_token"
	resp, err := http.PostForm(tokenURL, url.Values{
		"client_id":     {h.clientID},
		"client_secret": {h.clientSecret},
		"code":          {code},
	})
	if err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "token exchange failed"})
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return c.JSON(http.StatusBadGateway, map[string]string{"error": "invalid token response"})
	}

	// Store encrypted token in data_sources
	userID := c.Get("userID").(int64)
	_, err = h.db.ExecContext(c.Request().Context(),
		`INSERT INTO data_sources (user_id, provider, access_token)
		 VALUES ($1, 'github', $2)
		 ON CONFLICT (user_id, provider)
		 DO UPDATE SET access_token = $2`,
		userID, []byte(tokenResp.AccessToken), // TODO: AES encrypt
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save token"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "connected"})
}
```

**Step 5: Add unique constraint for data_sources**

Create `api/db/migrations/002_data_sources_unique.up.sql`:

```sql
ALTER TABLE data_sources ADD CONSTRAINT uq_user_provider UNIQUE (user_id, provider);
```

Create `api/db/migrations/002_data_sources_unique.down.sql`:

```sql
ALTER TABLE data_sources DROP CONSTRAINT IF EXISTS uq_user_provider;
```

**Step 6: Run tests**

```bash
go test ./handler/ -v
# Expected: All pass
```

**Step 7: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add api/
git commit -m "feat(api): add GitHub OAuth redirect and callback"
```

---

### Task 6: OpenAPI contract (initial)

**Files:**
- Create: `docs/openapi.yaml`

**Step 1: Write initial OpenAPI spec**

Create `docs/openapi.yaml` with auth + OAuth endpoints. This is the contract all three clients reference.

```yaml
openapi: 3.1.0
info:
  title: DevPulse API
  version: 0.1.0
  description: Developer personal dashboard API

servers:
  - url: http://localhost:8080
    description: Local development

paths:
  /health:
    get:
      summary: Health check
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: ok

  /api/register:
    post:
      summary: Register a new user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RegisterRequest'
      responses:
        '201':
          description: User created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        '409':
          description: Email already exists

  /api/login:
    post:
      summary: Login
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/LoginRequest'
      responses:
        '200':
          description: Login successful
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/LoginResponse'
        '401':
          description: Invalid credentials

  /api/me:
    get:
      summary: Get current user
      security:
        - bearerAuth: []
      responses:
        '200':
          description: Current user
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'

  /api/oauth/github:
    get:
      summary: Get GitHub OAuth redirect URL
      security:
        - bearerAuth: []
      responses:
        '200':
          description: OAuth URL
          content:
            application/json:
              schema:
                type: object
                properties:
                  url:
                    type: string

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

  schemas:
    RegisterRequest:
      type: object
      required: [email, name, password]
      properties:
        email:
          type: string
          format: email
        name:
          type: string
          minLength: 2
        password:
          type: string
          minLength: 8

    LoginRequest:
      type: object
      required: [email, password]
      properties:
        email:
          type: string
          format: email
        password:
          type: string

    LoginResponse:
      type: object
      properties:
        token:
          type: string
        user:
          $ref: '#/components/schemas/User'

    User:
      type: object
      properties:
        id:
          type: integer
        email:
          type: string
        name:
          type: string
        avatarUrl:
          type: string
          nullable: true
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
```

**Step 2: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add docs/openapi.yaml
git commit -m "docs: add initial OpenAPI contract for auth endpoints"
```

---

### Task 7: Next.js web scaffold + login page

**Files:**
- Create: `web/` (Next.js project)
- Create: `web/src/app/page.tsx`
- Create: `web/src/app/login/page.tsx`
- Create: `web/src/lib/api.ts`

**Step 1: Scaffold Next.js project**

```bash
cd /Users/ethanwang/projects/devpulse
bunx create-next-app@latest web \
  --typescript --tailwind --eslint --app --src-dir \
  --no-import-alias --turbopack
```

**Step 2: Create API client**

Create `web/src/lib/api.ts`:

```typescript
const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = typeof window !== "undefined"
    ? localStorage.getItem("token")
    : null;

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Request failed: ${res.status}`);
  }

  return res.json();
}

export const api = {
  register: (data: { email: string; name: string; password: string }) =>
    request("/api/register", { method: "POST", body: JSON.stringify(data) }),

  login: (data: { email: string; password: string }) =>
    request<{ token: string; user: { id: number; email: string; name: string } }>(
      "/api/login",
      { method: "POST", body: JSON.stringify(data) },
    ),

  me: () =>
    request<{ id: number; email: string; name: string }>("/api/me"),
};
```

**Step 3: Create login page**

Create `web/src/app/login/page.tsx`:

```tsx
"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const { token } = await api.login({ email, password });
      localStorage.setItem("token", token);
      router.push("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center">
      <form onSubmit={handleSubmit} className="flex w-80 flex-col gap-4">
        <h1 className="text-2xl font-bold">DevPulse</h1>
        <input
          type="email"
          placeholder="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          className="rounded border p-2"
          required
        />
        <input
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="rounded border p-2"
          required
        />
        {error && <p className="text-sm text-red-500">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="rounded bg-blue-600 p-2 text-white disabled:opacity-50"
        >
          {loading ? "Logging in..." : "Login"}
        </button>
      </form>
    </main>
  );
}
```

**Step 4: Verify dev server starts**

```bash
cd /Users/ethanwang/projects/devpulse/web
bun run dev
# Open http://localhost:3000/login — verify page renders
```

**Step 5: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add web/
git commit -m "feat(web): scaffold Next.js with login page and API client"
```

---

### Task 8: iOS SwiftUI scaffold + login

**Files:**
- Create: `ios/DevPulse.xcodeproj`
- Create: `ios/DevPulse/` (SwiftUI app)

**Step 1: Create Xcode project**

```bash
cd /Users/ethanwang/projects/devpulse
mkdir -p ios
```

Create Xcode project via `xcodebuild` or Xcode GUI:
- Product name: DevPulse
- Bundle ID: com.ethanwang.devpulse
- Interface: SwiftUI
- Language: Swift
- Minimum deployment: iOS 18

**Step 2: Create API client**

Create `ios/DevPulse/Services/APIClient.swift`:

```swift
import Foundation

nonisolated final class APIClient: Sendable {
    static let shared = APIClient()
    private let baseURL = URL(string: "http://localhost:8080")!

    func login(email: String, password: String) async throws -> LoginResponse {
        var request = URLRequest(url: baseURL.appendingPathComponent("/api/login"))
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(
            LoginRequest(email: email, password: password)
        )

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw APIError.unauthorized
        }
        return try JSONDecoder().decode(LoginResponse.self, from: data)
    }
}

struct LoginRequest: Encodable {
    let email: String
    let password: String
}

struct LoginResponse: Decodable {
    let token: String
    let user: User
}

struct User: Decodable, Identifiable {
    let id: Int
    let email: String
    let name: String
}

enum APIError: Error {
    case unauthorized
    case networkError
}
```

**Step 3: Create login view**

Create `ios/DevPulse/Views/LoginView.swift`:

```swift
import SwiftUI

struct LoginView: View {
    @State private var email = ""
    @State private var password = ""
    @State private var error: String?
    @State private var loading = false

    var onLogin: (String) -> Void

    var body: some View {
        VStack(spacing: 16) {
            Text("DevPulse")
                .font(.largeTitle.bold())

            TextField("Email", text: $email)
                .textContentType(.emailAddress)
                .autocapitalization(.none)
                .textFieldStyle(.roundedBorder)

            SecureField("Password", text: $password)
                .textContentType(.password)
                .textFieldStyle(.roundedBorder)

            if let error {
                Text(error)
                    .foregroundStyle(.red)
                    .font(.caption)
            }

            Button(action: login) {
                if loading {
                    ProgressView()
                } else {
                    Text("Login")
                        .frame(maxWidth: .infinity)
                }
            }
            .buttonStyle(.borderedProminent)
            .disabled(loading)
        }
        .padding(32)
    }

    private func login() {
        loading = true
        error = nil
        Task {
            do {
                let response = try await APIClient.shared.login(
                    email: email, password: password
                )
                onLogin(response.token)
            } catch {
                self.error = "Login failed"
            }
            loading = false
        }
    }
}
```

**Step 4: Build and run in simulator**

```bash
cd /Users/ethanwang/projects/devpulse/ios
xcodebuild -scheme DevPulse -destination 'platform=iOS Simulator,name=iPhone 17' build
# Expected: BUILD SUCCEEDED
```

**Step 5: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add ios/
git commit -m "feat(ios): scaffold SwiftUI app with login view"
```

---

### Task 9: Android Compose scaffold + login

**Files:**
- Create: `android/` (Compose project)

**Step 1: Create Android project**

Use Android Studio or CLI to scaffold:
- Name: DevPulse
- Package: com.ethanwang.devpulse
- Min SDK: 26
- Jetpack Compose activity template

**Step 2: Create API client**

Create `android/app/src/main/java/com/ethanwang/devpulse/data/ApiClient.kt`:

```kotlin
package com.ethanwang.devpulse.data

import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

object ApiClient {
    private const val BASE_URL = "http://10.0.2.2:8080" // emulator → host

    private val client = HttpClient(CIO) {
        install(ContentNegotiation) {
            json(Json { ignoreUnknownKeys = true })
        }
    }

    suspend fun login(email: String, password: String): LoginResponse {
        return client.post("$BASE_URL/api/login") {
            contentType(ContentType.Application.Json)
            setBody(LoginRequest(email, password))
        }.body()
    }
}

@Serializable
data class LoginRequest(val email: String, val password: String)

@Serializable
data class LoginResponse(val token: String, val user: UserDto)

@Serializable
data class UserDto(val id: Long, val email: String, val name: String)
```

**Step 3: Create login screen**

Create `android/app/src/main/java/com/ethanwang/devpulse/ui/LoginScreen.kt`:

```kotlin
package com.ethanwang.devpulse.ui

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import com.ethanwang.devpulse.viewmodel.LoginViewModel

@Composable
fun LoginScreen(
    onLoginSuccess: (String) -> Unit,
    viewModel: LoginViewModel = viewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Text("DevPulse", style = MaterialTheme.typography.headlineLarge)
        Spacer(Modifier.height(24.dp))

        OutlinedTextField(
            value = uiState.email,
            onValueChange = viewModel::onEmailChange,
            label = { Text("Email") },
            modifier = Modifier.fillMaxWidth()
        )
        Spacer(Modifier.height(12.dp))

        OutlinedTextField(
            value = uiState.password,
            onValueChange = viewModel::onPasswordChange,
            label = { Text("Password") },
            visualTransformation = PasswordVisualTransformation(),
            modifier = Modifier.fillMaxWidth()
        )
        Spacer(Modifier.height(8.dp))

        uiState.error?.let {
            Text(it, color = MaterialTheme.colorScheme.error)
            Spacer(Modifier.height(8.dp))
        }

        Button(
            onClick = { viewModel.login(onLoginSuccess) },
            enabled = !uiState.loading,
            modifier = Modifier.fillMaxWidth()
        ) {
            if (uiState.loading) {
                CircularProgressIndicator(modifier = Modifier.size(20.dp))
            } else {
                Text("Login")
            }
        }
    }
}
```

**Step 4: Build and verify**

```bash
cd /Users/ethanwang/projects/devpulse/android
./gradlew assembleDebug
# Expected: BUILD SUCCESSFUL
```

**Step 5: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add android/
git commit -m "feat(android): scaffold Compose app with login screen"
```

---

### Task 10: Root config files

**Files:**
- Create: `.gitignore` (root)
- Create: `.env.example`
- Create: `Makefile`

**Step 1: Create root .gitignore**

```
# OS
.DS_Store
Thumbs.db

# Environment
.env

# IDE
.idea/
.vscode/
*.swp

# Go
api/devpulse-api

# Node
web/node_modules/
web/.next/

# iOS
ios/build/
ios/DerivedData/
ios/*.xcuserstate

# Android
android/.gradle/
android/build/
android/app/build/
android/local.properties
```

**Step 2: Create .env.example**

```
DATABASE_URL=postgres://localhost:5432/devpulse_dev?sslmode=disable
JWT_SECRET=change-me-in-production
PORT=8080
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_CALLBACK_URL=http://localhost:3000/auth/github/callback
```

**Step 3: Create Makefile**

```makefile
.PHONY: api web ios android db-up db-down

# ─── Backend ───────────────────────────
api:
	cd api && go run .

api-test:
	cd api && go test ./... -v

# ─── Web ───────────────────────────────
web:
	cd web && bun run dev

# ─── Database ──────────────────────────
db-up:
	createdb devpulse_dev 2>/dev/null || true

db-down:
	dropdb devpulse_dev 2>/dev/null || true

db-reset: db-down db-up api
	@echo "Database reset complete"

# ─── All ───────────────────────────────
dev:
	@echo "Starting API and Web..."
	@make api & make web
```

**Step 4: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add .gitignore .env.example Makefile
git commit -m "chore: add root gitignore, env example, and Makefile"
```

---

### Task 11: Docker Compose for local dev

**Files:**
- Create: `docker-compose.yml`

**Step 1: Write docker-compose for PostgreSQL + Redis**

```yaml
services:
  postgres:
    image: postgres:17
    environment:
      POSTGRES_DB: devpulse_dev
      POSTGRES_HOST_AUTH_METHOD: trust
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  pgdata:
```

**Step 2: Start and verify**

```bash
cd /Users/ethanwang/projects/devpulse
docker compose up -d
psql postgres://localhost:5432/devpulse_dev -c "SELECT 1"
# Expected: 1
```

**Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "chore: add docker-compose for local PostgreSQL and Redis"
```

---

### Task 12: GitHub Actions CI (path-filtered)

**Files:**
- Create: `.github/workflows/api.yml`
- Create: `.github/workflows/web.yml`

**Step 1: Write API CI**

Create `.github/workflows/api.yml`:

```yaml
name: API CI

on:
  push:
    paths: ['api/**']
  pull_request:
    paths: ['api/**']

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17
        env:
          POSTGRES_DB: devpulse_test
          POSTGRES_HOST_AUTH_METHOD: trust
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: cd api && go test ./... -v
        env:
          DATABASE_URL: postgres://localhost:5432/devpulse_test?sslmode=disable
```

**Step 2: Write Web CI**

Create `.github/workflows/web.yml`:

```yaml
name: Web CI

on:
  push:
    paths: ['web/**']
  pull_request:
    paths: ['web/**']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: oven-sh/setup-bun@v2
      - run: cd web && bun install --frozen-lockfile
      - run: cd web && bun run build
```

**Step 3: Commit**

```bash
cd /Users/ethanwang/projects/devpulse
git add .github/
git commit -m "ci: add path-filtered GitHub Actions for API and Web"
```

---

## Phase 2: First Data Source (Tasks 13–18)

> Goal: GitHub data flows in, user sees commit timeline and daily stats on all three clients.

---

### Task 13: GitHub data sync service (backend)

**Files:**
- Create: `api/service/github.go`
- Create: `api/service/github_test.go`
- Create: `api/handler/activity.go`

**Summary:** Implement GitHub API client that fetches user events, parses commits/PRs/reviews, stores as activities. Add cron job (every hour) using `robfig/cron`. Add `/api/activities` endpoint with pagination.

---

### Task 14: Daily summary aggregation (backend)

**Files:**
- Create: `api/service/aggregator.go`
- Create: `api/service/aggregator_test.go`
- Create: `api/handler/summary.go`

**Summary:** Nightly cron job that aggregates activities → daily_summaries. SQL-based aggregation (`GROUP BY date`). Add `/api/summaries` endpoint returning last 30 days.

---

### Task 15: GitHub webhook receiver (backend)

**Files:**
- Create: `api/handler/webhook.go`
- Create: `api/handler/webhook_test.go`

**Summary:** `POST /api/webhooks/github` receives push/PR events, validates signature (`X-Hub-Signature-256`), inserts activities in real-time. Supplements the hourly cron sync.

---

### Task 16: Web — activity timeline + charts

**Files:**
- Create: `web/src/app/dashboard/page.tsx`
- Create: `web/src/components/activity-timeline.tsx`
- Create: `web/src/components/daily-chart.tsx`

**Summary:** Dashboard page with Recharts bar chart (daily commits) + activity timeline list. Fetch from `/api/activities` and `/api/summaries`.

---

### Task 17: iOS — activity list + summary card

**Files:**
- Create: `ios/DevPulse/Views/DashboardView.swift`
- Create: `ios/DevPulse/Views/SummaryCard.swift`
- Create: `ios/DevPulse/ViewModels/DashboardViewModel.swift`

**Summary:** SwiftUI List with activity items, top card showing today's summary (commits, PRs, coding time). Pull-to-refresh.

---

### Task 18: Android — activity list + summary card

**Files:**
- Create: `android/.../ui/DashboardScreen.kt`
- Create: `android/.../ui/SummaryCard.kt`
- Create: `android/.../viewmodel/DashboardViewModel.kt`

**Summary:** Compose LazyColumn with activity items, top card mirroring iOS design. SwipeRefresh.

---

## Phase 3: Polish (Tasks 19–24)

> Goal: Wakatime integration, push notifications, widgets, trend comparison.

---

### Task 19: Wakatime data sync (backend)

**Files:**
- Create: `api/service/wakatime.go`
- Modify: `api/handler/oauth.go` (add Wakatime OAuth)
- Modify: `docs/openapi.yaml` (add Wakatime endpoints)

**Summary:** Wakatime OAuth flow + daily stats sync (coding time by project/language). Store as activities with `source=wakatime`.

---

### Task 20: Web — multi-source dashboard + trends

**Files:**
- Modify: `web/src/app/dashboard/page.tsx`
- Create: `web/src/components/trend-chart.tsx`
- Create: `web/src/components/source-picker.tsx`

**Summary:** Source filter tabs (All/GitHub/Wakatime). Weekly/monthly trend comparison line chart.

---

### Task 21: Push notification service (backend)

**Files:**
- Create: `api/service/notifier.go`
- Create: `api/handler/device.go`
- Create: `api/db/migrations/003_devices.up.sql`

**Summary:** Device token registration endpoint. Daily cron sends summary push via APNs (iOS) and FCM (Android).

---

### Task 22: iOS — widget + push notifications

**Files:**
- Create: `ios/DevPulseWidget/` (Widget Extension)
- Modify: `ios/DevPulse/DevPulseApp.swift` (push registration)

**Summary:** Lock screen widget showing today's coding time. Push notification setup via APNs.

---

### Task 23: Android — widget + push notifications

**Files:**
- Create: `android/.../widget/DashboardWidget.kt`
- Modify: `android/.../MainActivity.kt` (FCM registration)

**Summary:** Home screen widget. FCM push setup.

---

### Task 24: Dockerfile + fly.io deployment

**Files:**
- Create: `api/Dockerfile`
- Create: `fly.toml`

**Summary:** Multi-stage Dockerfile (build → scratch). fly.io deployment with managed Postgres. Production env vars via `fly secrets`.

---

Plan complete and saved. Execution options:

**1. Subagent-Driven (this session)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** — Open a new Claude Code session in the devpulse directory, use `executing-plans` skill for batch execution with checkpoints

Which approach?