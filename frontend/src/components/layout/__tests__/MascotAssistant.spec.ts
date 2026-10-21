import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter } from 'vue-router'
import { nextTick } from 'vue'
import MascotAssistant from '../MascotAssistant.vue'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { getMyPlatformQuotas, getProfile } from '@/api/user'
import zh from '@/i18n/locales/zh/mascot'
import en from '@/i18n/locales/en/mascot'
import type { User } from '@/types'

vi.mock('@/api', () => ({ authAPI: {}, isTotp2FARequired: vi.fn(), passkeyAPI: {} }))
vi.mock('@/api/usage', () => ({ usageAPI: { getDashboardStats: vi.fn() } }))
vi.mock('@/api/user', () => ({ getProfile: vi.fn(), getMyPlatformQuotas: vi.fn() }))
vi.mock('@/api/admin/dashboard', () => ({ getStats: vi.fn() }))

let wrapper: VueWrapper | undefined
const user = { id: 7, username: 'Traveler', role: 'user', balance: 99 } as User

async function setup(locale = 'zh') {
  const pinia = createPinia()
  setActivePinia(pinia)
  const auth = useAuthStore()
  auth.user = { ...user }
  auth.token = 'test-session'
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }] })
  await router.push('/dashboard')
  const messages = Object.fromEntries(Object.entries({ zh, en }).map(([language, strings]) => [
    language, { mascot: Object.fromEntries(Object.entries(strings).map(([key, text]) => [key, () => text])) }
  ]))
  const i18n = createI18n({ legacy: false, locale, messages })
  wrapper = mount(MascotAssistant, { attachTo: document.body, global: { plugins: [pinia, router, i18n] } })
  return { auth, router }
}

async function click(label: string) {
  const button = document.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)
  expect(button).not.toBeNull()
  button!.click()
  await flushPromises()
  await vi.advanceTimersByTimeAsync(200)
}

function pointer(target: EventTarget, type: string, x: number, y: number, pointerType = 'mouse', pointerId = 1) {
  const event = new MouseEvent(type, { bubbles: true, cancelable: true, clientX: x, clientY: y, button: 0 })
  Object.assign(event, { pointerId, pointerType, isPrimary: true })
  target.dispatchEvent(event)
}

async function dragTo(x: number, y: number, pointerType = 'mouse') {
  const trigger = document.querySelector<HTMLButtonElement>('.mascot-trigger')!
  const root = document.querySelector<HTMLElement>('.mascot-assistant')!
  const startX = parseFloat(root.style.left) + 20
  const startY = parseFloat(root.style.top) + 20
  pointer(trigger, 'pointerdown', startX, startY, pointerType)
  pointer(window, 'pointermove', x + 20, y + 20, pointerType)
  await nextTick()
  pointer(window, 'pointerup', x + 20, y + 20, pointerType)
  trigger.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 1 }))
  await flushPromises()
  await vi.advanceTimersByTimeAsync(200)
}

describe('MascotAssistant', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.resetAllMocks()
    localStorage.clear()
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1024 })
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 768 })
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.classList.contains('mascot-assistant')) {
        const compact = this.classList.contains('mascot-compact')
        const mobile = window.innerWidth <= 640
        return new DOMRect(parseFloat(this.style.left) || 0, parseFloat(this.style.top) || 0, compact ? 44 : mobile ? 92 : 112, compact ? 44 : mobile ? 132 : 162)
      }
      if (this.classList.contains('mascot-panel')) {
        return new DOMRect(0, 0, Math.min(360, window.innerWidth - 24), Math.min(420, parseFloat(this.style.maxHeight) || 420))
      }
      return new DOMRect()
    })
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    vi.mocked(usageAPI.getDashboardStats).mockResolvedValue({
      today_tokens: 53_900_000, today_requests: 774, active_api_keys: 4, total_api_keys: 4,
      total_tokens: 1_700_000_000, today_actual_cost: 3.193, rpm: 0
    } as UserDashboardStats)
    vi.mocked(getProfile).mockResolvedValue({ ...user, balance: 11.9618 })
    vi.mocked(getMyPlatformQuotas).mockResolvedValue({ platform_quotas: [] })
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = undefined
    document.body.innerHTML = ''
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('fetches only while open and visible, and releases the refresh timer on unmount', async () => {
    await setup()
    await vi.advanceTimersByTimeAsync(60_000)
    expect(usageAPI.getDashboardStats).not.toHaveBeenCalled()
    await click('打开星璃酱')
    expect(document.body.textContent).toContain('$11.9618')
    expect(document.body.textContent).toContain('4/4')
    await vi.advanceTimersByTimeAsync(60_000)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(2)
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    await vi.advanceTimersByTimeAsync(60_000)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(2)
    await click('收起星璃酱')
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    await vi.advanceTimersByTimeAsync(60_000)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(2)
    wrapper!.unmount()
    wrapper = undefined
    await vi.advanceTimersByTimeAsync(60_000)
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(2)
  })

  it('retains the chosen mode after remount and provides a restore control after hiding', async () => {
    await setup()
    await click('打开星璃酱')
    const star = [...document.querySelectorAll('button')].find(button => button.textContent?.trim() === '星标')!
    star.click()
    await nextTick()
    expect(document.querySelector('.mascot-character')).toBeNull()
    wrapper!.unmount()
    await setup()
    expect(document.querySelector('.mascot-character')).toBeNull()
    await click('打开星璃酱')
    await click('隐藏星璃酱')
    expect(document.querySelector('.mascot-panel')).toBeNull()
    await click('显示星璃酱')
    expect(document.querySelector('.mascot-character')).not.toBeNull()
    expect(document.querySelector('.mascot-panel')).not.toBeNull()
  })

  it('closes on Escape, restores focus, and closes when navigating', async () => {
    const { router } = await setup()
    await click('打开星璃酱')
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    await vi.advanceTimersByTimeAsync(200)
    expect(document.querySelector('.mascot-panel')).toBeNull()
    expect(document.activeElement?.getAttribute('aria-label')).toBe('打开星璃酱')
    await click('打开星璃酱')
    await router.push('/keys')
    await flushPromises()
    await vi.advanceTimersByTimeAsync(200)
    expect(document.querySelector('.mascot-panel')).toBeNull()
  })

  it('removes the panel when the session ends', async () => {
    const { auth } = await setup()
    await click('打开星璃酱')
    auth.user = null
    auth.token = null
    await nextTick()
    expect(document.querySelector('.mascot-assistant')).toBeNull()
  })

  it('supports English and optional quota failures without losing user statistics', async () => {
    vi.mocked(getMyPlatformQuotas).mockRejectedValue(new Error('offline'))
    await setup('en')
    await click('Open Xingli')
    expect(document.body.textContent).toContain('Some data could not be updated')
    expect(document.body.textContent).toContain('53.9M')
    expect(document.body.textContent).toContain('$11.9618')
  })

  it('drags without opening the panel and restores the saved location after remount', async () => {
    await setup()
    await flushPromises()
    await dragTo(200, 160)
    const root = document.querySelector<HTMLElement>('.mascot-assistant')!
    expect(root.style.left).toBe('200px')
    expect(root.style.top).toBe('160px')
    expect(document.querySelector('.mascot-panel')).toBeNull()
    expect(usageAPI.getDashboardStats).not.toHaveBeenCalled()
    const saved = JSON.parse(localStorage.getItem('sub2api:mascot:position:v1')!)
    expect(saved.x).toBeGreaterThan(0)
    expect(saved.x).toBeLessThan(1)
    wrapper!.unmount()
    await setup()
    await flushPromises()
    expect(document.querySelector<HTMLElement>('.mascot-assistant')!.style.left).toBe('200px')
    expect(document.querySelector<HTMLElement>('.mascot-assistant')!.style.top).toBe('160px')
    await click('打开星璃酱')
    expect(document.querySelector('.mascot-panel')).not.toBeNull()
  })

  it('clamps touch drags and opens the panel inside a mobile viewport at either corner', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 393 })
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 852 })
    await setup()
    await flushPromises()
    await dragTo(-500, -500, 'touch')
    const root = document.querySelector<HTMLElement>('.mascot-assistant')!
    expect(root.style.left).toBe('12px')
    expect(root.style.top).toBe('12px')
    await click('打开星璃酱')
    let panel = document.querySelector<HTMLElement>('.mascot-panel')!
    expect(parseFloat(root.style.left) + parseFloat(panel.style.left)).toBe(12)
    expect(parseFloat(panel.style.top)).toBe(144)
    expect(parseFloat(panel.style.maxHeight)).toBe(684)
    await dragTo(2000, 2000, 'touch')
    expect(root.style.left).toBe('289px')
    expect(root.style.top).toBe('708px')
    expect(document.querySelector('.mascot-panel')).toBeNull()
    await click('打开星璃酱')
    panel = document.querySelector<HTMLElement>('.mascot-panel')!
    expect(parseFloat(root.style.left) + parseFloat(panel.style.left)).toBe(21)
    expect(parseFloat(panel.style.top)).toBeLessThan(0)
    expect(parseFloat(root.style.top) + parseFloat(panel.style.top)).toBeGreaterThanOrEqual(12)
  })

  it('keeps restored coordinates on screen after resizing and can reset to the default corner', async () => {
    localStorage.setItem('sub2api:mascot:position:v1', JSON.stringify({ x: 0, y: 0 }))
    await setup()
    await flushPromises()
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 393 })
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 500 })
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    const root = document.querySelector<HTMLElement>('.mascot-assistant')!
    expect(root.style.left).toBe('12px')
    expect(root.style.top).toBe('12px')
    await click('打开星璃酱')
    await click('恢复右下角默认位置')
    expect(root.style.left).toBe('289px')
    expect(root.style.top).toBe('356px')
    expect(JSON.parse(localStorage.getItem('sub2api:mascot:position:v1')!)).toEqual({ x: 1, y: 1 })
  })

  it('treats small pointer movements as a click and ignores other pointers during a drag', async () => {
    await setup()
    await flushPromises()
    const trigger = document.querySelector<HTMLButtonElement>('.mascot-trigger')!
    pointer(trigger, 'pointerdown', 950, 700)
    pointer(window, 'pointermove', 200, 200, 'touch', 2)
    pointer(window, 'pointermove', 952, 702)
    pointer(window, 'pointerup', 952, 702)
    trigger.dispatchEvent(new MouseEvent('click', { bubbles: true, detail: 1 }))
    await flushPromises()
    expect(document.querySelector('.mascot-panel')).not.toBeNull()
    expect(document.querySelector<HTMLElement>('.mascot-assistant')!.style.left).toBe('896px')
    expect(JSON.parse(localStorage.getItem('sub2api:mascot:position:v1')!)).toEqual({ x: 1, y: 1 })
  })
})
