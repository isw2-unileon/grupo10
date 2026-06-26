.PHONY: install run-backend run-frontend build-backend build-frontend test test-integration lint e2e

## Install all dependencies
install:
	go install github.com/air-verse/air@latest
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	go mod download
	cd frontend && npm ci
	cd e2e && npm ci

## Run backend with hot reload
run-backend:
	$(shell go env GOPATH)/bin/air -c backend/.air.toml

## Run frontend dev server
run-frontend:
	cd frontend && npm run dev

## Build backend binary
build-backend:
	go build -o backend/bin/server ./backend/cmd/server

## Build frontend for production
build-frontend:
	cd frontend && npm run build

## Run all tests
test:
	go test -v -race ./...
	cd frontend && npm run test

## Run the Postgres integration tests in a throwaway container (one command).
## Spins up a disposable Postgres on port 5433, runs the tagged tests against it,
## and always tears it down afterwards (even if the tests fail). Requires Docker.
test-integration:
	@docker rm -f lp-test-db >/dev/null 2>&1 || true
	@docker run -d --name lp-test-db -p 5433:5432 -e POSTGRES_PASSWORD=postgres postgres:17-alpine >/dev/null
	@echo "Waiting for Postgres to be ready..."
	@until docker exec lp-test-db pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
	@TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable" \
		go test -tags integration -count=1 ./backend/internal/notes/; \
		status=$$?; \
		docker rm -f lp-test-db >/dev/null 2>&1; \
		exit $$status

## Run linters
lint:
	$(shell go env GOPATH)/bin/golangci-lint run
	cd frontend && npm run lint

## Run E2E tests (requires backend + frontend running)
e2e:
	cd e2e && npx playwright test
