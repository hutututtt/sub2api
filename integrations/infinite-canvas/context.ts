export type CanvasSession = {
  userId: number
  keyId: number
  apiKey: string
  baseUrl: string
  apiFormat: 'openai' | 'gemini'
  models: { name: string; capability: 'image' | 'text' }[]
  locale: string
  theme: 'light' | 'dark'
  storageName: string
}

declare global {
  interface Window { __SUB2API_CANVAS__: CanvasSession }
}

export function notifyHost(type: string, extra = {}) {
  window.parent.postMessage({ type, ...extra }, window.location.origin)
}

export function scopeStorage(userId: number) {
  const prefix = `sub2api:canvas:${userId}:`
  const proto = Storage.prototype
  const get = proto.getItem
  const set = proto.setItem
  const remove = proto.removeItem
  const key = proto.key
  const length = Object.getOwnPropertyDescriptor(proto, 'length')!.get!
  const ownKeys = (storage: Storage) => Array.from({ length: length.call(storage) }, (_, i) => key.call(storage, i)!).filter(k => k?.startsWith(prefix))
  proto.getItem = function (name) { return get.call(this, prefix + name) }
  proto.setItem = function (name, value) { set.call(this, prefix + name, value) }
  proto.removeItem = function (name) { remove.call(this, prefix + name) }
  proto.clear = function () { for (const name of ownKeys(this)) remove.call(this, name) }
  proto.key = function (index) { return ownKeys(this)[index]?.slice(prefix.length) ?? null }
  Object.defineProperty(proto, 'length', { get() { return ownKeys(this).length } })
}
