# ADR-001: Monorepo with Go backend and frontend

## Status

Accepted

## Date

2026-06-04

## Context

We need a repository structure that supports developing a Go API backend and a
web frontend in the same codebase, with shared CI pipelines and clear boundaries
between the two. The layout follows the reference `isw2-unileon/proyect-scaffolding`.

## Decision

Use a monorepo with top-level `backend/`, `frontend/` and `e2e/` directories.

- The Go module lives at the repository **root** (`go.mod`) and covers the code
  under `backend/` (`backend/cmd/server` for the entry point, `backend/internal`
  for the domain packages). Imports are rooted at
  `github.com/isw2-unileon/grupo10/backend/...`.
- `frontend/` is a Vue 3 + Vite single-page application (TypeScript, Vue Router,
  Pinia) and manages its own dependencies (`package.json`). It talks to the
  backend over `/api/*`, proxied to `http://localhost:8080` in development.
- `e2e/` holds end-to-end tests (Playwright).
- CI workflows use path filters so changes to one area don't needlessly trigger
  the others.

## Consequences

- Atomic commits can span backend, frontend and e2e.
- A single `go.mod` at the root keeps backend tooling simple (one `go build ./...`).
- Developers need both Go and Node.js toolchains installed locally.
- Database migrations live in `backend/migrations` and run automatically on startup.
