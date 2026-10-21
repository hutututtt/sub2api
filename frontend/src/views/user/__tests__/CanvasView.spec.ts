import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { reactive } from 'vue'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter, RouterView } from 'vue-router'
import CanvasView from '../CanvasView.vue'
import { loadCanvasKeys, loadCanvasModels } from '@/api/canvas'
import zh from '@/i18n/locales/zh/canvas'

const auth = reactive({ user: { id: 7 } as { id: number } | undefined, refreshUser: vi.fn().mockResolvedValue({}) })
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/api/canvas', () => ({ loadCanvasKeys: vi.fn(), loadCanvasModels: vi.fn() }))
vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<div><slot name="header-left" /><slot /></div>' } }))
const key = { id: 2, user_id: 7, key: 'fixture-key', name: 'My channel', group: { name: 'OpenAI', platform: 'openai' } }
let wrapper: VueWrapper
async function setup() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/canvas', component: CanvasView }, { path: '/keys', component: { template: '<div />' } }] })
  await router.push('/canvas')
  const messages = { canvas: Object.fromEntries(Object.entries(zh.canvas).map(([key, value]) => [key, () => value])) }
  wrapper = mount(RouterView, { attachTo: document.body, global: { plugins: [router, createI18n({ legacy: false, locale: 'zh', messages: { zh: messages } })] } })
  await flushPromises()
}
beforeEach(() => {
  vi.clearAllMocks()
  auth.user = { id: 7 }
  vi.mocked(loadCanvasKeys).mockResolvedValue([key] as never)
  vi.mocked(loadCanvasModels).mockResolvedValue([{ name: 'gpt-image-1', capability: 'image' }])
})
afterEach(() => { wrapper?.unmount(); document.body.innerHTML = '' })

describe('CanvasView session', () => {
  it('keeps secrets out of iframe URLs and only initializes the current frame', async () => {
    await setup()
    const iframe = wrapper.find('iframe').element as HTMLIFrameElement
    const post = vi.spyOn(iframe.contentWindow!, 'postMessage')
    expect(iframe.src).not.toContain('fixture-key')
    window.dispatchEvent(new MessageEvent('message', { origin: 'https://other.invalid', source: iframe.contentWindow, data: { type: 'sub2api:canvas:boot' } }))
    expect(post).not.toHaveBeenCalled()
    window.dispatchEvent(new MessageEvent('message', { origin: window.location.origin, source: window, data: { type: 'sub2api:canvas:boot' } }))
    expect(post).not.toHaveBeenCalled()
    window.dispatchEvent(new MessageEvent('message', { origin: window.location.origin, source: iframe.contentWindow, data: { type: 'sub2api:canvas:boot' } }))
    expect(post).toHaveBeenCalledWith(expect.objectContaining({ session: expect.objectContaining({ userId: 7, apiKey: 'fixture-key' }) }), window.location.origin)
  })
  it('removes the frame and drops channels on logout', async () => {
    await setup()
    expect(wrapper.find('iframe').exists()).toBe(true)
    auth.user = undefined
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.find('select').exists()).toBe(false)
  })
  it('syncs appearance changes without replacing the active canvas', async () => {
    await setup()
    const iframe = wrapper.find('iframe').element as HTMLIFrameElement
    const post = vi.spyOn(iframe.contentWindow!, 'postMessage')
    try {
      document.documentElement.classList.add('dark')
      await flushPromises()
      expect(post).toHaveBeenCalledWith({ type: 'sub2api:canvas:theme', theme: 'dark' }, window.location.origin)
      expect(wrapper.find('iframe').element).toBe(iframe)
    } finally {
      document.documentElement.classList.remove('dark')
    }
  })
  it('ignores catalog results received after logout', async () => {
    let resolve!: (value: never) => void
    vi.mocked(loadCanvasModels).mockImplementationOnce(() => new Promise(done => { resolve = done }))
    await setup()
    auth.user = undefined
    await flushPromises()
    resolve([{ name: 'gpt-image-1', capability: 'image' }] as never)
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
  })
  it('refreshes balance on completion and locks channel selection while generating', async () => {
    await setup()
    const iframe = wrapper.find('iframe').element as HTMLIFrameElement
    const send = (type: string, extra = {}) => window.dispatchEvent(new MessageEvent('message', { origin: window.location.origin, source: iframe.contentWindow, data: { type, ...extra } }))
    send('sub2api:canvas:busy', { busy: true })
    await flushPromises()
    expect(wrapper.find('select').attributes('disabled')).toBeDefined()
    send('sub2api:canvas:settled')
    expect(auth.refreshUser).toHaveBeenCalledOnce()
  })
  it('shows an actionable state when models are unavailable', async () => {
    vi.mocked(loadCanvasModels).mockResolvedValue([])
    await setup()
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.text()).toContain(zh.canvas.noModels)
  })
})
