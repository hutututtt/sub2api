import { notifyHost, scopeStorage, type CanvasSession } from './context'

let started = false
window.addEventListener('message', async event => {
  if (event.source !== window.parent || event.origin !== location.origin || started || event.data?.type !== 'sub2api:canvas:init') return
  const session = event.data.session as CanvasSession
  if (!Number.isSafeInteger(session?.userId) || session.userId <= 0 || !session.apiKey || !session.models?.some(m => m.capability === 'image')) return
  started = true
  window.__SUB2API_CANVAS__ = { ...session, storageName: `sub2api-canvas-${session.userId}` }
  scopeStorage(session.userId)
  try {
    const { render } = await import('./render')
    render()
    notifyHost('sub2api:canvas:ready')
  } catch {
    notifyHost('sub2api:canvas:error')
  }
})
if (window.parent !== window) notifyHost('sub2api:canvas:boot')
else document.getElementById('root')!.textContent = 'Please open AI Canvas from your Sub2API account.'
