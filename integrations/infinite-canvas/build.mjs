import { cp, mkdir, readFile, readdir, writeFile, rm } from 'node:fs/promises'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { createHash } from 'node:crypto'
import { applyWorkspace } from './apply-workspace.mjs'

const root = fileURLToPath(new URL('../../', import.meta.url))
const revision = '9f2bd9f1b27c45fd1b8f5323072b07243f9e0fac'
const source = path.join(root, '.dev/infinite-canvas-upstream')
const build = path.join(root, '.dev/infinite-canvas-build')
const integration = fileURLToPath(new URL('./', import.meta.url))
const out = path.join(root, 'frontend/public/canvas-app')
const hash = createHash('sha256')
for (const file of (await readdir(integration)).filter(file => /\.(mjs|ts|tsx|css)$/.test(file)).sort()) {
  hash.update(file).update(await readFile(path.join(integration, file)))
}
const integrationSha256 = hash.digest('hex')
if (process.argv.includes('--if-needed')) {
  try {
    const manifest = JSON.parse(await readFile(path.join(out, 'upstream.json'), 'utf8'))
    if (manifest.integrationSha256 === integrationSha256 && manifest.revision === revision && manifest.files?.length) {
      for (const file of manifest.files) {
        if (createHash('sha256').update(await readFile(path.join(out, file.path))).digest('hex') !== file.sha256) throw new Error('Stale canvas asset')
      }
      console.log('Infinite Canvas assets verified')
      process.exit(0)
    }
  } catch { /* Missing or stale assets are rebuilt below. */ }
}
const run = (cmd, args, cwd = root, env = {}) => execFileSync(cmd, args, { cwd, stdio: 'inherit', env: { ...process.env, ...env } })
await mkdir(path.join(root, '.dev'), { recursive: true })
try { await readFile(path.join(source, 'LICENSE')) } catch {
  run('git', ['clone', '--no-checkout', 'https://github.com/basketikun/infinite-canvas.git', source])
  run('git', ['checkout', '--detach', revision], source)
}
const actual = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: source, encoding: 'utf8' }).trim()
if (actual !== revision) throw new Error(`Expected upstream ${revision}, got ${actual}`)
await cp(source, build, { recursive: true, filter: p => !['.git', 'node_modules', 'dist'].includes(path.basename(p)) })
const web = path.join(build, 'web')
async function replace(file, from, to) {
  const target = path.join(web, file)
  const content = await readFile(target, 'utf8')
  if (!content.includes(from)) throw new Error(`Upstream changed: ${file}`)
  await writeFile(target, content.replace(from, to))
}
// Keep the upstream application intact, with scoped storage and host-owned credentials.
async function scopeDatabases(dir) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const file = path.join(dir, entry.name)
    if (entry.isDirectory()) await scopeDatabases(file)
    else if (/\.tsx?$/.test(file)) {
      const text = await readFile(file, 'utf8')
      if (text.includes('localforage')) await writeFile(file, text.replaceAll('name: "infinite-canvas"', 'name: window.__SUB2API_CANVAS__.storageName'))
    }
  }
}
await scopeDatabases(path.join(web, 'src'))
await replace('src/components/layout/app-providers.tsx', 'import { ClientRootInit } from "@/components/layout/client-root-init";', '')
await replace('src/components/layout/app-providers.tsx', '<ClientRootInit>{children}</ClientRootInit>', '{children}')
await replace('src/pages/canvas/project.tsx', 'if (!searchParams.has("agentUrl") && !localAgentEnabled && !fragmentBootstrap) openAgentPanel();', 'return;')
await replace('src/pages/canvas/hooks/use-plugin-host.tsx', 'void ensurePluginsLoaded();', '/* Remote plugins are disabled in the embedded application. */')
await replace('src/stores/use-canvas-side-panel-store.ts', 'return localStorage.getItem(OPEN_KEY) !== "0";', 'return window.innerWidth >= 768 && localStorage.getItem(OPEN_KEY) !== "0";')
await replace('src/components/canvas/canvas-node.tsx', 'import React,', 'import { EmbeddedNodePanel } from "@/node-panel";\nimport React,')
await replace('src/components/canvas/canvas-node.tsx', 'showPanel: boolean;', 'showPanel: boolean;\n    onDismissPanel?: () => void;')
await replace('src/components/canvas/canvas-node.tsx', '    showPanel,', '    showPanel,\n    onDismissPanel,')
await replace('src/components/canvas/canvas-node.tsx', '<div className="absolute left-1/2 top-full z-[70] w-[600px] pt-4" style={{ transform: `translateX(-50%) scale(${1 / scale})`, transformOrigin: "top center" }} data-node-panel={data.id}>\n                    {renderPanel(data)}\n                </div>', '<EmbeddedNodePanel nodeId={data.id} scale={scale} onClose={onDismissPanel}>\n                    {renderPanel(data)}\n                </EmbeddedNodePanel>')
await replace('src/pages/canvas/project.tsx', 'renderPanel={renderNodePanel}', 'renderPanel={renderNodePanel}\n                            onDismissPanel={() => setDialogNodeId(null)}')
await replace('src/components/canvas/canvas-node-prompt-panel.tsx', 'mt-2 flex min-w-0 items-center', 'mt-2 flex min-w-0 flex-wrap items-center')
await replace('src/components/canvas/canvas-node-prompt-panel.tsx', 'flex min-w-0 items-center gap-2', 'flex min-w-0 flex-wrap items-center gap-2')
await replace('src/components/canvas/canvas-node-prompt-panel.tsx', 'import { CanvasPromptChipInput } from "./canvas-prompt-chip-input";', 'import { CanvasResourceMentionTextarea as CanvasPromptChipInput } from "./canvas-resource-mention-textarea";')
await replace('src/components/canvas/canvas-node-prompt-panel.tsx', 'data-canvas-no-zoom', 'data-canvas-prompt-panel data-canvas-no-zoom')
await replace('src/components/canvas/canvas-node-prompt-panel.tsx', 'className="mt-2 flex', 'data-canvas-prompt-actions className="mt-2 flex')
await replace('src/components/canvas/canvas-node-prompt-panel.tsx', 'type="primary"', 'data-canvas-generate type="primary"')
for (const file of ['src/components/canvas/canvas-node-prompt-panel.tsx', 'src/components/canvas/canvas-prompt-library.tsx']) {
  const target = path.join(web, file)
  await writeFile(target, (await readFile(target, 'utf8')).replaceAll('!rounded-full', '!rounded-md').replaceAll('!h-10', '!h-[34px]').replaceAll('!min-w-16', '!min-w-[38px]'))
}
await replace('src/components/model-picker.tsx', 'gap-2 rounded-full border', 'gap-2 rounded-md border')
await replace('src/components/canvas/canvas-node.tsx', 'const selectionBlue = "#2f80ff";', 'const selectionBlue = "#14b8a6";')
await replace('src/components/canvas/canvas-top-bar.tsx', '<div className="pointer-events-none absolute', '<div data-canvas-topbar className="pointer-events-none absolute')
for (const file of ['src/components/canvas/canvas-node-hover-toolbar.tsx', 'src/components/canvas/canvas-selection-toolbar.tsx']) {
  await replace(file, 'className="absolute z-[70] flex h-12', 'data-canvas-node-tools className="absolute z-[70] flex h-12')
}
await replace('src/components/canvas/canvas-image-settings-popover.tsx', 'const width = 356;', 'const width = Math.min(356, window.innerWidth - 24);')
await replace('src/components/canvas/canvas-zoom-controls.tsx', '<div className="absolute bottom-5 left-5 z-50"', '<div data-canvas-zoom-controls className="absolute bottom-5 left-5 z-50"')
await replace('src/components/canvas/canvas-top-bar.tsx', '<CompactAgentStatus status={compactAgentStatus} onClick={onToggleAgent} />', '')
const topbar = path.join(web, 'src/components/canvas/canvas-top-bar.tsx')
await writeFile(topbar, (await readFile(topbar, 'utf8')).replace(/<div className="pointer-events-auto flex items-center gap-1\.5">[\s\S]*?<\/div>/, ''))
const toolbar = path.join(web, 'src/components/canvas/canvas-toolbar.tsx')
await writeFile(toolbar, (await readFile(toolbar, 'utf8')).replace(/\s*<ToolbarButton id="tool-(?:video|audio)"[\s\S]*?<\/ToolbarButton>/g, ''))
const menus = path.join(web, 'src/components/canvas/canvas-create-menus.tsx')
await writeFile(menus, (await readFile(menus, 'utf8')).replace(/^.*<ConnectionCreateOption.*onCreate\(CanvasNodeType\.(Video|Audio)\).*\n/gm, '').replace('def.showInCreateMenu !== false', 'def.showInCreateMenu !== false && ![CanvasNodeType.Video, CanvasNodeType.Audio].includes(def.type as CanvasNodeType)'))
const picker = path.join(web, 'src/components/model-picker.tsx')
await writeFile(picker, (await readFile(picker, 'utf8')).replaceAll('"/icons/', '"/canvas-app/icons/'))
await replace('src/stores/use-config-store.ts', 'partialize: (state) => ({ config: state.config, webdav: state.webdav }),', 'partialize: (state) => ({ config: { ...state.config, apiKey: "", channels: state.config.channels.map(channel => ({ ...channel, apiKey: "" })) }, webdav: state.webdav }),')
await replace('index.html', '<script src="/config.js"></script>', '')
await replace('index.html', 'href="/logo.svg"', 'href="/canvas-app/logo.svg"')
const html = await readFile(path.join(web, 'index.html'), 'utf8')
await writeFile(path.join(web, 'index.html'), html.replace(/\s*<script>[\s\S]*?<\/script>/, ''))
for (const file of ['main.ts', 'render.tsx', 'router.tsx', 'context.ts', 'host-layout.tsx', 'embedded.css', 'workspace.css', 'node-panel.tsx']) {
  await cp(new URL(`./${file}`, import.meta.url), path.join(web, 'src', file))
}
for (const file of ['canvas-theme.ts', 'app-theme.ts']) {
  await cp(new URL(`./${file}`, import.meta.url), path.join(web, 'src/lib', file))
}
await applyWorkspace(web, replace)
await replace('index.html', '/src/main.tsx', '/src/main.ts')
run('bun', ['install', '--frozen-lockfile'], web)
run('bun', ['run', 'build'], web, { VITE_BASE: '/canvas-app/' })
await rm(out, { recursive: true, force: true })
await mkdir(out, { recursive: true })
await cp(path.join(web, 'dist'), out, { recursive: true })
await cp(path.join(source, 'LICENSE'), path.join(out, 'LICENSE.txt'))
const files = []
async function collect(dir) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const file = path.join(dir, entry.name)
    if (entry.isDirectory()) await collect(file)
    else files.push({ path: path.relative(out, file), sha256: createHash('sha256').update(await readFile(file)).digest('hex') })
  }
}
await collect(out)
await writeFile(path.join(out, 'upstream.json'), JSON.stringify({ repository: 'https://github.com/basketikun/infinite-canvas', revision, integrationSha256, files }, null, 2) + '\n')
console.log(`Built Infinite Canvas ${revision} at ${out}`)
