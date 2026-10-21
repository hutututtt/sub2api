<template>
  <AppLayout>
    <div class="space-y-5 pb-8">
      <div class="flex flex-col gap-4 border-b border-gray-200 pb-4 dark:border-dark-700 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('admin.promotions.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.promotions.description') }}</p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <input v-model="filters.start_at" type="date" class="input w-full sm:w-40" :aria-label="t('admin.promotions.start')" @change="reload" />
          <span class="hidden text-gray-400 sm:inline">-</span>
          <input v-model="filters.end_at" type="date" class="input w-full sm:w-40" :aria-label="t('admin.promotions.end')" @change="reload" />
          <Select v-model="channelFilter" :options="channelOptions" class="w-full sm:w-48" @change="reload" />
          <button class="btn btn-secondary px-2.5" :disabled="loading" :title="t('admin.promotions.refresh')" @click="loadAll">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button v-if="activeTab === 'channels'" class="btn btn-primary" @click="openChannelDialog()">
            <Icon name="plus" size="sm" />{{ t('admin.promotions.addChannel') }}
          </button>
          <button v-if="activeTab === 'expenses'" class="btn btn-primary" :disabled="!channels.length" @click="openExpenseDialog">
            <Icon name="plus" size="sm" />{{ t('admin.promotions.addExpense') }}
          </button>
        </div>
      </div>

      <nav class="flex gap-1 overflow-x-auto border-b border-gray-200 dark:border-dark-700" aria-label="Promotion sections">
        <button v-for="tab in tabs" :key="tab.key" class="whitespace-nowrap border-b-2 px-3 py-2.5 text-sm font-medium transition-colors"
          :class="activeTab === tab.key ? 'border-primary-500 text-primary-600 dark:text-primary-400' : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'"
          @click="selectTab(tab.key)">{{ t(`admin.promotions.${tab.label}`) }}</button>
      </nav>

      <template v-if="activeTab === 'overview'">
        <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
          <div v-for="card in summaryCards" :key="card.label" class="rounded-md border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
            <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400"><Icon :name="card.icon" size="sm" />{{ t(`admin.promotions.${card.label}`) }}</div>
            <div class="mt-2 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatNumber(card.value) }}</div>
            <div v-if="card.detail" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ card.detail }}</div>
          </div>
        </div>

        <section class="border-y border-gray-200 py-4 dark:border-dark-700">
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.promotions.clickTrend') }}</h2>
            <span v-if="report?.trend?.length" class="text-xs text-gray-500 dark:text-gray-400">{{ report.trend.length }} {{ t('admin.promotions.date') }}</span>
          </div>
          <div v-if="loading" class="flex h-64 items-center justify-center text-sm text-gray-400"><Icon name="refresh" class="animate-spin" />{{ t('common.loading') }}</div>
          <div v-else-if="report?.trend?.length" class="h-64"><Line :data="chartData" :options="chartOptions" /></div>
          <div v-else class="flex h-48 items-center justify-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.promotions.noClicks') }}</div>
        </section>

        <section>
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.promotions.channels') }}</h2>
            <button class="text-sm font-medium text-primary-600 hover:underline dark:text-primary-400" @click="selectTab('channels')">{{ t('common.view') }}</button>
          </div>
          <DataTable :columns="channelColumns" :data="report?.channels || []" :loading="loading" row-key="channel_id">
            <template #cell-name="{ row }">
              <div class="min-w-36">
                <div class="font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
                <div class="font-mono text-xs text-gray-500">{{ row.code }}</div>
              </div>
            </template>
            <template #cell-conversion="{ row }">{{ percent(row.converted_visitors, row.visitors) }}</template>
            <template #cell-click_cost="{ row }">{{ formatCosts(row, 'clicks') }}</template>
            <template #cell-registration_cost="{ row }">{{ formatCosts(row, 'registrations') }}</template>
            <template #cell-payer_cost="{ row }">{{ formatCosts(row, 'paying_users') }}</template>
            <template #cell-amounts="{ row }"><div class="space-y-1"><div v-for="amount in row.amounts" :key="amount.currency">{{ amount.currency }} {{ formatMoney(amount.net) }}</div><span v-if="!row.amounts?.length" class="text-gray-400">-</span></div></template>
            <template #empty><div class="py-8 text-center text-sm text-gray-500">{{ t('admin.promotions.empty') }}</div></template>
          </DataTable>
        </section>
      </template>

      <template v-else-if="activeTab === 'channels'">
        <DataTable :columns="channelColumns" :data="report?.channels || []" :loading="loading" row-key="channel_id">
          <template #cell-name="{ row }">
            <div class="min-w-36">
              <div class="flex items-center gap-2 font-medium text-gray-900 dark:text-white"><span>{{ row.name }}</span><span :class="['badge', row.enabled ? 'badge-success' : 'badge-warning']">{{ t(row.enabled ? 'admin.promotions.active' : 'admin.promotions.disabled') }}</span></div>
              <div class="font-mono text-xs text-gray-500">{{ row.code }}</div>
            </div>
          </template>
          <template #cell-link="{ row }">
            <div class="flex min-w-44 items-center gap-1">
              <code class="max-w-56 truncate text-xs text-gray-600 dark:text-gray-300">{{ promotionLink(row.code) }}</code>
              <button class="rounded p-1 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700" :title="t('admin.promotions.copyLink')" @click="copyLink(row.code)"><Icon name="copy" size="xs" /></button>
            </div>
          </template>
          <template #cell-conversion="{ row }">{{ percent(row.converted_visitors, row.visitors) }}</template>
          <template #cell-click_cost="{ row }">{{ formatCosts(row, 'clicks') }}</template>
          <template #cell-registration_cost="{ row }">{{ formatCosts(row, 'registrations') }}</template>
          <template #cell-payer_cost="{ row }">{{ formatCosts(row, 'paying_users') }}</template>
          <template #cell-amounts="{ row }"><div class="space-y-1"><div v-for="amount in row.amounts" :key="amount.currency">{{ amount.currency }} {{ formatMoney(amount.net) }}</div><span v-if="!row.amounts?.length" class="text-gray-400">-</span></div></template>
          <template #cell-actions="{ row }">
            <div class="flex justify-end">
              <button class="btn btn-secondary px-2 py-1.5" :title="t('admin.promotions.editChannel')" @click="editChannel(row)"><Icon name="edit" size="sm" /></button>
            </div>
          </template>
          <template #empty><div class="py-10 text-center text-sm text-gray-500">{{ t('admin.promotions.empty') }}</div></template>
        </DataTable>
      </template>

      <template v-else-if="activeTab === 'settings'">
        <section class="max-w-3xl border-y border-gray-200 py-5 dark:border-dark-700">
          <form class="grid gap-5 sm:grid-cols-2" @submit.prevent="saveSettings">
            <label class="space-y-1.5">
              <span class="input-label">{{ t('admin.promotions.attributionModel') }}</span>
              <Select v-model="settings.attribution_model" :options="attributionOptions" />
            </label>
            <label class="space-y-1.5">
              <span class="input-label">{{ t('admin.promotions.windowDays') }}</span>
              <input v-model.number="settings.window_days" class="input" type="number" min="1" max="90" required />
            </label>
            <div class="sm:col-span-2 flex justify-end">
              <button class="btn btn-primary" type="submit" :disabled="savingSettings">{{ t('admin.promotions.saveSettings') }}</button>
            </div>
          </form>
        </section>
      </template>

      <template v-else>
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="relative w-full sm:max-w-sm">
            <Icon name="search" size="sm" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input v-model="detailSearch" class="input pl-9" :placeholder="t('admin.promotions.search')" @input="scheduleDetails" />
          </div>
          <Select v-if="activeTab === 'users' || activeTab === 'visitors'" v-model="detailStatus" :options="statusOptions" class="w-full sm:w-48" @change="loadDetails" />
        </div>
        <DataTable :columns="detailColumns" :data="details.items" :loading="detailsLoading" row-key="id">
          <template #cell-channel_name="{ value }"><span class="font-medium text-gray-800 dark:text-gray-200">{{ value || '-' }}</span></template>
          <template #cell-email="{ row }"><router-link v-if="row.user_id" :to="{ name: 'AdminUsers', query: { search: row.email } }" class="text-primary-600 hover:underline dark:text-primary-400">{{ row.email || `#${row.user_id}` }}</router-link><span v-else>{{ row.email || '-' }}</span></template>
          <template #cell-amounts="{ row }"><div v-for="amount in (row.amounts || [])" :key="amount.currency">{{ amount.currency }} {{ formatMoney(amount.net) }}</div><span v-if="!row.amounts?.length" class="text-gray-400">-</span></template>
          <template #cell-status="{ row }"><span :class="['badge', row.status === 'paid' ? 'badge-success' : row.status === 'registered' ? 'badge-primary' : 'badge-gray']">{{ t(`admin.promotions.${row.status || 'unregistered'}`) }}</span></template>
          <template #cell-first_paid_at="{ value }">{{ formatDateTime(value) }}</template>
          <template #cell-last_visit_at="{ value }">{{ formatDateTime(value) }}</template>
          <template #cell-paid_amount="{ row }">{{ row.currency }} {{ formatMoney(row.net) }}</template>
          <template #cell-amount="{ row }">{{ row.currency }} {{ formatMoney(row.amount) }}</template>
          <template #cell-created_at="{ value }">{{ formatDateTime(value) }}</template>
          <template #cell-actions="{ row }"><button v-if="activeTab === 'expenses'" class="btn btn-secondary px-2 py-1.5 text-red-600" :title="t('admin.promotions.delete')" @click="removeExpense(row.id)"><Icon name="trash" size="sm" /></button></template>
          <template #empty><div class="py-10 text-center text-sm text-gray-500">{{ emptyLabel }}</div></template>
        </DataTable>
        <Pagination v-if="details.total > 0" :page="details.page" :total="details.total" :page-size="details.page_size" @update:page="changePage" @update:pageSize="changePageSize" />
      </template>
    </div>

    <BaseDialog :show="channelDialog" :title="editingChannel ? t('admin.promotions.editChannel') : t('admin.promotions.addChannel')" width="normal" @close="channelDialog = false">
      <form id="promotion-channel-form" class="space-y-4" @submit.prevent="saveChannel">
        <label class="block"><span class="input-label">{{ t('admin.promotions.name') }}</span><input v-model="channelForm.name" class="input" maxlength="100" required /></label>
        <label class="block"><span class="input-label">{{ t('admin.promotions.code') }}</span><input v-model="channelForm.code" class="input font-mono" maxlength="64" :disabled="!!editingChannel" required /><span class="mt-1 block text-xs text-gray-500">{{ t('admin.promotions.channelCodePlaceholder') }}</span></label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block"><span class="input-label">{{ t('admin.promotions.website') }}</span><input v-model="channelForm.website" class="input" maxlength="255" /></label>
          <label class="block"><span class="input-label">{{ t('admin.promotions.placement') }}</span><input v-model="channelForm.placement" class="input" maxlength="255" /></label>
        </div>
        <label class="block"><span class="input-label">{{ t('admin.promotions.landingPath') }}</span><input v-model="channelForm.landing_path" class="input" maxlength="1000" placeholder="/register" required /><span class="mt-1 block text-xs text-gray-500">{{ t('admin.promotions.landingHint') }}</span></label>
        <label class="block"><span class="input-label">{{ t('admin.promotions.notes') }}</span><textarea v-model="channelForm.notes" class="input" rows="2" maxlength="4000" /></label>
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"><input v-model="channelForm.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600" />{{ t('admin.promotions.enabled') }}</label>
      </form>
      <template #footer><div class="flex justify-end gap-2"><button class="btn btn-secondary" @click="channelDialog = false">{{ t('admin.promotions.cancel') }}</button><button class="btn btn-primary" form="promotion-channel-form" type="submit" :disabled="savingChannel">{{ t('admin.promotions.save') }}</button></div></template>
    </BaseDialog>

    <BaseDialog :show="expenseDialog" :title="t('admin.promotions.addExpense')" width="normal" @close="expenseDialog = false">
      <form id="promotion-expense-form" class="space-y-4" @submit.prevent="saveExpense">
        <label class="block"><span class="input-label">{{ t('admin.promotions.channel') }}</span><Select v-model="expenseForm.channel_id" :options="expenseChannelOptions" /></label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block"><span class="input-label">{{ t('admin.promotions.spentOn') }}</span><input v-model="expenseForm.spent_on" class="input" type="date" required /></label>
          <label class="block"><span class="input-label">{{ t('admin.promotions.currencyCode') }}</span><input v-model="expenseForm.currency" class="input uppercase" maxlength="3" required /></label>
        </div>
        <label class="block"><span class="input-label">{{ t('admin.promotions.amount') }}</span><input v-model="expenseForm.amount" class="input" type="number" step="0.0001" min="0.0001" required /></label>
        <label class="block"><span class="input-label">{{ t('admin.promotions.expenseNotes') }}</span><textarea v-model="expenseForm.notes" class="input" rows="2" maxlength="1000" /></label>
      </form>
      <template #footer><div class="flex justify-end gap-2"><button class="btn btn-secondary" @click="expenseDialog = false">{{ t('admin.promotions.cancel') }}</button><button class="btn btn-primary" form="promotion-expense-form" type="submit" :disabled="savingExpense">{{ t('admin.promotions.addExpense') }}</button></div></template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, CategoryScale, Filler, Legend, LinearScale, LineElement, PointElement, Tooltip } from 'chart.js'
import { Line } from 'vue-chartjs'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import type { Column } from '@/components/common/types'
import { formatDateTime } from '@/utils/format'
import {
  addPromotionExpense,
  deletePromotionExpense,
  getPromotionDetails,
  getPromotionReport,
  getPromotionSettings,
  listPromotionChannels,
  savePromotionChannel as savePromotionChannelRequest,
  savePromotionSettings as savePromotionSettingsRequest,
  type PromotionChannel,
  type PromotionDetails,
  type PromotionMetric,
  type PromotionReport,
  type PromotionSettings,
} from '@/api/admin/promotions'

ChartJS.register(CategoryScale, Filler, Legend, LinearScale, LineElement, PointElement, Tooltip)
type TabKey = 'overview' | 'channels' | 'visitors' | 'users' | 'payments' | 'expenses' | 'settings'
const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const tabs: Array<{ key: TabKey; label: string }> = [
  { key: 'overview', label: 'overview' }, { key: 'channels', label: 'channels' }, { key: 'visitors', label: 'visitors' },
  { key: 'users', label: 'users' }, { key: 'payments', label: 'payments' }, { key: 'expenses', label: 'expenses' }, { key: 'settings', label: 'settings' },
]
const activeTab = ref<TabKey>('overview')
const now = new Date()
const filters = reactive({ start_at: localDate(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 29)), end_at: localDate(now) })
const channelFilter = ref('0')
const channels = ref<PromotionChannel[]>([])
const report = ref<PromotionReport | null>(null)
const settings = reactive<PromotionSettings>({ attribution_model: 'last_touch', window_days: 30 })
const loading = ref(false)
const detailsLoading = ref(false)
const savingSettings = ref(false)
const savingChannel = ref(false)
const savingExpense = ref(false)
const channelDialog = ref(false)
const expenseDialog = ref(false)
const editingChannel = ref<PromotionChannel | null>(null)
const channelForm = reactive({ id: undefined as number | undefined, name: '', code: '', website: '', placement: '', landing_path: '/register', notes: '', enabled: true })
const expenseForm = reactive({ channel_id: 0, spent_on: localDate(now), amount: '', currency: 'CNY', notes: '' })
const detailSearch = ref('')
const detailStatus = ref('')
const detailPage = ref(1)
const detailPageSize = ref(20)
const details = ref<PromotionDetails<Record<string, any>>>({ items: [], total: 0, page: 1, page_size: 20, timezone: 'UTC' })
let detailTimer: ReturnType<typeof setTimeout> | undefined

const channelOptions = computed(() => [{ value: '0', label: t('admin.promotions.allChannels') }, ...channels.value.map((item) => ({ value: String(item.id), label: item.name }))])
const expenseChannelOptions = computed(() => channels.value.map((item) => ({ value: item.id, label: `${item.name} (${item.code})` })))
const attributionOptions = computed(() => [
  { value: 'last_touch', label: t('admin.promotions.lastTouch') }, { value: 'first_touch', label: t('admin.promotions.firstTouch') },
])
const statusOptions = computed(() => [
  { value: '', label: t('admin.promotions.allStatuses') }, { value: 'unregistered', label: t('admin.promotions.unregistered') },
  { value: 'registered', label: t('admin.promotions.registered') }, { value: 'paid', label: t('admin.promotions.paid') },
])
const channelColumns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.promotions.channel') },
  ...(activeTab.value === 'channels' ? [{ key: 'link', label: t('admin.promotions.copyLink') }] : []),
  { key: 'clicks', label: t('admin.promotions.clicks') }, { key: 'effective_clicks', label: t('admin.promotions.effectiveClicks') },
  { key: 'visitors', label: t('admin.promotions.uniqueVisitors') }, { key: 'converted_visitors', label: t('admin.promotions.cohortStats') },
  { key: 'conversion', label: t('admin.promotions.registrationRate') }, { key: 'registrations', label: t('admin.promotions.registrations') },
  { key: 'paying_users', label: t('admin.promotions.paidUsers') },
  { key: 'click_cost', label: t('admin.promotions.clickCost') },
  { key: 'registration_cost', label: t('admin.promotions.registrationCost') },
  { key: 'payer_cost', label: t('admin.promotions.payingCost') },
  { key: 'amounts', label: t('admin.promotions.net') },
  ...(activeTab.value === 'channels' ? [{ key: 'actions', label: '' }] : []),
])
const detailColumns = computed<Column[]>(() => {
  if (activeTab.value === 'users') return [
    { key: 'channel_name', label: t('admin.promotions.channel') }, { key: 'email', label: t('admin.promotions.user') },
    { key: 'created_at', label: t('admin.promotions.registeredAt') }, { key: 'first_channel', label: t('admin.promotions.firstTouch') },
    { key: 'last_channel', label: t('admin.promotions.lastTouch') }, { key: 'first_paid_at', label: t('admin.promotions.firstPaidAt') },
    { key: 'orders', label: t('admin.promotions.orders') }, { key: 'amounts', label: t('admin.promotions.net') }, { key: 'status', label: t('admin.promotions.status') },
  ]
  if (activeTab.value === 'visitors') return [
    { key: 'channel_name', label: t('admin.promotions.channel') }, { key: 'visitor_label', label: t('admin.promotions.visitor') },
    { key: 'referrer_host', label: t('admin.promotions.source') }, { key: 'device', label: t('admin.promotions.device') },
    { key: 'clicks', label: t('admin.promotions.clicks') }, { key: 'effective_clicks', label: t('admin.promotions.effectiveClicks') },
    { key: 'registrations', label: t('admin.promotions.registrations') }, { key: 'last_visit_at', label: t('admin.promotions.lastVisit') },
    { key: 'status', label: t('admin.promotions.status') },
  ]
  if (activeTab.value === 'payments') return [
    { key: 'channel_name', label: t('admin.promotions.channel') }, { key: 'email', label: t('admin.promotions.user') },
    { key: 'out_trade_no', label: t('admin.promotions.orders') }, { key: 'created_at', label: t('admin.promotions.date') },
    { key: 'paid_amount', label: t('admin.promotions.net') }, { key: 'cash_refund', label: t('admin.promotions.refunded') },
    { key: 'bonus_usd', label: t('admin.promotions.bonus') }, { key: 'payment_type', label: t('admin.promotions.paymentMethod') },
  ]
  return [
    { key: 'channel_name', label: t('admin.promotions.channel') }, { key: 'created_at', label: t('admin.promotions.spentOn') },
    { key: 'amount', label: t('admin.promotions.amount') }, { key: 'notes', label: t('admin.promotions.expenseNotes') }, { key: 'actions', label: '' },
  ]
})
const summary = computed(() => report.value?.summary)
const revenueSummary = computed(() => (summary.value?.amounts || []).map((amount) => `${amount.currency} ${formatMoney(amount.net)}`).join(' · '))
const summaryCards = computed(() => [
  { label: 'clicks', icon: 'chartBar' as const, value: summary.value?.clicks || 0, detail: `${t('admin.promotions.effectiveClicks')} ${formatNumber(summary.value?.effective_clicks || 0)}` },
  { label: 'uniqueVisitors', icon: 'users' as const, value: summary.value?.visitors || 0, detail: `${t('admin.promotions.cohortStats')} ${formatNumber(summary.value?.converted_visitors || 0)}` },
  { label: 'registrations', icon: 'userPlus' as const, value: summary.value?.registrations || 0, detail: `${t('admin.promotions.registrationRate')} ${percent(summary.value?.converted_visitors || 0, summary.value?.visitors || 0)}` },
  { label: 'paidUsers', icon: 'creditCard' as const, value: summary.value?.paying_users || 0, detail: revenueSummary.value ? `${t('admin.promotions.net')} ${revenueSummary.value}` : `${formatNumber(summary.value?.orders || 0)} ${t('admin.promotions.paidOrders')}` },
])
const chartData = computed(() => ({
  labels: (report.value?.trend || []).map((item) => item.date.slice(5)),
  datasets: [
    { label: t('admin.promotions.clicks'), data: (report.value?.trend || []).map((item) => item.clicks), borderColor: '#2563eb', backgroundColor: 'rgba(37, 99, 235, 0.10)', fill: true, tension: 0.3 },
    { label: t('admin.promotions.registrations'), data: (report.value?.trend || []).map((item) => item.registrations), borderColor: '#16a34a', backgroundColor: 'rgba(22, 163, 74, 0.08)', fill: false, tension: 0.3 },
  ],
}))
const chartOptions = {
  responsive: true, maintainAspectRatio: false,
  plugins: { legend: { position: 'bottom' as const }, tooltip: { mode: 'index' as const, intersect: false } },
  interaction: { mode: 'index' as const, intersect: false },
  scales: { y: { beginAtZero: true, ticks: { precision: 0 } }, x: { grid: { display: false } } },
}
const emptyLabel = computed(() => t(activeTab.value === 'users' ? 'admin.promotions.noUsers' : activeTab.value === 'payments' ? 'admin.promotions.noPayments' : 'admin.promotions.empty'))

function localDate(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}
function formatNumber(value: number): string { return new Intl.NumberFormat().format(Number(value || 0)) }
function formatMoney(value: number | string): string { return Number(value || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 4 }) }
function percent(numerator: number, denominator: number): string { return denominator > 0 ? `${((numerator / denominator) * 100).toFixed(1)}%` : '0.0%' }
function formatCosts(metric: PromotionMetric, denominatorKey: 'clicks' | 'registrations' | 'paying_users'): string {
  const denominator = Number(metric[denominatorKey] || 0)
  if (denominator <= 0) return '-'
  return (metric.expenses || []).map((expense) => `${expense.currency} ${formatMoney(Number(expense.amount || 0) / denominator)}`).join(' / ') || '-'
}
function promotionLink(code: string): string { return `${window.location.origin}/p/${encodeURIComponent(code)}` }
function filterPayload() {
  return { start_at: filters.start_at, end_at: filters.end_at, channel_id: Number(channelFilter.value) || 0,
    page: detailPage.value, page_size: detailPageSize.value, search: detailSearch.value.trim(), status: detailStatus.value }
}
async function loadAll() {
  loading.value = true
  try {
    const [channelData, reportData, settingData] = await Promise.all([listPromotionChannels(), getPromotionReport(filterPayload()), getPromotionSettings()])
    channels.value = channelData
    report.value = reportData
    Object.assign(settings, settingData)
    if (activeTab.value !== 'overview' && activeTab.value !== 'channels' && activeTab.value !== 'settings') await loadDetails()
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally { loading.value = false }
}
async function loadReport() {
  loading.value = true
  try { report.value = await getPromotionReport(filterPayload()) }
  catch (error: any) { appStore.showError(error?.message || t('common.error')) }
  finally { loading.value = false }
}
async function loadDetails() {
  if (activeTab.value === 'overview' || activeTab.value === 'channels' || activeTab.value === 'settings') return
  detailsLoading.value = true
  try {
    details.value = await getPromotionDetails(activeTab.value as 'users' | 'visitors' | 'payments' | 'expenses', filterPayload())
  } catch (error: any) { appStore.showError(error?.message || t('common.error')) }
  finally { detailsLoading.value = false }
}
async function reload() {
  detailPage.value = 1
  await Promise.all([loadReport(), loadDetails()])
}
async function selectTab(tab: TabKey) {
  activeTab.value = tab
  detailPage.value = 1
  await loadDetails()
}
function scheduleDetails() {
  if (detailTimer) clearTimeout(detailTimer)
  detailTimer = setTimeout(() => { detailPage.value = 1; loadDetails() }, 250)
}
function changePage(page: number) { detailPage.value = page; loadDetails() }
function changePageSize(size: number) { detailPageSize.value = size; detailPage.value = 1; loadDetails() }
async function copyLink(code: string) {
  await copyToClipboard(promotionLink(code), t('admin.promotions.linkCopied'))
}
function resetChannelForm(channel?: PromotionChannel) {
  editingChannel.value = channel || null
  Object.assign(channelForm, channel ? { ...channel } : { id: undefined, name: '', code: '', website: '', placement: '', landing_path: '/register', notes: '', enabled: true })
  channelDialog.value = true
}
function openChannelDialog() { resetChannelForm() }
function editChannel(metric: PromotionMetric) {
  const channel = channels.value.find((item) => item.id === metric.channel_id)
  if (channel) resetChannelForm(channel)
}
async function saveChannel() {
  savingChannel.value = true
  try {
    await savePromotionChannelRequest({ ...channelForm })
    appStore.showSuccess(t('admin.promotions.saveSuccess'))
    channelDialog.value = false
    await loadAll()
  } catch (error: any) { appStore.showError(error?.message || t('common.error')) }
  finally { savingChannel.value = false }
}
async function saveSettings() {
  savingSettings.value = true
  try {
    Object.assign(settings, await savePromotionSettingsRequest({ ...settings }))
    appStore.showSuccess(t('admin.promotions.settingsSaved'))
    await loadReport()
  } catch (error: any) { appStore.showError(error?.message || t('common.error')) }
  finally { savingSettings.value = false }
}
function openExpenseDialog() {
  Object.assign(expenseForm, { channel_id: channels.value[0]?.id || 0, spent_on: localDate(new Date()), amount: '', currency: 'CNY', notes: '' })
  expenseDialog.value = true
}
async function saveExpense() {
  savingExpense.value = true
  try {
    await addPromotionExpense({ ...expenseForm, currency: expenseForm.currency.toUpperCase() })
    appStore.showSuccess(t('admin.promotions.expenseSuccess'))
    expenseDialog.value = false
    await Promise.all([loadReport(), loadDetails()])
  } catch (error: any) { appStore.showError(error?.message || t('common.error')) }
  finally { savingExpense.value = false }
}
async function removeExpense(id: number) {
  if (!window.confirm(t('common.confirm'))) return
  try { await deletePromotionExpense(id); await Promise.all([loadReport(), loadDetails()]) }
  catch (error: any) { appStore.showError(error?.message || t('common.error')) }
}
watch([channelFilter, detailStatus], () => { detailPage.value = 1 })
onMounted(loadAll)
</script>
