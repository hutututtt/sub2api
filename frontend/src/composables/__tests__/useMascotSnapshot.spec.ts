import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, type EffectScope } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useMascotSnapshot } from '../useMascotSnapshot'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { getMyPlatformQuotas, getProfile } from '@/api/user'
import { getStats } from '@/api/admin/dashboard'
import type { DashboardStats, User } from '@/types'

vi.mock('@/api/usage', () => ({ usageAPI: { getDashboardStats: vi.fn() } }))
vi.mock('@/api', () => ({ authAPI: {}, isTotp2FARequired: vi.fn(), passkeyAPI: {} }))
vi.mock('@/api/user', () => ({ getProfile: vi.fn(), getMyPlatformQuotas: vi.fn() }))
vi.mock('@/api/admin/dashboard', () => ({ getStats: vi.fn() }))

const user = { id: 7, username: 'Traveler', role: 'user', balance: 12 } as User
const usage = { today_tokens: 53_900_000, today_requests: 774, total_tokens: 1_700_000_000 } as UserDashboardStats
let scope: EffectScope

function setup() {
  scope = effectScope()
  return scope.run(useMascotSnapshot)!
}

describe('useMascotSnapshot', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
    useAuthStore().user = { ...user }
    vi.mocked(usageAPI.getDashboardStats).mockResolvedValue(usage)
    vi.mocked(getProfile).mockResolvedValue({ ...user, balance: 8 })
    vi.mocked(getMyPlatformQuotas).mockResolvedValue({ platform_quotas: [] })
  })

  afterEach(() => scope?.stop())

  it('refreshes balance and renders usage even when optional quotas fail', async () => {
    vi.mocked(getMyPlatformQuotas).mockRejectedValue(new Error('unavailable'))
    const snapshot = setup()
    await snapshot.refresh()
    expect(snapshot.stats.value?.today_tokens).toBe(usage.today_tokens)
    expect(snapshot.balance.value).toBe(8)
    expect(snapshot.partial.value).toBe(true)
    expect(snapshot.failed.value).toBe(false)
    expect(snapshot.updatedAt.value).toBeInstanceOf(Date)
  })

  it('does not report cached balance as freshly loaded when profile fails', async () => {
    vi.mocked(getProfile).mockRejectedValue(new Error('profile unavailable'))
    const snapshot = setup()
    await snapshot.refresh()
    expect(snapshot.stats.value).not.toBeNull()
    expect(snapshot.balance.value).toBeNull()
    expect(snapshot.partial.value).toBe(true)
  })

  it('preserves the previous snapshot and timestamp when a later usage refresh fails', async () => {
    const snapshot = setup()
    await snapshot.refresh()
    const previousTime = snapshot.updatedAt.value
    vi.mocked(usageAPI.getDashboardStats).mockRejectedValue(new Error('offline'))
    await snapshot.refresh()
    expect(snapshot.stats.value?.today_requests).toBe(774)
    expect(snapshot.updatedAt.value).toBe(previousTime)
    expect(snapshot.failed.value).toBe(true)
  })

  it('ignores a previous account response and allows the new account to refresh', async () => {
    let finishOld!: (value: UserDashboardStats) => void
    vi.mocked(usageAPI.getDashboardStats).mockImplementationOnce(() => new Promise(resolve => { finishOld = resolve }))
    const snapshot = setup()
    const oldRequest = snapshot.refresh()
    useAuthStore().user = { ...user, id: 8 }
    expect(snapshot.loading.value).toBe(false)
    await snapshot.refresh()
    finishOld({ ...usage, today_requests: 999 })
    await oldRequest
    expect(snapshot.stats.value?.today_requests).toBe(774)
    expect(snapshot.loading.value).toBe(false)
  })

  it('clears account data on logout and discards an outstanding response', async () => {
    let finish!: (value: UserDashboardStats) => void
    vi.mocked(usageAPI.getDashboardStats).mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const snapshot = setup()
    const pending = snapshot.refresh()
    useAuthStore().user = null
    finish(usage)
    await pending
    expect(snapshot.stats.value).toBeNull()
    expect(snapshot.balance.value).toBeNull()
    expect(snapshot.updatedAt.value).toBeNull()
  })

  it('prevents overlapping refreshes', async () => {
    const snapshot = setup()
    await Promise.all([snapshot.refresh(), snapshot.refresh()])
    expect(usageAPI.getDashboardStats).toHaveBeenCalledTimes(1)
  })

  it('loads actual administrator statistics without requesting user endpoints', async () => {
    useAuthStore().user = { ...user, role: 'admin' }
    vi.mocked(getStats).mockResolvedValue({ today_requests: 123, total_users: 10 } as DashboardStats)
    const snapshot = setup()
    await snapshot.refresh()
    expect(snapshot.stats.value?.today_requests).toBe(123)
    expect(usageAPI.getDashboardStats).not.toHaveBeenCalled()
    expect(getProfile).not.toHaveBeenCalled()
    expect(getMyPlatformQuotas).not.toHaveBeenCalled()
  })
})
