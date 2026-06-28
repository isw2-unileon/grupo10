// Base URL of the backend API, injected at build time from VITE_API_URL.
// Empty in local dev, where Vite's proxy forwards /api to the backend (:8080);
// in production it points to the deployed backend so absolute calls reach it.
export const API_BASE = import.meta.env.VITE_API_URL || ''
