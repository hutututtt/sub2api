import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import AccountQualityGroups from '../AccountQualityGroups.vue'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'

const api = vi.hoisted(() => ({
  groups: vi.fn(),
  page: vi.fn(),
  runs: vi.fn(),
  results: vi.fn(),
  save: vi.fn(),
  run: vi.fn(),
  cancel: vi.fn(),
  preview: vi.fn(),
  events: vi.fn(),
  setSchedulable: vi.fn()
}))
vi.mock('@/api/admin/accountQuality', () => ({
  qualityRoutingPreview: api.preview,
  qualityRoutingEvents: api.events,
  qualityCatalog: async () => ({
    reference_models: ['reference-model'],
    bank_revision: 'test'
  }),
  listQualityGroups: api.groups,
  qualityPage: api.page,
  qualityRuns: api.runs,
  qualityResults: api.results,
  saveQualityPolicy: api.save,
  runQuality: api.run,
  cancelQuality: api.cancel
}))
vi.mock('@/api/admin/accounts', () => ({ setSchedulable: api.setSchedulable }))
vi.mock('vue-i18n', async () => {
  const { ref } = await import('vue')
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: ref('en') }) }
})
let wrapper: ReturnType<typeof mount> | undefined
const policy = {
  enabled: false,
  model: '',
  effort: 'low',
  service_tier: '',
  interval_minutes: 15,
  daily_budget: 200,
  max_accounts: 50,
  endpoint: 'responses',
  trace_on_anomaly: false,
  reference_model: '',
  context_check: false,
  context_chars: 32000,
  score_weights: { quality: 40, ttft: 25, cache: 15, stability: 20 },
  mode: 'observe'
}
const account = {
  id: 7,
  name: 'Pinned account',
  platform: 'openai',
  type: 'apikey',
  homepage_url: '',
  status: 'active',
  error_message: null as string | null,
  temp_unschedulable_until: null as string | null,
  temp_unschedulable_reason: null as string | null,
  rate_limit_reset_at: null as string | null,
  overload_until: null as string | null,
  schedulable: true,
  concurrency: 10,
  capacity: { platform: 'openai', type: 'apikey', concurrency: 10, current_concurrency: 3, quota_daily_limit: 20, quota_daily_used: 5 },
  candy_status: 'untested',
  quality_samples: 0,
  quality_passed: 0,
  last_test_at: null as string | null,
  evidence_ids: [] as number[],
  p50_ms: null,
  p95_ms: null,
  cache_rate: null as number | null,
  cache_samples: 0,
  score: null,
  suggested_weight: null,
  failures: 0,
  reliability_known: false,
  success_rate: null,
  trace_status: 'untested',
  trace_candidate: '',
  effective_multiplier: null,
  components: [],
  requests: 0,
  ttft_samples: 0
}
async function render(overrides: Record<string, unknown> = {}, accountOverrides: Partial<typeof account> = {}) {
  const fixturePolicy = { ...policy, ...overrides }
  let accountSnapshot = { ...account, ...accountOverrides }
  api.setSchedulable.mockImplementation(async (_id, schedulable) => { accountSnapshot = { ...accountSnapshot, schedulable }; return { id: 7, schedulable } })
  api.groups.mockResolvedValue([
    {
      id: 1,
      name: 'Group A',
      platform: 'openai',
      total: 9,
      available: 9,
      policy: fixturePolicy
    },
    {
      id: 2,
      name: 'Group B',
      platform: 'openai',
      total: 1,
      available: 1,
      policy
    }
  ])
  api.page.mockImplementation(async (id: number, params: { page: number }) => ({
    items: [{ ...accountSnapshot }],
    page: params.page,
    page_size: 8,
    total: id === 1 ? 9 : 1,
    window_hours: 24,
    cache_window_hours: 24,
    model: '',
    mode: 'observe'
  }))
  api.runs.mockResolvedValue([])
  api.results.mockResolvedValue([])
  api.preview.mockResolvedValue({
    configured_mode: 'observe',
    applied: false,
    reason: 'not_automatic',
    weights: {}
  })
  api.events.mockResolvedValue([])
  api.save.mockImplementation(async (_id, p) => p)
  api.run.mockResolvedValue({
    id: 1,
    status: 'queued',
    planned: 1,
    completed: 0,
    benchmark: 'candy'
  })
  wrapper = mount(AccountQualityGroups, {
    global: { stubs: { Teleport: true } }
  })
  await flushPromises()
  return wrapper
}
afterEach(() => {
  wrapper?.unmount()
  vi.clearAllTimers()
  vi.useRealTimers()
  localStorage.clear()
  vi.restoreAllMocks()
  vi.clearAllMocks()
})
describe('group account observations', () => {
  it('shows temporary unschedulability and recovery time with a details action', async () => {
    const view = await render({}, {
      temp_unschedulable_until: new Date(Date.now() + 600000).toISOString(),
      temp_unschedulable_reason: 'upstream timeout'
    })
    const row = view.find('tbody tr')
    expect(row.text()).toContain('admin.accounts.status.tempUnschedulable')
    expect(row.text()).toContain('admin.accounts.status.tempUnschedulableUntil')
    expect(row.find('.quality-account-identity--disabled').exists()).toBe(true)
    await row.find('button[title="admin.accounts.status.viewTempUnschedDetails"]').trigger('click')
    expect(view.emitted('show-temp-unsched')).toEqual([[7]])
    expect(api.setSchedulable).not.toHaveBeenCalled()
  })
  it('shows error details and treats expired temporary blocks as available', async () => {
    const view = await render({}, { status: 'error', error_message: 'upstream authentication failed' })
    const row = view.find('tbody tr')
    expect(row.text()).toContain('admin.accounts.status.error')
    expect(row.find('.quality-account-identity--error').exists()).toBe(true)
    const indicator = row.findComponent(AccountStatusIndicator)
    expect(indicator.findComponent(HelpTooltip).props('content')).toBe('upstream authentication failed')
    view.unmount()
    const active = await render({}, { temp_unschedulable_until: new Date(Date.now() - 60000).toISOString() })
    expect(active.find('tbody tr').text()).toContain('admin.accounts.status.active')
    expect(active.find('tbody tr .quality-account-identity--disabled').exists()).toBe(false)
  })
  it('refreshes runtime status and clears the badge when the account recovers', async () => {
    vi.useFakeTimers()
    const view = await render()
    const originalPage = api.page.getMockImplementation()!
    let blocked = true
    api.page.mockImplementation(async (...args) => {
      const page = await originalPage(...args)
      return { ...page, items: page.items.map((row: typeof account) => ({
        ...row, temp_unschedulable_until: blocked ? new Date(Date.now() + 600000).toISOString() : null
      })) }
    })
    await vi.advanceTimersByTimeAsync(10000)
    await flushPromises()
    expect(view.find('tbody tr').text()).toContain('admin.accounts.status.tempUnschedulable')
    blocked = false
    await vi.advanceTimersByTimeAsync(10000)
    await flushPromises()
    expect(view.find('tbody tr').text()).not.toContain('admin.accounts.status.tempUnschedulable')
    expect(view.find('tbody tr').text()).toContain('admin.accounts.status.active')
  })
  it('silently refreshes expanded cards every ten seconds while preserving page and selection', async () => {
    vi.useFakeTimers()
    const view = await render()
    await view.setProps({ selectedIds: [7] })
    await view.findAll('button.quality-group-title')[1].trigger('click')
    await view.findAll('article')[0].findAll('button').find(b => b.text() === 'accountQuality.next')!.trigger('click')
    await flushPromises()
    const originalPage = api.page.getMockImplementation()!
    api.page.mockImplementation(async (...args) => {
      const page = await originalPage(...args)
      return { ...page, items: page.items.map((row: typeof account) => ({ ...row, candy_status: 'incorrect', cache_rate: 75, cache_samples: 32, effective_multiplier: 0.1 })) }
    })
    api.groups.mockClear()
    api.page.mockClear()
    await vi.advanceTimersByTimeAsync(10000)
    await flushPromises()
    expect(api.groups).toHaveBeenCalledTimes(1)
    expect(api.page).toHaveBeenCalledTimes(1)
    expect(api.page).toHaveBeenCalledWith(1, expect.objectContaining({ page: 2 }), expect.any(AbortSignal))
    expect(view.text()).toContain('75.0%')
    expect(view.text()).toContain('accountQuality.incorrect')
    expect((view.find('input[aria-label="accountQuality.selectAccount"]').element as HTMLInputElement).checked).toBe(true)
    expect(view.text()).not.toContain('accountQuality.loading')
    expect(api.run).not.toHaveBeenCalled()
  })
  it('allows changing or disabling refresh and persists the preference', async () => {
    vi.useFakeTimers()
    const view = await render()
    const select = view.find('select[aria-label="accountQuality.autoRefresh"]')
    await select.setValue('5')
    api.groups.mockClear()
    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()
    expect(api.groups).toHaveBeenCalledTimes(1)
    expect(JSON.parse(localStorage.getItem('account-quality-auto-refresh')!)).toEqual({ enabled: true, interval_seconds: 5 })
    await select.setValue('0')
    api.groups.mockClear()
    api.page.mockClear()
    await vi.advanceTimersByTimeAsync(20000)
    expect(api.groups).not.toHaveBeenCalled()
    expect(api.page).not.toHaveBeenCalled()
    expect(JSON.parse(localStorage.getItem('account-quality-auto-refresh')!).enabled).toBe(false)
  })
  it('pauses in hidden tabs and refreshes immediately on return', async () => {
    vi.useFakeTimers()
    const view = await render()
    const hidden = vi.spyOn(document, 'hidden', 'get').mockReturnValue(true)
    api.groups.mockClear()
    api.runs.mockClear()
    await vi.advanceTimersByTimeAsync(20000)
    expect(api.groups).not.toHaveBeenCalled()
    expect(api.runs).not.toHaveBeenCalled()
    hidden.mockReturnValue(false)
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()
    expect(api.groups).toHaveBeenCalledTimes(1)
    expect(view.find('select[aria-label="accountQuality.autoRefresh"]').element).toBeTruthy()
  })
  it('pauses for parent and card dialogs without resetting the edited form', async () => {
    vi.useFakeTimers()
    const view = await render()
    await view.setProps({ pauseAutoRefresh: true })
    api.groups.mockClear()
    await vi.advanceTimersByTimeAsync(20000)
    expect(api.groups).not.toHaveBeenCalled()
    await view.setProps({ pauseAutoRefresh: false })
    await view.findAll('button').find(b => b.text() === 'accountQuality.settings')!.trigger('click')
    await vi.advanceTimersByTimeAsync(20000)
    expect(api.groups).not.toHaveBeenCalled()
    expect(view.find('form').exists()).toBe(true)
    await view.findAll('button').find(b => b.text() === 'accountQuality.cancel')!.trigger('click')
    await vi.advanceTimersByTimeAsync(10000)
    await flushPromises()
    expect(api.groups).toHaveBeenCalledTimes(1)
  })
  it('does not overlap refreshes or clear cards while a request is pending', async () => {
    vi.useFakeTimers()
    const view = await render()
    let resolveGroups!: (groups: unknown[]) => void
    api.groups.mockImplementationOnce(() => new Promise(resolve => { resolveGroups = resolve }))
    api.groups.mockClear()
    await vi.advanceTimersByTimeAsync(30000)
    expect(api.groups).toHaveBeenCalledTimes(1)
    expect(view.text()).toContain('Pinned account')
    expect(view.text()).not.toContain('accountQuality.loading')
    view.unmount()
    wrapper = undefined
    api.page.mockClear()
    resolveGroups([])
    await flushPromises()
    await vi.advanceTimersByTimeAsync(30000)
    expect(api.page).not.toHaveBeenCalled()
    expect(api.groups).toHaveBeenCalledTimes(1)
  })
  it('updates cards before an inspection round has finished', async () => {
    vi.useFakeTimers()
    const view = await render()
    api.runs.mockResolvedValue([{ id: 5, status: 'running', planned: 4, completed: 1, benchmark: 'candy' }])
    api.page.mockClear()
    await vi.advanceTimersByTimeAsync(10000)
    await flushPromises()
    expect(api.page).toHaveBeenCalledTimes(2)
    expect(view.text()).toContain('Pinned account')
    expect(api.run).not.toHaveBeenCalled()
  })
  it('labels business request, TTFT and cache counts explicitly', async () => {
    const view = await render({}, { requests: 200, ttft_samples: 150, cache_rate: 82.5, cache_samples: 29 })
    const samples = view.find('[title="accountQuality.cacheSampleHint"]')
    expect(samples.text()).toBe('accountQuality.requestsCount 29')
    expect(samples.element.parentElement!.textContent).toContain('82.5%')
    expect(view.find('[title="accountQuality.sampleHint"]').text()).toContain('accountQuality.requestsCount 200')
    expect(view.find('[title="accountQuality.sampleHint"]').text()).toContain('accountQuality.validTTFTCount 150')
    expect(view.text()).toContain('accountQuality.successCount 200')
    expect(view.text()).toContain('accountQuality.failureCount 0')
  })
  it('filters business models independently and retains the filter during automatic refresh', async () => {
    vi.useFakeTimers()
    const view = await render({ model: 'inspection-model', effort: 'low' })
    expect(api.page).toHaveBeenCalledWith(1, expect.objectContaining({ metrics_model: '' }), expect.any(AbortSignal))
    expect(view.text()).toContain('inspection-model / low')
    api.page.mockClear()
    await view.find('input[aria-label="accountQuality.metricsModel"]').setValue('business-model')
    await vi.advanceTimersByTimeAsync(301)
    await flushPromises()
    expect(api.page).toHaveBeenCalledWith(1, expect.objectContaining({ page: 1, metrics_model: 'business-model' }), expect.any(AbortSignal))
    api.page.mockClear()
    await vi.advanceTimersByTimeAsync(10000)
    await flushPromises()
    expect(api.page).toHaveBeenCalledWith(1, expect.objectContaining({ metrics_model: 'business-model' }), expect.any(AbortSignal))
    expect(api.save).not.toHaveBeenCalled()
    expect(api.run).not.toHaveBeenCalled()
  })
  it('requires explicit current consent before enabling scheduled disclosure', async () => {
    const view = await render({ enabled: true, model: 'fixture-model' })
    await view.findAll('button').find(b => b.text().includes('accountQuality.settings'))!.trigger('click')
    await view.find('form').trigger('submit')
    await flushPromises()
    expect(api.save).not.toHaveBeenCalled()
    expect(view.text()).toContain('accountQuality.consentRequired')
    const consent = view.findAll('label').find(label => label.text().includes('accountQuality.scheduledConsent'))!
    await consent.find('input').setValue(true)
    await view.find('form').trigger('submit')
    await flushPromises()
    expect(api.save).toHaveBeenCalledWith(1, expect.objectContaining({ enabled: true, candy_consent_version: 'manxue-v1' }))
  })
  it('shows the timestamp of expired evidence even with no current samples', async () => {
    const view = await render({}, { candy_status: 'stale', last_test_at: '2026-10-19T03:04:00Z' })
    const cell = view.find('button.quality-verdict').element.parentElement!
    expect(cell.textContent).toContain('accountQuality.stale')
    expect(cell.textContent).toContain('10/19')
    expect(view.find('button.quality-verdict').classes()).not.toContain('text-emerald-700')
  })
  it('opens the exact cross-group evidence shown in the list', async () => {
    const view = await render({}, { candy_status: 'passed', evidence_ids: [41, 42] })
    api.results.mockResolvedValue([{ id: 41, account_id: 7, group_id: 2, model: 'fixture-model', effort: 'low', benchmark: 'candy-manxue-v1', verdict: 'passed', created_at: '2026-10-19T03:04:00Z', duration_ms: 100 }])
    await view.find('button.quality-verdict').trigger('click')
    await flushPromises()
    expect(api.results).toHaveBeenCalledWith(1, 7, [41, 42])
    expect(view.text()).toContain('accountQuality.evidenceSource')
  })
  it('links account names to the upstream homepage with the original tooltip and styling', async () => {
    const view = await render({}, { homepage_url: 'https://relay.example.com/api/v1/' })
    const links = view.findAll('tbody a')
    expect(links).toHaveLength(2)
    for (const link of links) {
      expect(link.text()).toBe(account.name)
      expect(link.attributes()).toMatchObject({
        href: 'https://relay.example.com',
        target: '_blank',
        rel: 'noopener noreferrer'
      })
      expect(link.classes()).toEqual(expect.arrayContaining(['quality-account-link']))
    }
    const tooltip = view.findAllComponents(HelpTooltip).find(item => item.props('content') === 'https://relay.example.com')!
    expect(tooltip.props('content')).toBe('https://relay.example.com')
    expect(tooltip.props('widthClass')).toBe('w-max max-w-sm break-all')
    await links[0].trigger('click')
    expect(view.emitted('selection')).toBeUndefined()
    expect(view.emitted('edit')).toBeUndefined()
    expect(api.run).not.toHaveBeenCalled()
  })
  it.each([
    { type: 'oauth', homepage_url: 'https://oauth.example.com' },
    { type: 'apikey', homepage_url: '' },
    { type: 'apikey', homepage_url: 'javascript:alert(1)' }
  ])('keeps account names as text for $type with URL "$homepage_url"', async (overrides) => {
    const view = await render({}, overrides)
    expect(view.findAll('tbody a')).toHaveLength(0)
    expect(view.text()).toContain(account.name)
  })
  it('renders the original capacity cell with runtime concurrency and configured quotas', async () => {
    const view = await render()
    const cells = view.findAllComponents(AccountCapacityCell)
    expect(cells).toHaveLength(2)
    expect(cells[0].props('account')).toEqual(account.capacity)
    expect(cells[0].text()).toContain('3/10')
    expect(cells[0].text()).toContain('$5.00/$20.00')
  })
  it('saves one scheduling change and synchronizes duplicate account cards', async () => {
    const view = await render()
    await view.findAll('[role="switch"]')[0].trigger('click')
    await flushPromises()
    expect(api.setSchedulable).toHaveBeenCalledTimes(1)
    expect(api.setSchedulable).toHaveBeenCalledWith(7, false)
    expect(view.findAll('[role="switch"]').map(s => s.attributes('aria-checked'))).toEqual(['false', 'false'])
    expect(view.emitted('schedulingChange')?.at(-1)).toEqual([7, false])
    await view.findAll('[role="switch"]')[1].trigger('click')
    await flushPromises()
    expect(api.setSchedulable).toHaveBeenLastCalledWith(7, true)
    expect(view.findAll('[role="switch"]').map(s => s.attributes('aria-checked'))).toEqual(['true', 'true'])
    expect(api.run).not.toHaveBeenCalled()
  })
  it('locks duplicate switches while saving and preserves state on failure', async () => {
    const view = await render()
    let rejectSave!: (error: Error) => void
    api.setSchedulable.mockReturnValueOnce(new Promise((_resolve, reject) => { rejectSave = reject }))
    await view.findAll('[role="switch"]')[0].trigger('click')
    expect(view.findAll('[role="switch"]').every(s => s.attributes('disabled') !== undefined)).toBe(true)
    rejectSave(new Error('upstream database unavailable'))
    await flushPromises()
    expect(view.findAll('[role="switch"]').map(s => s.attributes('aria-checked'))).toEqual(['true', 'true'])
    expect(view.findAll('[role="switch"]').every(s => s.attributes('disabled') === undefined)).toBe(true)
    expect(view.text()).toContain('admin.accounts.failedToToggleSchedulable')
    expect(view.emitted('schedulingChange')).toBeUndefined()
  })
  it('loads each group independently and does not fabricate missing quality', async () => {
    const view = await render()
    expect(view.findAll('article')).toHaveLength(2)
    expect(api.page).toHaveBeenCalledWith(
      1,
      expect.objectContaining({ page: 1, page_size: 8 }),
      expect.any(AbortSignal)
    )
    expect(api.page).toHaveBeenCalledWith(
      2,
      expect.objectContaining({ page: 1 }),
      expect.any(AbortSignal)
    )
    expect(view.text()).toContain('accountQuality.noData')
    expect(view.text()).toContain('accountQuality.untested')
    expect(api.run).not.toHaveBeenCalled()
  })
  it('paginates only the selected group', async () => {
    const view = await render()
    api.page.mockClear()
    const next = view
      .findAll('article')[0]
      .findAll('button')
      .find((b) => b.text() === 'accountQuality.next')
    await next!.trigger('click')
    await flushPromises()
    expect(api.page).toHaveBeenCalledTimes(1)
    expect(api.page).toHaveBeenCalledWith(
      1,
      expect.objectContaining({ page: 2 }),
      expect.any(AbortSignal)
    )
  })
  it('requires model configuration before starting any paid inspection', async () => {
    const view = await render()
    const inspect = view
      .findAll('button')
      .find((b) => b.text() === 'accountQuality.inspect')
    await inspect!.trigger('click')
    await flushPromises()
    expect(view.text()).toContain('accountQuality.configure')
    expect(api.run).not.toHaveBeenCalled()
    expect(api.save).not.toHaveBeenCalled()
  })
  it('shows an empty evidence state without inventing test results', async () => {
    const view = await render()
    const evidence = view
      .findAll('tbody button')
      .find((b) => b.text() === 'accountQuality.history')
    await evidence!.trigger('click')
    await flushPromises()
    expect(api.results).toHaveBeenCalledWith(1, 7, [])
    expect(view.text()).toContain('accountQuality.noHistory')
  })
  it('deduplicates the same account across multiple group cards', async () => {
    const view = await render()
    const rows = view.findAll(
      'input[aria-label="accountQuality.selectAccount"]'
    )
    expect(rows).toHaveLength(2)
    await rows[0].setValue(true)
    const first = view.emitted('selection')!.at(-1)![0] as { id: number }[]
    expect(first.map((r) => r.id)).toEqual([7])
    await view.setProps({ selectedIds: [7] })
    expect((rows[1].element as HTMLInputElement).checked).toBe(true)
    await rows[1].setValue(false)
    expect(view.emitted('selection')!.at(-1)![0]).toEqual([])
  })
  it('sends an explicitly selected local fingerprint benchmark after confirmation', async () => {
    const view = await render({
      model: 'fixture-model',
      reference_model: 'reference-model'
    })
    await view
      .find('select[aria-label="accountQuality.benchmark"]')
      .setValue('modeltrace')
    await view
      .findAll('button')
      .find((b) => b.text() === 'accountQuality.inspect')!
      .trigger('click')
    expect(api.run).not.toHaveBeenCalled()
    await view
      .findAll('button')
      .find((b) => b.text() === 'accountQuality.start')!
      .trigger('click')
    await flushPromises()
    expect(api.run).toHaveBeenCalledWith(1, 0, 'modeltrace')
  })
  it('discloses credential submission before starting a Manxue candy task', async () => {
    const view = await render({ model: 'fixture-model' })
    await view.findAll('button').find(b => b.text() === 'accountQuality.inspect')!.trigger('click')
    expect(view.text()).toContain('accountQuality.manxueNotice')
    expect(api.run).not.toHaveBeenCalled()
    await view.findAll('button').find(b => b.text() === 'accountQuality.start')!.trigger('click')
    await flushPromises()
    expect(api.run).toHaveBeenCalledWith(1, 0, 'candy')
  })
  it('excludes OAuth from direct candy checks and keeps local fingerprint checks available', async () => {
    const view = await render({ model: 'fixture-model', reference_model: 'reference-model' }, { type: 'oauth' })
    const candy = view.findAll('tbody button').find(b => b.text() === 'accountQuality.candy')!
    expect(candy.attributes('disabled')).toBeDefined()
    expect(candy.attributes('title')).toBe('accountQuality.oauthUnsupported')
    expect(view.text()).toContain('accountQuality.oauthUnsupported')
    await view.find('select[aria-label="accountQuality.benchmark"]').setValue('modeltrace')
    const trace = view.findAll('tbody button').find(b => b.text() === 'accountQuality.benchmarkTrace')!
    expect(trace.attributes('disabled')).toBeUndefined()
    expect(api.run).not.toHaveBeenCalled()
  })
  it('uses saved account credentials without separate URL or key settings', async () => {
    const view = await render({ model: 'fixture-model' })
    await view.findAll('button').find(b => b.text() === 'accountQuality.settings')!.trigger('click')
    expect(view.text()).toContain('accountQuality.accountCredentials')
    expect(view.find('input[type="url"]').exists()).toBe(false)
    expect(view.find('input[aria-label="accountQuality.relayURL"]').exists()).toBe(false)
    await view.findAll('button').find(b => b.text() === 'accountQuality.cancel')!.trigger('click')
    const candy = view.findAll('tbody button').find(b => b.text() === 'accountQuality.candy')!
    expect(candy.attributes('disabled')).toBeUndefined()
    await candy.trigger('click')
    expect(api.run).not.toHaveBeenCalled()
    await view.findAll('button').find(b => b.text() === 'accountQuality.start')!.trigger('click')
    await flushPromises()
    expect(api.run).toHaveBeenCalledWith(1, 7, 'candy')
  })
  it('rejects incompatible candy settings before creating a task', async () => {
    const view = await render({ model: 'fixture-model', effort: 'none', service_tier: 'flex' })
    await view.findAll('button').find(b => b.text() === 'accountQuality.inspect')!.trigger('click')
    expect(view.text()).toContain('accountQuality.manxueParameters')
    expect(api.run).not.toHaveBeenCalled()
    expect(view.find('form').exists()).toBe(true)
  })
  it('labels remote evidence and translates expired remote tasks', async () => {
    const view = await render({ model: 'fixture-model' })
    api.results.mockResolvedValue([{ id: 5, account_id: 7, model: 'fixture-model', effort: 'low', verdict: 'error', benchmark: 'candy-manxue-v1', evidence: { provider: 'manxue' }, detail: 'manxue_http_404', created_at: new Date().toISOString(), duration_ms: 1000 }])
    await view.findAll('tbody button').find(b => b.text() === 'accountQuality.history')!.trigger('click')
    await flushPromises()
    expect(view.text()).toContain('accountQuality.manxueResult')
    expect(view.text()).toContain('accountQuality.manxueHTTP')
  })
  it('keeps edited routing modes local until settings are saved', async () => {
    const view = await render({ model: 'fixture-model' })
    await view
      .findAll('button')
      .find((b) => b.text() === 'accountQuality.settings')!
      .trigger('click')
    const select = view
      .findAll('select')
      .find((s) => s.find('option[value="auto"]').exists())!
    await select.setValue('auto')
    expect(api.save).not.toHaveBeenCalled()
    await view.find('form').trigger('submit')
    await flushPromises()
    expect(api.save).toHaveBeenCalledWith(
      1,
      expect.objectContaining({
        mode: 'auto',
        enabled: false,
        model: 'fixture-model',
        interval_minutes: 15
      })
    )
  })
  it('shows runtime diagnostics without enabling weighting', async () => {
    const view = await render()
    await view
      .findAll('button')
      .find((b) => b.text() === 'accountQuality.routingDiagnostics')!
      .trigger('click')
    await flushPromises()
    expect(api.preview).toHaveBeenCalled()
    expect(api.events).toHaveBeenCalledWith(1)
    expect(view.text()).toContain('accountQuality.routeReason.not_automatic')
    expect(api.save).not.toHaveBeenCalled()
    expect(api.run).not.toHaveBeenCalled()
  })
})
