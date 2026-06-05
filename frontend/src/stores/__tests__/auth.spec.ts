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

  it('rehydrates the user from /api/me when a token is present', async () => {
    const me = { id: 'uuid-1', email: 'student@unileon.es', role: 'student' }
    const fetchSpy = vi.spyOn(global, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(me), { status: 200 }),
    )
    localStorage.setItem('token', 'jwt-token')

    const auth = useAuthStore()
    await auth.fetchMe()

    const [url, options] = fetchSpy.mock.calls[0]
    expect(url).toBe('/api/me')
    // The stored token must be sent so the backend can identify the user.
    const headers = new Headers(options!.headers)
    expect(headers.get('Authorization')).toBe('Bearer jwt-token')
    expect(auth.user?.email).toBe('student@unileon.es')
  })

  it('clears the session when /api/me rejects the token', async () => {
    vi.spyOn(global, 'fetch').mockResolvedValue(
      new Response('unauthenticated', { status: 401 }),
    )
    localStorage.setItem('token', 'expired-token')

    const auth = useAuthStore()
    auth.$patch({ token: 'expired-token' })
    await auth.fetchMe()

    expect(auth.isAuthenticated).toBe(false)
    expect(auth.user).toBeNull()
    expect(localStorage.getItem('token')).toBeNull()
  })
})
