import { computed, onScopeDispose, ref, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { getMyPlatformQuotas, getProfile } from '@/api/user'
import { getStats } from '@/api/admin/dashboard'
import type { DashboardStats, PlatformQuotaItem } from '@/types'

export function useMascotSnapshot() {
  const auth = useAuthStore()
  const identity = computed(() => `${auth.user?.id ?? ''}:${auth.user?.role ?? ''}`)
  const stats = ref<UserDashboardStats | DashboardStats | null>(null)
  const balance = ref<number | null>(null)
  const quotas = ref<PlatformQuotaItem[]>([])
  const loading = ref(false)
  const failed = ref(false)
  const partial = ref(false)
  const updatedAt = ref<Date | null>(null)
  let generation = 0

  watch(identity, () => {
    generation++
    stats.value = null
    balance.value = null
    quotas.value = []
    loading.value = false
    failed.value = false
    partial.value = false
    updatedAt.value = null
  }, { flush: 'sync' })

  async function refresh() {
    if (!auth.user || loading.value) return
    const requestGeneration = generation
    const admin = auth.isAdmin
    loading.value = true

    try {
      // Optional data must not prevent the main usage snapshot from rendering.
      if (admin) {
        const summary = await getStats()
        if (requestGeneration !== generation) return
        stats.value = summary
        updatedAt.value = new Date()
        failed.value = false
        partial.value = false
      } else {
        const [summary, profile, limits] = await Promise.allSettled([
          usageAPI.getDashboardStats(), getProfile(), getMyPlatformQuotas()
        ])
        if (requestGeneration !== generation) return
        failed.value = summary.status === 'rejected'
        partial.value = profile.status === 'rejected' || limits.status === 'rejected'
        if (summary.status === 'fulfilled') {
          stats.value = summary.value
          updatedAt.value = new Date()
        }
        if (profile.status === 'fulfilled') balance.value = profile.value.balance
        if (limits.status === 'fulfilled') quotas.value = limits.value.platform_quotas ?? []
      }
    } catch {
      if (requestGeneration === generation) failed.value = true
    } finally {
      if (requestGeneration === generation) loading.value = false
    }
  }

  onScopeDispose(() => { generation++ })

  return { stats, balance, quotas, loading, failed, partial, updatedAt, refresh }
}
