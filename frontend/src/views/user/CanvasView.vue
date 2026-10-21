<template>
  <AppLayout immersive compact-sidebar>
    <template #header-left>
      <div class="flex min-w-0 items-center gap-2">
        <label v-if="keys.length" class="flex min-w-0 flex-1 items-center gap-2 text-sm">
          <span class="sr-only">{{ t('canvas.channel') }}</span>
          <select :value="selectedId" class="h-8 w-32 min-w-0 rounded-md border border-gray-200 bg-gray-50 pl-2 pr-6 text-xs text-gray-700 outline-none transition focus:border-primary-500 focus:ring-2 focus:ring-primary-500/15 disabled:opacity-50 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200 sm:w-40 xl:w-56" :disabled="busy || loading" @change="selectKey">
            <option v-for="key in keys" :key="key.id" :value="key.id">{{ key.name }} · {{ key.group?.name }}</option>
          </select>
        </label>
        <span v-if="busy" role="status" class="hidden text-xs text-emerald-600 sm:inline">{{ t('canvas.generating') }}</span>
        <button type="button" class="ml-auto grid h-8 w-8 shrink-0 place-items-center rounded-md text-gray-500 transition hover:bg-gray-100 hover:text-primary-600 disabled:opacity-40 dark:text-gray-400 dark:hover:bg-dark-800" :title="t('canvas.refresh')" :aria-label="t('canvas.refresh')" :disabled="busy || loading" @click="reload">
          <Icon name="refresh" size="sm" />
        </button>
      </div>
    </template>
    <section class="flex h-full min-h-0 flex-col bg-white dark:bg-dark-950">
      <div class="relative min-h-0 flex-1">
        <iframe v-if="session && !error && auth.user?.id === session.userId" :key="frameVersion" ref="frame" src="/canvas-app/index.html#/canvas?mode=recent" class="h-full w-full border-0" :title="t('canvas.title')" allow="clipboard-write; fullscreen" @load="initializeFrame" @error="error = t('canvas.frameError')" />
        <div v-if="loading || (session && !ready && !error)" role="status" class="absolute inset-0 flex items-center justify-center bg-white dark:bg-dark-950">{{ t('canvas.loading') }}</div>
        <div v-else-if="error || !session" class="flex h-full flex-col items-center justify-center gap-4 px-6 text-center">
          <p role="status" class="max-w-lg text-sm text-gray-600 dark:text-gray-300">{{ error || t('canvas.noKeys') }}</p>
          <RouterLink v-if="!keys.length && !error" to="/keys" class="btn btn-primary">{{ t('canvas.manageKeys') }}</RouterLink>
          <button v-else type="button" class="btn btn-secondary" @click="reload">{{ t('canvas.retry') }}</button>
        </div>
      </div>
    </section>
  </AppLayout>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { onBeforeRouteLeave } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores/auth'
import { loadCanvasKeys, loadCanvasModels } from '@/api/canvas'
import { buildGatewayUrl } from '@/api/url'
import type { ApiKey } from '@/types'
import type { CanvasSession } from '../../../../integrations/infinite-canvas/context'

const auth = useAuthStore()
const { t, locale } = useI18n()
const keys = ref<ApiKey[]>([])
const selectedId = ref<number>()
const session = ref<CanvasSession>()
const frame = ref<HTMLIFrameElement>()
const loading = ref(false)
const ready = ref(false)
const busy = ref(false)
const error = ref('')
const frameVersion = ref(0)
let request: AbortController | undefined

function initializeFrame() {
  if (session.value && auth.user?.id === session.value.userId) {
    frame.value?.contentWindow?.postMessage({ type: 'sub2api:canvas:init', session: { ...session.value, models: session.value.models.map(model => ({ ...model })) } }, window.location.origin)
  }
}

function receive(event: MessageEvent) {
  if (event.origin !== window.location.origin || event.source !== frame.value?.contentWindow || !session.value || auth.user?.id !== session.value.userId) return
  switch (event.data?.type) {
    case 'sub2api:canvas:boot': initializeFrame(); break
    case 'sub2api:canvas:ready': ready.value = true; break
    case 'sub2api:canvas:error': error.value = t('canvas.frameError'); break
    case 'sub2api:canvas:busy': busy.value = event.data.busy === true; break
    case 'sub2api:canvas:settled': void auth.refreshUser().catch(() => {}); break
  }
}
window.addEventListener('message', receive)
const themeObserver = new MutationObserver(() => {
  if (!session.value) return
  session.value.theme = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
  frame.value?.contentWindow?.postMessage({ type: 'sub2api:canvas:theme', theme: session.value.theme }, window.location.origin)
})
themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

async function reload() {
  request?.abort()
  const controller = new AbortController()
  request = controller
  const userId = auth.user?.id
  session.value = undefined
  ready.value = false
  busy.value = false
  error.value = ''
  keys.value = []
  loading.value = false
  if (!userId) return
  loading.value = true
  try {
    const available = await loadCanvasKeys(userId, controller.signal)
    if (controller.signal.aborted || auth.user?.id !== userId) return
    keys.value = available
    // Prefer the prior channel, then find the first channel with image models.
    const ordered = [...available].sort((a, b) => Number(b.id === selectedId.value) - Number(a.id === selectedId.value))
    let catalogFailed = false
    for (const key of ordered) {
      try {
        const models = await loadCanvasModels(key, controller.signal)
        if (controller.signal.aborted || auth.user?.id !== userId) return
        if (!models.some(model => model.capability === 'image')) continue
        start(key, models, userId)
        return
      } catch {
        if (controller.signal.aborted) return
        catalogFailed = true
      }
    }
    if (available.length) error.value = t(catalogFailed ? 'canvas.catalogError' : 'canvas.noModels')
  } catch {
    if (!controller.signal.aborted) error.value = t('canvas.keysError')
  } finally {
    if (request === controller) loading.value = false
  }
}

function start(key: ApiKey, models: CanvasSession['models'], userId: number) {
  selectedId.value = key.id
  session.value = {
    userId, keyId: key.id, apiKey: key.key, baseUrl: buildGatewayUrl('/'),
    apiFormat: key.group?.platform === 'gemini' ? 'gemini' : 'openai', models,
    locale: locale.value, theme: document.documentElement.classList.contains('dark') ? 'dark' : 'light',
    storageName: `sub2api-canvas-${userId}`
  }
  frameVersion.value++
}

async function selectKey(event: Event) {
  if (busy.value) return
  const key = keys.value.find(key => key.id === Number((event.target as HTMLSelectElement).value))
  if (!key || !auth.user) return
  request?.abort()
  const controller = new AbortController()
  request = controller
  const userId = auth.user.id
  selectedId.value = key.id
  session.value = undefined
  ready.value = false
  loading.value = true
  error.value = ''
  try {
    const models = await loadCanvasModels(key, controller.signal)
    if (controller.signal.aborted || auth.user?.id !== userId) return
    if (!models.some(model => model.capability === 'image')) error.value = t('canvas.noModels')
    else start(key, models, userId)
  } catch {
    if (!controller.signal.aborted) error.value = t('canvas.catalogError')
  } finally {
    if (request === controller) loading.value = false
  }
}

watch(() => auth.user?.id, () => void reload(), { immediate: true })
onBeforeRouteLeave(() => !busy.value || window.confirm(t('canvas.leaveConfirm')))
const preventUnload = (event: BeforeUnloadEvent) => { if (busy.value) { event.preventDefault(); event.returnValue = '' } }
window.addEventListener('beforeunload', preventUnload)
onBeforeUnmount(() => {
  themeObserver.disconnect()
  request?.abort()
  window.removeEventListener('message', receive)
  window.removeEventListener('beforeunload', preventUnload)
})
</script>
