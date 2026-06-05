// Thin fetch wrapper around the Go backend.
// Uses VITE_API_URL from .env files to support separate frontend/backend deployments.

export interface ApiError {
  message: string
  status: number
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem('token')
  
  // 1. Leemos la URL base del backend desde nuestro archivo .env
  // Si por algún motivo no existe la variable, usamos un string vacío por defecto
  const baseUrl = import.meta.env.VITE_API_URL || '';

  // 2. Inyectamos la URL base justo antes del /api
  const response = await fetch(`${baseUrl}/api${path}`, {
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