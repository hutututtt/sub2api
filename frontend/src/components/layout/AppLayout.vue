<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <!-- Background Decoration -->
    <div class="pointer-events-none fixed inset-0 bg-mesh-gradient"></div>

    <!-- Sidebar -->
    <AppSidebar :compact="compactSidebar" @collapse-change="workspaceCollapsed = $event" />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen transition-all duration-300"
      :class="[(compactSidebar ? workspaceCollapsed : sidebarCollapsed) ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader><template v-if="$slots['header-left']" #default><slot name="header-left" /></template></AppHeader>

      <!-- Main Content -->
      <main :class="immersive ? 'h-[calc(100dvh-65px)] min-h-0 overflow-hidden' : 'p-4 md:p-6 lg:p-8'">
        <slot />
      </main>
    </div>
    <MascotAssistant v-if="!immersive" />
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted, ref } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import MascotAssistant from './MascotAssistant.vue'

defineProps<{ immersive?: boolean; compactSidebar?: boolean }>()
const workspaceCollapsed = ref(true)

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
