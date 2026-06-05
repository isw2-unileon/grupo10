import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

import { api } from '@/services/api'

// Mirrors the JSON returned by the backend `users` slice.
export interface User {
  id: string
  email: string
  role: string
}

interface AuthResponse {
  token: string
  user: User
}

// Roles seeded by the backend migrations. There is no "admin" role.
export type Role = 'student' | 'teacher'

export interface RegisterPayload {
  name: string
  email: string
  password: string
  role: Role
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const user = ref<User | null>(null)

  const isAuthenticated = computed(() => token.value !== null)

  function setSession(payload: AuthResponse) {
    token.value = payload.token
    user.value = payload.user
    localStorage.setItem('token', payload.token)
  }

  // POST /api/login → { token, user }
  async function login(email: string, password: string) {
    const payload = await api.post<AuthResponse>('/login', { email, password })
    setSession(payload)
  }

  // POST /api/register → { token, user }
  // The backend requires name, email, password and role (student | teacher).
  async function register(input: RegisterPayload) {
    const payload = await api.post<AuthResponse>('/register', input)
    setSession(payload)
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
  }

  return { token, user, isAuthenticated, login, register, logout }
})
