import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

import { useAuthStore } from '../auth'

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('starts unauthenticated when no token is stored', () => {
    const auth = useAuthStore()
    expect(auth.isAuthenticated).toBe(false)
    expect(auth.user).toBeNull()
  })

  it('stores token and user after a successful login', async () => {
    const fakeResponse = {
      token: 'jwt-token',
      user: { id: 'uuid-1', email: 'student@unileon.es', role: 'student' },
    }
    vi.spyOn(global, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(fakeResponse), { status: 200 }),
    )

    const auth = useAuthStore()
    await auth.login('student@unileon.es', 'secret')

    expect(auth.isAuthenticated).toBe(true)
    expect(auth.user?.email).toBe('student@unileon.es')
    expect(localStorage.getItem('token')).toBe('jwt-token')
  })

  it('clears the session on logout', async () => {
    const auth = useAuthStore()
    localStorage.setItem('token', 'jwt-token')
    auth.$patch({ token: 'jwt-token' })

    auth.logout()

    expect(auth.isAuthenticated).toBe(false)
    expect(localStorage.getItem('token')).toBeNull()
  })
})
