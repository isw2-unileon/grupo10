# Frontend — Learning Platform

Single-page application built with **Vue 3**, **Vite**, **TypeScript**, **Vue Router** and **Pinia**.

## Prerequisites

- Node.js `^18.0.0 || >=20.0.0`
- The Go backend running on `http://localhost:8080` (see the [root README](../README.md))

## Setup

```bash
npm ci          # or: npm install
npm run dev     # dev server at http://localhost:5173
```

During development, requests to `/api/*` are proxied to the backend at
`http://localhost:8080` (configured in `vite.config.ts`), so no CORS setup is
required.

## Scripts

| Command | Description |
|---|---|
| `npm run dev` | Start the Vite dev server |
| `npm run build` | Type-check and build for production into `dist/` |
| `npm run preview` | Preview the production build locally |
| `npm run test` | Run unit tests once (Vitest) |
| `npm run test:watch` | Run unit tests in watch mode |
| `npm run type-check` | Type-check without emitting |
| `npm run lint` | Lint and auto-fix with ESLint |

All of these are also wired into the root `Makefile`
(`make run-frontend`, `make build-frontend`, `make test`, `make lint`).

## Structure

```
frontend/
├── public/            # Static assets served as-is
├── src/
│   ├── assets/        # Global styles
│   ├── components/    # Reusable UI components
│   ├── router/        # Vue Router routes
│   ├── services/      # API client (talks to the Go backend)
│   ├── stores/        # Pinia stores (auth, …)
│   ├── views/         # Route-level pages
│   ├── App.vue        # Root component
│   └── main.ts        # App entry point
├── index.html
├── vite.config.ts     # Build + dev proxy to the backend
└── vitest.config.ts   # Unit test config
```
