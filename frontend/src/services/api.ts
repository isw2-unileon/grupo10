// Thin fetch wrapper around the Go backend. Requests go to `/api/*`, which
// the Vite dev server proxies to http://localhost:8080 (see vite.config.ts).
// In production the frontend is served behind the same origin as the API.

export interface ApiError {
  message: string
  status: number
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('token')

  const response = await fetch(`/api${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  })

  if (!response.ok) {
    const error: ApiError = {
      message: await response.text().catch(() => response.statusText),
      status: response.status,
    }
    throw error
  }

  // 204 No Content has an empty body.
  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
}
