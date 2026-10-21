import { apiClient } from '../client'

export interface PromotionSettings {
  attribution_model: 'first_touch' | 'last_touch'
  window_days: number
}

export interface PromotionChannel {
  id: number
  code: string
  name: string
  website: string
  placement: string
  landing_path: string
  notes: string
  enabled: boolean
  created_at: string
}

export interface PromotionFilter {
  start_at: string
  end_at: string
  channel_id?: number
  page?: number
  page_size?: number
  search?: string
  status?: string
}

export interface PromotionAmount {
  currency: string
  paid: number
  refunded: number
  net: number
  bonus_usd?: number
  orders?: number
  payers?: number
}

export interface PromotionMetric {
  channel_id: number
  name: string
  code: string
  enabled: boolean
  clicks: number
  effective_clicks: number
  visitors: number
  converted_visitors: number
  converted_payers: number
  registrations: number
  paying_users: number
  orders: number
  cohort_payers: number
  amounts: PromotionAmount[]
  cohort_amounts: PromotionAmount[]
  expenses: Array<{ currency: string; amount: number }>
}

export interface PromotionReport {
  summary: PromotionMetric
  channels: PromotionMetric[]
  trend: Array<{ date: string; clicks: number; visitors: number; registrations: number }>
}

export interface PromotionDetails<T = Record<string, unknown>> {
  items: T[]
  total: number
  page: number
  page_size: number
  timezone: string
}

export interface PromotionExpenseInput {
  channel_id: number
  spent_on: string
  amount: string
  currency: string
  notes: string
}

function params(filter: PromotionFilter) {
  return {
    ...filter,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
  }
}

export async function listPromotionChannels(): Promise<PromotionChannel[]> {
  const { data } = await apiClient.get<PromotionChannel[]>('/admin/promotions/channels')
  return data || []
}

export async function savePromotionChannel(channel: Partial<PromotionChannel>): Promise<PromotionChannel> {
  const { data } = channel.id
    ? await apiClient.put<PromotionChannel>(`/admin/promotions/channels/${channel.id}`, channel)
    : await apiClient.post<PromotionChannel>('/admin/promotions/channels', channel)
  return data
}

export async function getPromotionSettings(): Promise<PromotionSettings> {
  const { data } = await apiClient.get<PromotionSettings>('/admin/promotions/settings')
  return data
}

export async function savePromotionSettings(settings: PromotionSettings): Promise<PromotionSettings> {
  const { data } = await apiClient.put<PromotionSettings>('/admin/promotions/settings', settings)
  return data
}

export async function getPromotionReport(filter: PromotionFilter): Promise<PromotionReport> {
  const { data } = await apiClient.get<PromotionReport>('/admin/promotions/report', { params: params(filter) })
  return data
}

export async function getPromotionDetails<T = Record<string, unknown>>(
  kind: 'users' | 'visitors' | 'payments' | 'expenses',
  filter: PromotionFilter,
): Promise<PromotionDetails<T>> {
  const { data } = await apiClient.get<PromotionDetails<T>>(`/admin/promotions/details/${kind}`, {
    params: params(filter),
  })
  return data
}

export async function addPromotionExpense(expense: PromotionExpenseInput): Promise<void> {
  await apiClient.post('/admin/promotions/expenses', expense)
}

export async function deletePromotionExpense(id: number): Promise<void> {
  await apiClient.delete(`/admin/promotions/expenses/${id}`)
}

export default {
  listPromotionChannels,
  savePromotionChannel,
  getPromotionSettings,
  savePromotionSettings,
  getPromotionReport,
  getPromotionDetails,
  addPromotionExpense,
  deletePromotionExpense,
}
