.PHONY: infra infra-down backend frontend dev dev-down test test-backend test-frontend test-e2e

# ── Infrastructure ────────────────────────────────────────────────────────────

# Start postgres / redis / kafka (detached)
infra:
	docker compose up -d
	@echo "Waiting for postgres to be healthy..."
	@until docker inspect banking-postgres --format='{{.State.Health.Status}}' 2>/dev/null | grep -q healthy; do sleep 1; done
	@echo "All infrastructure is healthy."

# Stop all infrastructure containers
infra-down:
	docker compose down

# ── Application ───────────────────────────────────────────────────────────────

# Run the Go backend (blocks — run in a separate terminal)
backend:
	cd backend && go run ./cmd/server/main.go

# Run the Vite frontend dev server (blocks — run in a separate terminal)
frontend:
	cd frontend && npm run dev

# ── Tests ─────────────────────────────────────────────────────────────────────

test-backend:
	cd backend && go test ./... -race -v

test-integration:
	cd backend && go test -tags integration ./integration/... -v

test-frontend:
	cd frontend && npm test

test-e2e:
	cd frontend && npm run test:e2e

test: test-backend test-frontend
	@echo "All tests passed."
