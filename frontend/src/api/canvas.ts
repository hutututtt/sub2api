import { keysAPI } from './keys'
import { buildGatewayUrl } from './url'
import type { ApiKey } from '@/types'

export interface CanvasModel { name: string; capability: 'image' | 'text' }

export function isCanvasKey(key: ApiKey, userId: number): boolean {
  return key.user_id === userId && key.status === 'active' && !!key.key &&
    (!key.expires_at || Date.parse(key.expires_at) > Date.now()) &&
    (!key.quota || key.quota_used < key.quota) && key.group?.allow_image_generation !== false &&
    ['openai', 'grok', 'gemini', 'composite'].includes(key.group?.platform ?? '')
}

export async function loadCanvasKeys(userId: number, signal: AbortSignal): Promise<ApiKey[]> {
  const keys: ApiKey[] = []
  for (let page = 1; ; page++) {
    const result = await keysAPI.list(page, 100, { status: 'active' }, { signal })
    keys.push(...result.items.filter(key => isCanvasKey(key, userId)))
    if (page >= result.pages || !result.items.length) return keys
  }
}

export function canvasModels(ids: string[], gemini: boolean): CanvasModel[] {
  return [...new Set(ids)].flatMap((name): CanvasModel[] => {
    const image = gemini ? /gemini.*image|imagen/i.test(name) : /gpt-image|grok-imagine.*image/i.test(name)
    // Imagen uses a different prediction protocol from Gemini generateContent.
    if (gemini && /imagen/i.test(name)) return []
    if (image) return [{ name, capability: 'image' }]
    const text = gemini ? /^gemini-/i.test(name) : /^(gpt-|o[134](?:-|$)|grok-)/i.test(name)
    if (text && !/image|audio|tts|realtime|transcribe|video|whisper/i.test(name)) {
      return [{ name, capability: 'text' }]
    }
    return []
  })
}

export async function loadCanvasModels(key: ApiKey, signal: AbortSignal): Promise<CanvasModel[]> {
  const response = await fetch(buildGatewayUrl('/v1/models'), {
    headers: { Authorization: `Bearer ${key.key}` }, signal, credentials: 'omit'
  })
  if (!response.ok) throw new Error(`HTTP ${response.status}`)
  const body = await response.json() as { data?: { id: string }[] }
  if (!Array.isArray(body.data)) throw new Error('Invalid model catalog')
  return canvasModels(body.data.map(model => model.id).filter(id => typeof id === 'string'), key.group?.platform === 'gemini')
}
