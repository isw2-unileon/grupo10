# Learning Platform

Integrated platform designed to optimize interaction between students and teachers, using AI to provide instant feedback on notes and reduce administrative friction.

> Ingeniería del Software II — Universidad de León · 2025–2026

---

## Prerequisites

Make sure you have the following installed before proceeding:

| Tool | Version | Download |
|---|---|---|
| Git | Any recent | [git-scm.com](https://git-scm.com/) |
| Docker Desktop | Latest | [docker.com/get-started](https://www.docker.com/get-started/) |
| Go | 1.23+ | [go.dev/dl](https://go.dev/dl/) *(only needed to run the server outside Docker)* |
| Node.js | 18+ | [nodejs.org](https://nodejs.org/) *(only needed to run the frontend outside Docker)* |

> **Windows users:** Docker Desktop already includes Docker Compose. Make sure Docker Desktop is running before executing any `docker` command.

---

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/isw2-unileon/grupo10.git
cd grupo10
```

### 2. Set up environment variables

**Linux / macOS**
```bash
cp .env.example .env
```

**Windows (Command Prompt)**
```cmd
copy .env.example .env
```

**Windows (PowerShell)**
```powershell
Copy-Item .env.example .env
```

The `.env` file should contain:

```env
DB_USER=user
DB_PASSWORD=password
DB_NAME=learning_platform
DB_PORT_HOST=5432
SERVER_PORT_HOST=8080
```

> The default values work out of the box for local development. Only change them if you have port conflicts.

### 3. Start the stack

```bash
docker compose up
```

This command works the same on Linux, macOS, and Windows. The server will automatically run the database migrations on startup.

Once running:
- Backend: [http://localhost:8080](http://localhost:8080)
- Frontend: [http://localhost:5173](http://localhost:5173)

Verify the backend is healthy:

**Linux / macOS / Windows (PowerShell 5+)**
```bash
curl http://localhost:8080/health
# → {"status":"ok"}
```

---

## Resetting the database

Useful during development if you need to wipe and recreate the schema:

```bash
docker compose down -v
docker compose up
```

The `-v` flag removes the postgres volume, giving you a clean slate. Migrations will run again automatically on the next startup.

---

## Project structure

```
grupo10/
├── backend/              # Go backend
│   ├── cmd/server/       # Entry point (composition root)
│   ├── internal/         # Domain packages (users, notes, calendar, analytics)
│   └── migrations/       # up.sql / down.sql
├── frontend/             # Vue 3 + Vite SPA (see frontend/README.md)
├── e2e/                  # End-to-end tests (Playwright)
├── docs/                 # Technical documentation and ADRs
├── go.mod                # Go module (root)
├── Makefile
├── docker-compose.yml
└── .env.example
```

---

## Running the tests

```bash
# Backend (from the repo root)
go test ./...

# Frontend
cd frontend && npm run test
```

Or run everything via the `Makefile`:

```bash
make test
```

### Integration tests (Postgres)

The repository layer has integration tests that run against a **real Postgres**
to validate the SQL against the actual schema. They are behind the `integration`
build tag and skip themselves unless `TEST_DATABASE_URL` is set, so the regular
`go test ./...` above stays database-free and never fails on a fresh clone.

CI runs them automatically against a disposable Postgres service, so you don't
need anything for pull requests.

> ⚠️ **The tests `TRUNCATE` tables.** Point `TEST_DATABASE_URL` at a **throwaway**
> Postgres, never at your dev database. Note that `docker compose up` already runs
> the dev DB on port `5432`, so the test database must use a **different port**
> (we use `5433` below) to avoid wiping your dev data.

To run them locally, spin up a disposable Postgres on its own port and point the
variable at it:

```bash
# 1) Start a throwaway Postgres for tests only (port 5433, isolated from the dev DB)
docker run -d --name lp-test-db -p 5433:5432 \
  -e POSTGRES_PASSWORD=postgres postgres:17-alpine

# 2) Run the integration tests against it
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/postgres?sslmode=disable"
go test -tags integration ./backend/internal/notes/

# 3) Tear it down when you're done
docker rm -f lp-test-db
```

The tests apply `backend/migrations/up.sql` themselves, so the throwaway database
needs no manual setup.

## Running the frontend (dev)

The frontend is a Vue 3 + Vite SPA in [`frontend/`](frontend/README.md). With
the backend running, start the dev server:

```bash
cd frontend
npm ci
npm run dev   # http://localhost:5173
```

Requests to `/api/*` are proxied to the backend at `http://localhost:8080`.

---

## Contributing

This project follows [Trunk Based Development](https://trunkbaseddevelopment.com/). Please read the following before opening a Pull Request.

**Branch naming**

Branches must be short-lived and named after the task they address:

```
feat/user-authentication
fix/note-status-transition
chore/update-dependencies
```

**Commit messages**

Write commits in English using the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
feat: add AI feedback endpoint for notes
fix: correct foreign key constraint on enrollments
chore: upgrade Go to 1.23.1
```

**Pull Requests**

- Every PR must be reviewed by at least one other team member before merging.
- Link the PR to its corresponding GitHub Projects task.
- Keep PRs small and focused — one task per PR.
- Delete the branch after merging.

**Everything in English:** code, comments, branch names, commit messages, PR descriptions, and GitHub Projects tasks.

---

## Technical documentation

Full architecture notes, data model and design decisions are available in [`/docs`](/docs).

---

## Contact

For questions about the assignment: [jferrl@unileon.es](mailto:jferrl@unileon.es)
