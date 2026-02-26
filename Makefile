# DevPulse development commands

# --- API (Go) ---

.PHONY: api-dev api-build api-test api-lint

api-dev:
	cd api && go run ./cmd/api

api-build:
	cd api && go build -o devpulse-api ./cmd/api

api-test:
	cd api && go test ./... -count=1

api-lint:
	cd api && go vet ./...

# --- Database ---

.PHONY: db-migrate db-migrate-down db-sqlc

db-migrate:
	cd api && migrate -path db/migrations -database "$$DATABASE_URL" up

db-migrate-down:
	cd api && migrate -path db/migrations -database "$$DATABASE_URL" down 1

db-sqlc:
	cd api && sqlc generate

# --- Web (Next.js) ---

.PHONY: web-dev web-build web-lint

web-dev:
	cd web && bun run dev

web-build:
	cd web && bun run build

web-lint:
	cd web && bun run lint

# --- All ---

.PHONY: test

test: api-test
