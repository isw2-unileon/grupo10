import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

import RegisterView from '../RegisterView.vue'

// Hoisted so the vue-router mock factory can reference it.
const { push } = vi.hoisted(() => ({ push: vi.fn() }))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))

function mountView() {
  return mount(RegisterView, {
    global: {
      // RouterLink is provided by the router plugin in the real app; stub it here.
      stubs: { RouterLink: true },
    },
  })
}

describe('RegisterView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    push.mockClear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows an error and does not call the API when passwords do not match', async () => {
    const fetchSpy = vi.spyOn(global, 'fetch')
    const wrapper = mountView()

    await wrapper.find('input[type=email]').setValue('student@unileon.es')
    const passwords = wrapper.findAll('input[type=password]')
    await passwords[0].setValue('123456')
    await passwords[1].setValue('654321')
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.find('.error').text()).toBe('Passwords do not match')
    expect(fetchSpy).not.toHaveBeenCalled()
    expect(push).not.toHaveBeenCalled()
  })

  it('registers and redirects home when passwords match', async () => {
    const fakeResponse = {
      token: 'jwt-token',
      user: { id: 'uuid-1', email: 'student@unileon.es', role: 'student' },
    }
    vi.spyOn(global, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(fakeResponse), { status: 201 }),
    )
    const wrapper = mountView()

    await wrapper.find('input[type=email]').setValue('student@unileon.es')
    const passwords = wrapper.findAll('input[type=password]')
    await passwords[0].setValue('123456')
    await passwords[1].setValue('123456')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(global.fetch).toHaveBeenCalledWith('/api/register', expect.anything())
    expect(push).toHaveBeenCalledWith({ name: 'home' })
    expect(localStorage.getItem('token')).toBe('jwt-token')
  })
})
