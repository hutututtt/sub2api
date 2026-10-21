import React from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import axios from 'axios'
import 'antd/dist/reset.css'
import 'streamdown/styles.css'
import './styles/globals.css'
import './embedded.css'
import './workspace.css'
import { AppProviders } from '@/components/layout/app-providers'
import { useConfigStore } from '@/stores/use-config-store'
import { useThemeStore } from '@/stores/use-theme-store'
import i18n from '@/i18n'
import { router } from './router'
import { notifyHost } from './context'

export function render() {
  const session = window.__SUB2API_CANVAS__
  const imageModels = session.models.filter(m => m.capability === 'image')
  const textModels = session.models.filter(m => m.capability === 'text')
  const current = useConfigStore.getState().config
  const image = imageModels.find(m => `sub2api::${m.name}` === current.imageModel) ?? imageModels[0]
  const text = textModels.find(m => `sub2api::${m.name}` === current.textModel) ?? textModels[0]
  useConfigStore.setState({ config: {
    ...current,
    baseUrl: session.baseUrl, apiKey: session.apiKey, apiFormat: session.apiFormat, channelMode: 'local',
    channels: [{ id: 'sub2api', name: 'Sub2API', baseUrl: session.baseUrl, apiKey: session.apiKey, apiFormat: session.apiFormat, models: session.models }],
    models: session.models.map(m => `sub2api::${m.name}`),
    imageModel: `sub2api::${image.name}`, model: `sub2api::${image.name}`,
    textModel: text ? `sub2api::${text.name}` : '', audioModel: '', videoModel: '',
    proxyEnabled: false, count: '1', canvasImageCount: '1'
  } })
  useThemeStore.getState().setTheme(session.theme)
  window.addEventListener('message', event => {
    if (event.source !== window.parent || event.origin !== location.origin || event.data?.type !== 'sub2api:canvas:theme') return
    if (event.data.theme === 'light' || event.data.theme === 'dark') useThemeStore.getState().setTheme(event.data.theme)
  })
  void i18n.changeLanguage(session.locale.startsWith('zh') ? 'zh-CN' : 'en-US')
  let pending = 0
  const isGeneration = (url = '') => /\/(images\/(generations|edits)|responses|chat\/completions)(\?|$)|:generateContent|:streamGenerateContent/.test(url)
  axios.interceptors.request.use(config => {
    const url = new URL(config.url!, location.origin)
    // This embedded build only uses the selected user's gateway channel.
    if (url.origin !== new URL(session.baseUrl).origin || !/^\/v1(beta)?\//.test(url.pathname)) throw new Error('This request is outside the selected Sub2API channel.')
    if (isGeneration(config.url)) { pending++; notifyHost('sub2api:canvas:busy', { busy: true }) }
    return config
  })
  const settled = (url?: string) => {
    if (!isGeneration(url)) return
    pending = Math.max(0, pending - 1)
    notifyHost('sub2api:canvas:busy', { busy: pending > 0 })
    notifyHost('sub2api:canvas:settled')
  }
  axios.interceptors.response.use(response => { settled(response.config.url); return response }, error => { settled(error.config?.url); return Promise.reject(error) })
  const originalFetch = window.fetch.bind(window)
  window.fetch = async (input, init) => {
    const url = input instanceof Request ? input.url : String(input)
    if (!isGeneration(url)) return originalFetch(input, init)
    const target = new URL(url, location.origin)
    if (target.origin !== new URL(session.baseUrl).origin || !/^\/v1(beta)?\//.test(target.pathname)) throw new Error('Invalid Sub2API channel')
    pending++; notifyHost('sub2api:canvas:busy', { busy: true })
    let response: Response
    try { response = await originalFetch(input, init) } catch (error) { settled(url); throw error }
    if (!response.body) { settled(url); return response }
    const reader = response.body.getReader()
    let finished = false
    const finish = () => { if (!finished) { finished = true; settled(url) } }
    return new Response(new ReadableStream({
      async pull(controller) {
        try {
          const { done, value } = await reader.read()
          if (done) { controller.close(); finish() } else controller.enqueue(value)
        } catch (error) { controller.error(error); finish() }
      },
      async cancel(reason) { try { await reader.cancel(reason) } finally { finish() } }
    }), { status: response.status, statusText: response.statusText, headers: response.headers })
  }
  createRoot(document.getElementById('root')!).render(<React.StrictMode><AppProviders><RouterProvider router={router} /></AppProviders></React.StrictMode>)
}
