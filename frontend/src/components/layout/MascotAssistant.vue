<template>
  <Teleport to="body">
    <aside
      v-if="auth.isAuthenticated"
      ref="root"
      class="mascot-assistant"
      :class="{ 'mascot-compact': mode !== 'character', 'mascot-dragging': dragging }"
      :style="rootStyle"
      :aria-label="t('mascot.name')"
    >
      <Transition name="mascot-panel">
        <section
          v-if="open"
          ref="panel"
          :id="panelId"
          class="mascot-panel"
          :style="panelStyle"
          :aria-label="t('mascot.panel')"
          :aria-busy="loading"
        >
          <header class="mascot-header">
            <img :src="avatarImage" width="40" height="40" alt="" class="mascot-avatar" />
            <div class="mascot-heading">
              <p class="mascot-brand">STAR ROUTE</p>
              <h2 :title="displayName">{{ displayName }} · {{ t('mascot.name') }}</h2>
            </div>
            <div class="mascot-header-actions">
              <button type="button" class="mascot-icon" :title="t('mascot.refresh')" :aria-label="t('mascot.refresh')" :disabled="loading" @click="refresh">
                <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
              </button>
              <button type="button" class="mascot-icon" :title="t('mascot.hide')" :aria-label="t('mascot.hide')" @click="setMode('hidden')">
                <Icon name="eyeOff" size="sm" />
              </button>
              <button ref="closeButton" type="button" class="mascot-icon" :title="t('mascot.close')" :aria-label="t('mascot.close')" @click="closePanel(true)">
                <Icon name="x" size="sm" />
              </button>
            </div>
          </header>

          <div class="mascot-content">
            <p v-if="failed || partial" class="mascot-notice" role="status">
              <Icon name="exclamationCircle" size="sm" />
              <span>{{ t(failed ? 'mascot.failed' : 'mascot.partial') }}</span>
            </p>
            <div v-if="loading && !stats" class="mascot-loading" role="status">
              <Icon name="refresh" size="md" class="animate-spin" />
              {{ t('mascot.loading') }}
            </div>
            <template v-else>
              <div class="mascot-metrics">
                <div v-for="metric in metrics" :key="metric.label" class="mascot-metric" :class="`mascot-${metric.color}`">
                  <span>{{ metric.label }}</span>
                  <strong :title="metric.value">{{ metric.value }}</strong>
                </div>
              </div>
              <div class="mascot-summary">
                <div v-for="metric in summary" :key="metric.label">
                  <span>{{ metric.label }}</span>
                  <strong :title="metric.value">{{ metric.value }}</strong>
                </div>
              </div>
              <div v-if="quotaRows.length" class="mascot-quotas">
                <h3>{{ t('mascot.quotas') }}</h3>
                <div v-for="quota in quotaRows" :key="quota.key" class="mascot-quota">
                  <div><span>{{ quota.label }}</span><span>{{ money(quota.used) }} / {{ money(quota.limit) }}</span></div>
                  <progress :value="quota.used" :max="quota.limit" :aria-label="quota.label" />
                </div>
              </div>
            </template>

            <div class="mascot-display" role="group" :aria-label="t('mascot.display')">
              <span>{{ t('mascot.display') }}</span>
              <button v-for="option in modes" :key="option.value" type="button" :aria-pressed="mode === option.value" @click="setMode(option.value)">
                {{ t(option.label) }}
              </button>
              <button type="button" :title="t('mascot.resetPosition')" :aria-label="t('mascot.resetPosition')" @click="resetPosition">{{ t('mascot.reset') }}</button>
            </div>
            <nav class="mascot-links" :aria-label="t('mascot.panel')">
              <RouterLink v-for="action in actions" :key="action.path" :to="action.path" @click="closePanel()">
                <Icon :name="action.icon" size="sm" />{{ t(action.label) }}
              </RouterLink>
            </nav>
            <p v-if="updatedAt" class="mascot-updated">{{ t('mascot.updated', { time: updatedTime }) }}</p>
          </div>
        </section>
      </Transition>

      <button
        ref="trigger"
        type="button"
        class="mascot-trigger"
        :class="{ 'mascot-character': mode === 'character', 'mascot-hidden': mode === 'hidden' }"
        :title="`${triggerLabel} · ${t('mascot.dragHint')}`"
        :aria-label="triggerLabel"
        :aria-describedby="`${panelId}-drag-hint`"
        :aria-expanded="open"
        :aria-controls="panelId"
        @pointerdown="startDrag"
        @click="onTriggerClick"
      >
        <img v-if="mode === 'character'" :src="characterImage" width="112" height="162" alt="" draggable="false" />
        <Icon v-else name="sparkles" :size="mode === 'hidden' ? 'sm' : 'lg'" />
      </button>
      <span :id="`${panelId}-drag-hint`" class="sr-only">{{ t('mascot.dragHint') }}</span>
    </aside>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { onClickOutside, useEventListener, useStorage } from '@vueuse/core'
import { useAuthStore } from '@/stores/auth'
import { useMascotSnapshot } from '@/composables/useMascotSnapshot'
import { useMascotPosition } from '@/composables/useMascotPosition'
import Icon from '@/components/icons/Icon.vue'

type MascotMode = 'character' | 'star' | 'hidden'
const { t, locale } = useI18n()
const auth = useAuthStore()
const route = useRoute()
const characterImage = `${import.meta.env.BASE_URL}mascot/xingli.webp`
const avatarImage = `${import.meta.env.BASE_URL}mascot/avatar.webp`
const storedMode = useStorage<string>('sub2api:mascot:mode:v1', 'character')
const mode = computed<MascotMode>(() => storedMode.value === 'star' || storedMode.value === 'hidden' ? storedMode.value : 'character')
const open = ref(false)
const root = ref<HTMLElement>()
const trigger = ref<HTMLButtonElement>()
const panel = ref<HTMLElement>()
const closeButton = ref<HTMLButtonElement>()
const { rootStyle, panelStyle, dragging, startDrag, consumeDragClick, resetPosition } = useMascotPosition(root, trigger, panel, () => closePanel())
const panelId = `mascot-${useId()}`
const { stats, balance, quotas, loading, failed, partial, updatedAt, refresh } = useMascotSnapshot()
const displayName = computed(() => auth.user?.username?.trim() || t('mascot.guest'))
const triggerLabel = computed(() => t(mode.value === 'hidden' ? 'mascot.restore' : open.value ? 'mascot.close' : 'mascot.open'))
const modes = [
  { value: 'character', label: 'mascot.character' },
  { value: 'star', label: 'mascot.star' },
  { value: 'hidden', label: 'mascot.hidden' }
] as const

function compact(value: number | null | undefined) {
  if (value == null || !Number.isFinite(value)) return '-'
  return new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 1 }).format(value)
}

function money(value: number | null | undefined) {
  return value == null || !Number.isFinite(value) ? '-' : `$${value.toFixed(4)}`
}

const metrics = computed(() => [
  { label: t('mascot.todayTokens'), value: compact(stats.value?.today_tokens), color: 'blue' },
  { label: t('mascot.todayRequests'), value: compact(stats.value?.today_requests), color: 'gold' },
  auth.isAdmin
    ? { label: t('mascot.todayCost'), value: money(stats.value?.today_actual_cost), color: 'green' }
    : auth.isSimpleMode
      ? { label: t('mascot.totalTokens'), value: compact(stats.value?.total_tokens), color: 'green' }
      : { label: t('mascot.balance'), value: money(balance.value), color: 'green' },
  { label: t('mascot.apiKeys'), value: stats.value ? `${stats.value.active_api_keys}/${stats.value.total_api_keys}` : '-', color: 'violet' }
])

const summary = computed(() => {
  if (auth.isAdmin) {
    const admin = stats.value && 'total_users' in stats.value ? stats.value : null
    return [
      { label: t('mascot.users'), value: compact(admin?.total_users) },
      { label: t('mascot.accounts'), value: compact(admin?.total_accounts) },
      { label: t('mascot.recent'), value: stats.value ? `${compact(stats.value.rpm)} RPM` : '-' }
    ]
  }
  return [
    { label: t('mascot.totalTokens'), value: compact(stats.value?.total_tokens) },
    { label: t(auth.isSimpleMode ? 'mascot.todayRequests' : 'mascot.todayCost'), value: auth.isSimpleMode ? compact(stats.value?.today_requests) : money(stats.value?.today_actual_cost) },
    { label: t('mascot.recent'), value: stats.value ? `${compact(stats.value.rpm)} RPM` : '-' }
  ]
})

const quotaRows = computed(() => auth.isAdmin || auth.isSimpleMode ? [] : quotas.value.flatMap(quota => (
  ['daily', 'weekly', 'monthly'] as const
).flatMap(window => {
  const limit = quota[`${window}_limit_usd`]
  if (limit == null || limit <= 0) return []
  return [{
    key: `${quota.platform}:${window}`,
    label: `${quota.platform} · ${t(`mascot.${window}`)}`,
    used: quota[`${window}_usage_usd`], limit
  }]
})))

const actions = computed(() => auth.isAdmin ? [
  { label: 'mascot.accounts', path: '/admin/accounts', icon: 'server' as const },
  { label: 'mascot.pricing', path: '/admin/channels/pricing', icon: 'sparkles' as const },
  { label: 'mascot.userManagement', path: '/admin/users', icon: 'users' as const },
  { label: 'mascot.monitoring', path: '/admin/ops', icon: 'chart' as const }
] : [
  { label: 'mascot.keys', path: '/keys', icon: 'key' as const },
  { label: 'mascot.usage', path: '/usage', icon: 'chart' as const },
  { label: 'mascot.models', path: '/model-plaza', icon: 'sparkles' as const },
  { label: 'mascot.security', path: '/profile', icon: 'shield' as const }
])
const updatedTime = computed(() => updatedAt.value?.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' }))

function closePanel(restoreFocus = false) {
  open.value = false
  if (restoreFocus) trigger.value?.focus()
}

function setMode(value: MascotMode) {
  storedMode.value = value
  if (value === 'hidden') closePanel(true)
}

async function onTriggerClick(event: MouseEvent) {
  if (consumeDragClick(event)) return
  if (mode.value === 'hidden') storedMode.value = 'character'
  open.value = !open.value
  if (open.value) {
    void refresh()
    await nextTick()
    closeButton.value?.focus()
  }
}

onClickOutside(root, () => closePanel())
useEventListener(document, 'keydown', (event: KeyboardEvent) => {
  if (event.key === 'Escape' && open.value && !event.defaultPrevented && !document.body.classList.contains('modal-open')) closePanel(true)
})
watch(() => route.fullPath, () => closePanel())
watch(() => auth.user?.id, () => closePanel())
let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  timer = setInterval(() => {
    if (open.value && document.visibilityState === 'visible') void refresh()
  }, 60_000)
})
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>

<style scoped>
.mascot-assistant {
  --mascot-safe-left: env(safe-area-inset-left, 0px);
  --mascot-safe-right: env(safe-area-inset-right, 0px);
  --mascot-safe-top: env(safe-area-inset-top, 0px);
  --mascot-safe-bottom: env(safe-area-inset-bottom, 0px);
  position: fixed;
  right: max(16px, env(safe-area-inset-right));
  bottom: max(16px, env(safe-area-inset-bottom));
  z-index: 35;
  width: 112px;
  height: 162px;
  pointer-events: none;
}
.mascot-compact { width: 44px; height: 44px; }
.mascot-trigger, .mascot-panel { pointer-events: auto; }
.mascot-trigger {
  @apply flex h-11 w-11 items-center justify-center rounded-lg border border-gray-200 bg-white text-primary-600 shadow-md transition dark:border-dark-700 dark:bg-dark-800 dark:text-primary-400;
  cursor: grab;
  touch-action: none;
  user-select: none;
}
.mascot-dragging .mascot-trigger { cursor: grabbing; }
.mascot-dragging .mascot-character:hover { transform: none; }
.mascot-character { width: 112px; height: 162px; }
.mascot-trigger.mascot-character { border: 0; background: transparent; box-shadow: none; }
.mascot-character img { display: block; width: 100%; height: 100%; object-fit: contain; filter: drop-shadow(0 5px 8px rgb(0 0 0 / 10%)); }
.mascot-character:hover { transform: translateY(-3px); }
.mascot-hidden { opacity: 0.55; }
.mascot-hidden:hover { opacity: 1; }
.mascot-panel {
  @apply absolute right-0 overflow-y-auto rounded-lg border border-gray-200 bg-white text-gray-900 shadow-xl dark:border-dark-700 dark:bg-dark-900 dark:text-gray-100;
  bottom: 174px;
  width: min(360px, calc(100vw - 32px));
  max-height: calc(100dvh - 206px);
  overscroll-behavior: contain;
}
.mascot-compact .mascot-panel { bottom: 56px; max-height: calc(100dvh - 88px); }
.mascot-header { @apply flex items-center gap-2 border-b border-gray-100 px-3 py-3 dark:border-dark-700; }
.mascot-avatar { @apply h-10 w-10 shrink-0 rounded-lg bg-primary-50 object-contain dark:bg-dark-800; }
.mascot-heading { @apply min-w-0 flex-1; }
.mascot-brand { @apply text-[10px] font-semibold text-gray-400; }
.mascot-heading h2 { @apply truncate text-sm font-semibold; }
.mascot-header-actions { @apply flex shrink-0; }
.mascot-icon { @apply flex h-7 w-7 items-center justify-center rounded-md text-gray-500 hover:bg-gray-100 disabled:cursor-wait disabled:opacity-50 dark:text-dark-400 dark:hover:bg-dark-700; }
.mascot-content { @apply space-y-3 p-3; }
.mascot-notice { @apply flex items-start gap-2 rounded-md bg-amber-50 p-2 text-xs text-amber-800 dark:bg-amber-950/40 dark:text-amber-200; }
.mascot-notice svg { @apply shrink-0; }
.mascot-loading { @apply flex min-h-40 items-center justify-center gap-2 text-sm text-gray-500; }
.mascot-metrics { @apply grid grid-cols-2 gap-2; }
.mascot-metric { @apply min-w-0 rounded-lg border border-gray-100 bg-gray-50 px-3 py-2.5 dark:border-dark-700 dark:bg-dark-800; }
.mascot-metric span, .mascot-summary span { @apply block text-[11px] text-gray-500 dark:text-dark-400; }
.mascot-metric strong { @apply mt-1 block truncate text-lg font-semibold tabular-nums; }
.mascot-blue strong { @apply text-primary-600 dark:text-primary-400; }
.mascot-gold strong { @apply text-amber-700 dark:text-amber-300; }
.mascot-green strong { @apply text-emerald-700 dark:text-emerald-400; }
.mascot-violet strong { @apply text-violet-600 dark:text-violet-300; }
.mascot-summary { @apply grid grid-cols-3 gap-2 border-y border-dashed border-gray-200 py-2.5 dark:border-dark-700; }
.mascot-summary > div { @apply min-w-0; }
.mascot-summary strong { @apply mt-1 block truncate text-xs font-semibold tabular-nums; }
.mascot-quotas h3 { @apply mb-2 text-xs font-medium text-gray-500 dark:text-dark-400; }
.mascot-quota + .mascot-quota { margin-top: 8px; }
.mascot-quota > div { @apply mb-1 flex flex-wrap items-center justify-between gap-x-2 text-[11px] text-gray-600 dark:text-dark-300; }
.mascot-quota progress { @apply block h-1.5 w-full overflow-hidden rounded-full; appearance: none; }
.mascot-quota progress::-webkit-progress-bar { @apply rounded-full bg-gray-100 dark:bg-dark-700; }
.mascot-quota progress::-webkit-progress-value { @apply rounded-full bg-primary-500; }
.mascot-quota progress::-moz-progress-bar { @apply rounded-full bg-primary-500; }
.mascot-display { @apply flex flex-wrap items-center gap-1 text-[11px]; }
.mascot-display > span { @apply mr-auto text-gray-500 dark:text-dark-400; }
.mascot-display button { @apply rounded-md border border-gray-200 px-2 py-1 text-gray-600 dark:border-dark-600 dark:text-dark-300; }
.mascot-display button[aria-pressed='true'] { @apply border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-950 dark:text-primary-300; }
.mascot-links { @apply grid grid-cols-2 gap-2; }
.mascot-links a { @apply flex min-w-0 items-center gap-1.5 rounded-md border border-gray-200 px-2 py-2 text-xs text-gray-600 hover:border-primary-400 hover:text-primary-600 dark:border-dark-700 dark:text-dark-300 dark:hover:text-primary-400; }
.mascot-links svg { @apply shrink-0; }
.mascot-updated { @apply text-[10px] text-gray-400 dark:text-dark-500; }
.mascot-assistant button:focus-visible, .mascot-links a:focus-visible { @apply outline-none ring-2 ring-primary-500 ring-offset-2 dark:ring-offset-dark-900; }
.mascot-panel-enter-active, .mascot-panel-leave-active { transition: opacity 150ms ease, transform 150ms ease; }
.mascot-panel-enter-from, .mascot-panel-leave-to { opacity: 0; transform: translateY(8px); }
@media (max-width: 640px) {
  .mascot-assistant { right: max(12px, env(safe-area-inset-right)); bottom: max(12px, env(safe-area-inset-bottom)); width: 92px; height: 132px; }
  .mascot-character { width: 92px; height: 132px; }
  .mascot-compact { width: 44px; height: 44px; }
  .mascot-panel { bottom: 144px; width: min(360px, calc(100vw - 24px)); max-height: calc(100dvh - 168px); }
  .mascot-compact .mascot-panel { max-height: calc(100dvh - 80px); }
}
@media (prefers-reduced-motion: reduce) {
  .mascot-trigger, .mascot-panel-enter-active, .mascot-panel-leave-active { transition: none; }
  .mascot-character:hover { transform: none; }
  .animate-spin { animation: none; }
}
</style>
