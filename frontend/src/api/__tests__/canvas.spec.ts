import { describe, expect, it, vi, afterEach } from 'vitest'
import { canvasModels, isCanvasKey, loadCanvasKeys, loadCanvasModels } from '../canvas'
import { keysAPI } from '../keys'
import type { ApiKey } from '@/types'

vi.mock('../keys', () => ({ keysAPI: { list: vi.fn() } }))
const key = { id: 2, user_id: 7, key: 'test-user-key', status: 'active', quota: 0, group: { platform: 'openai', allow_image_generation: true } } as ApiKey
afterEach(() => vi.unstubAllGlobals())

describe('canvas channels', () => {
  it('only accepts usable image keys owned by the signed in user', () => {
    expect(isCanvasKey(key, 7)).toBe(true)
    for (const changed of [{ user_id: 8 }, { status: 'expired' }, { quota: 1, quota_used: 1 }, { expires_at: '2020-01-01' }, { group: { platform: 'openai', allow_image_generation: false } }]) {
      expect(isCanvasKey({ ...key, ...changed } as ApiKey, 7)).toBe(false)
    }
  })
  it('paginates and filters keys without creating keys', async () => {
    vi.mocked(keysAPI.list).mockResolvedValueOnce({ items: [{ ...key, user_id: 8 }], pages: 2 } as never).mockResolvedValueOnce({ items: [key], pages: 2 } as never)
    expect(await loadCanvasKeys(7, new AbortController().signal)).toEqual([key])
    expect(keysAPI.list).toHaveBeenLastCalledWith(2, 100, { status: 'active' }, expect.anything())
  })
  it('only exposes supported protocols and separates media from text', () => {
    expect(canvasModels(['gpt-image-1', 'gpt-image-1', 'gpt-5', 'gpt-4o-mini-tts', 'flux', 'gemini-3-pro-image', 'grok-imagine-image', 'grok-imagine-video'], false)).toEqual([
      { name: 'gpt-image-1', capability: 'image' }, { name: 'gpt-5', capability: 'text' }, { name: 'grok-imagine-image', capability: 'image' }
    ])
    expect(canvasModels(['gemini-3-pro-image', 'gemini-2.5-flash', 'gemini-2.5-flash-preview-tts', 'imagen-4'], true)).toEqual([
      { name: 'gemini-3-pro-image', capability: 'image' }, { name: 'gemini-2.5-flash', capability: 'text' }
    ])
  })
  it('uses the selected user key without attaching login credentials', async () => {
    const fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ data: [{ id: 'gpt-image-1' }] }) })
    vi.stubGlobal('fetch', fetch)
    await loadCanvasModels(key, new AbortController().signal)
    expect(fetch).toHaveBeenCalledWith(expect.stringMatching(/\/v1\/models$/), expect.objectContaining({ headers: { Authorization: 'Bearer test-user-key' }, credentials: 'omit' }))
  })
  it('rejects catalog authorization failures', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 403 }))
    await expect(loadCanvasModels(key, new AbortController().signal)).rejects.toThrow('HTTP 403')
  })
})
