import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import AccountQualityGroups from '../AccountQualityGroups.vue'

const api = vi.hoisted(() => ({ groups: vi.fn(), page: vi.fn(), runs: vi.fn() }))
vi.mock('@/api/admin/accountQuality', () => ({
  qualityCatalog: async () => ({ reference_models: [], bank_revision: 'review' }),
  listQualityGroups: api.groups, qualityPage: api.page, qualityRuns: api.runs,
  qualityResults: vi.fn(), qualityRoutingPreview: vi.fn(), qualityRoutingEvents: vi.fn(),
  saveQualityPolicy: vi.fn(), runQuality: vi.fn(), cancelQuality: vi.fn()
}))
vi.mock('@/api/admin/accounts', () => ({ setSchedulable: vi.fn() }))
vi.mock('vue-i18n', async () => {
  const { ref } = await import('vue')
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: ref('en') }) }
})
const p={enabled:false,model:'review',effort:'low',service_tier:'',interval_minutes:15,daily_budget:200,max_accounts:50,mode:'observe',endpoint:'responses',trace_on_anomaly:false,reference_model:'',context_check:false,context_chars:32000,score_weights:{quality:40,ttft:25,cache:15,stability:20}}
const groups=[{id:1,name:'Review A',platform:'openai',total:1,available:1,policy:p}]
let view: ReturnType<typeof mount> | undefined
afterEach(()=>{view?.unmount(); view=undefined;vi.clearAllTimers();vi.useRealTimers();vi.clearAllMocks()})
describe('account quality regressions',()=>{
 it('refreshes a previously loaded collapsed group for changed status filters',async()=>{
  vi.useFakeTimers()
  api.groups.mockResolvedValue(groups)
  api.page.mockImplementation(async (_id,params)=>({items:[],total:0,page:params.page,page_size:8,window_hours:24,model:'review',mode:'observe'}))
  api.runs.mockResolvedValue([])
  view=mount(AccountQualityGroups,{global:{stubs:{Teleport:true}}})
  await flushPromises()
  await view.find('button.quality-group-title').trigger('click')
  api.page.mockClear()
  await view.find('select[aria-label="accountQuality.statusFilter"]').setValue('unavailable')
  await vi.advanceTimersByTimeAsync(301)
  await view.find('button.quality-group-title').trigger('click')
  await flushPromises()
  expect(api.page).toHaveBeenCalledWith(1,expect.objectContaining({status:'unavailable'}),expect.any(AbortSignal))
 })
 it('does not start polling when mounting requests resolve after unmount',async()=>{
  vi.useFakeTimers()
  let resolveGroups!:(v:typeof groups)=>void
  api.groups.mockImplementation(()=>new Promise(resolve=>{resolveGroups=resolve}))
  api.runs.mockResolvedValue([])
  view=mount(AccountQualityGroups,{global:{stubs:{Teleport:true}}})
  await flushPromises()
  view.unmount();view=undefined
  api.runs.mockClear()
  resolveGroups(groups)
  await flushPromises()
  await vi.advanceTimersByTimeAsync(5001)
  expect(api.runs).not.toHaveBeenCalled()
 })
})
