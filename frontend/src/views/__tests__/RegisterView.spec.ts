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

    expect(wrapper.find('.error').text()).toBe('Las contraseñas no coinciden')
    expect(fetchSpy).not.toHaveBeenCalled()
    expect(push).not.toHaveBeenCalled()
  })

  it('registers and redirects home when passwords match', async () => {
    const fakeResponse = {
      token: 'jwt-token',
      user: { id: 'uuid-1', email: 'teacher@unileon.es', role: 'teacher' },
    }
    const fetchSpy = vi.spyOn(global, 'fetch').mockResolvedValue(
      new Response(JSON.stringify(fakeResponse), { status: 201 }),
    )
    const wrapper = mountView()

    await wrapper.find('input[type=text]').setValue('Ada Lovelace')
    await wrapper.find('input[type=email]').setValue('teacher@unileon.es')
    await wrapper.find('select').setValue('teacher')
    const passwords = wrapper.findAll('input[type=password]')
    await passwords[0].setValue('123456')
    await passwords[1].setValue('123456')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    // The backend requires all four fields; verify they are sent.
    const [url, options] = fetchSpy.mock.calls[0]
    expect(url).toBe('/api/register')
    expect(JSON.parse(options!.body as string)).toEqual({
      name: 'Ada Lovelace',
      email: 'teacher@unileon.es',
      password: '123456',
      role: 'teacher',
    })
    expect(push).toHaveBeenCalledWith({ name: 'home' })
    expect(localStorage.getItem('token')).toBe('jwt-token')
  })
})
