import { useEffect, useRef, useState, type ComponentProps, type Dispatch, type SetStateAction } from 'react'
import { App, Button, Dropdown, Modal, Tooltip } from 'antd'
import { ArrowLeft, Check, ChevronDown, Download, Ellipsis, Focus, FolderOpen, Hand, ImagePlus, Images, Layers, LoaderCircle, Minus, MousePointer2, Plus, Redo2, Settings2, Sparkles, Trash2, Type, Undo2, Upload, X } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { ModelPicker } from '@/components/model-picker'
import { ImageSettingsPanel } from '@/components/image-settings-panel'
import { computeMediaSize, inferMediaRatio, inferMediaScale } from '@/lib/media-size'
import { createCanvasNode } from '@/lib/canvas/canvas-node-factory'
import { resolveModelForCapability, type AiConfig } from '@/stores/use-config-store'
import { canvasThemes } from '@/lib/canvas-theme'
import { useThemeStore } from '@/stores/use-theme-store'
import { useCanvasSidePanelStore } from '@/stores/use-canvas-side-panel-store'
import { CanvasNodeType, type CanvasNodeData, type CanvasConnection, type Position, type ViewportTransform } from '@/types/canvas'
import type { CanvasTopBar } from '@/components/canvas/canvas-top-bar'
import type { CanvasToolbar } from '@/components/canvas/canvas-toolbar'
import type { CanvasNodeHoverToolbar } from '@/components/canvas/canvas-node-hover-toolbar'
import type { CanvasZoomControls } from '@/components/canvas/canvas-zoom-controls'

const zh = () => window.__SUB2API_CANVAS__.locale.startsWith('zh')
const label = (cn: string, en: string) => zh() ? cn : en
type Draft = { prompt: string; model?: string; size: string; quality: string; count: number; background: string; references: string[] }
const emptyDraft = (): Draft => ({ prompt: '', size: '1024x1024', quality: 'auto', count: 1, background: '', references: [] })
function readDraft(projectId: string): Draft {
  try {
    const data = JSON.parse(localStorage.getItem(`workspace-draft:${projectId}`) || '{}')
    const draft = emptyDraft()
    if (typeof data.prompt === 'string') draft.prompt = data.prompt
    if (typeof data.model === 'string') draft.model = data.model
    if (typeof data.size === 'string') draft.size = data.size
    if (typeof data.quality === 'string') draft.quality = data.quality
    if (typeof data.background === 'string') draft.background = data.background
    draft.count = Math.min(4, Math.max(1, Math.floor(Number(data.count) || 1)))
    draft.references = Array.isArray(data.references) ? data.references.filter((id: unknown) => typeof id === 'string') : []
    return draft
  } catch { return emptyDraft() }
}

type WorkspaceOptions = {
  projectId: string; loaded: boolean; nodes: CanvasNodeData[]; config: AiConfig; runningId: string | null
  setNodes: Dispatch<SetStateAction<CanvasNodeData[]>>; setConnections: Dispatch<SetStateAction<CanvasConnection[]>>
  select: (ids: Set<string>) => void; center: () => Position; setViewport: Dispatch<SetStateAction<ViewportTransform>>
  size: { width: number; height: number }; generate: (id: string, mode: 'image', prompt: string) => Promise<void>
  onFailure: (id: string) => void
}

export function useSimpleWorkspace(options: WorkspaceOptions) {
  const [draft, setDraft] = useState(() => readDraft(options.projectId))
  const [pending, setPending] = useState<{ id: string; prompt: string } | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const submitted = useRef<string | null>(null)
  const locked = useRef(false)
  const { message } = App.useApp()
  const draftProject = useRef(options.projectId)
  useEffect(() => {
    if (draftProject.current !== options.projectId) {
      draftProject.current = options.projectId
      setDraft(readDraft(options.projectId))
      return
    }
    try { localStorage.setItem(`workspace-draft:${options.projectId}`, JSON.stringify(draft)) } catch { /* Storage may be full; the canvas still works in memory. */ }
  }, [draft, options.projectId])
  const references = draft.references.map(id => options.nodes.find(node => node.id === id && node.type === CanvasNodeType.Image && node.metadata?.content)).filter((node): node is CanvasNodeData => Boolean(node))
  const model = draft.model && options.config.models.includes(draft.model) ? draft.model : options.config.imageModel
  const config: AiConfig = { ...options.config, model: resolveModelForCapability(options.config, model, 'image'), size: draft.size, quality: draft.quality, background: draft.background, count: String(draft.count) }
  const addReferences = (ids: string[]) => setDraft(current => ({ ...current, references: [...new Set([...current.references, ...ids.filter(id => options.nodes.some(node => node.id === id && node.type === CanvasNodeType.Image && node.metadata?.content))])] }))
  const generate = () => {
    if (locked.current || options.runningId || !draft.prompt.trim() || !options.loaded) return
    locked.current = true
    setSubmitting(true)
    const center = options.center()
    const source = references[references.length - 1]
    const node = createCanvasNode(CanvasNodeType.Image, center, { prompt: draft.prompt.trim(), model: config.model, size: draft.size, quality: draft.quality, count: draft.count, background: draft.background })
    if (source) node.position = { x: source.position.x + source.width + 100, y: source.position.y }
    // Find a free slot instead of stacking repeated generations over each other.
    let attempts = 0
    while (attempts++ < 100 && options.nodes.some(existing => node.position.x < existing.position.x + existing.width + 32 && node.position.x + node.width + 32 > existing.position.x && node.position.y < existing.position.y + existing.height + 32 && node.position.y + node.height + 32 > existing.position.y)) node.position.y += node.height + 56
    options.setNodes(previous => [...previous, node])
    options.setConnections(previous => [...previous, ...references.map(source => ({ id: crypto.randomUUID(), fromNodeId: source.id, toNodeId: node.id }))])
    options.select(new Set([node.id]))
    const frame = [...references, node]
    const left = Math.min(...frame.map(item => item.position.x))
    const right = Math.max(...frame.map(item => item.position.x + item.width))
    const top = Math.min(...frame.map(item => item.position.y))
    const bottom = Math.max(...frame.map(item => item.position.y + item.height))
    const availableHeight = Math.max(120, options.size.height - 310)
    const scale = Math.max(.1, Math.min(1, (options.size.width - 110) / (right - left), availableHeight / (bottom - top)))
    options.setViewport({ k: scale, x: options.size.width / 2 - (left + right) / 2 * scale, y: 60 + availableHeight / 2 - (top + bottom) / 2 * scale })
    setPending({ id: node.id, prompt: draft.prompt.trim() })
  }
  useEffect(() => {
    if (!pending || submitted.current === pending.id || !options.nodes.some(node => node.id === pending.id)) return
    submitted.current = pending.id
    // Run after the canvas has committed both the new node and its reference connections.
    void options.generate(pending.id, 'image', pending.prompt).catch(error => {
      options.onFailure(pending.id)
      const detail = error instanceof Error ? error.message : label('生成失败，请重试', 'Generation failed. Please retry.')
      options.setNodes(previous => previous.map(node => node.id === pending.id ? { ...node, metadata: { ...node.metadata, status: 'error', errorDetails: detail } } : node))
      message.error(detail)
    }).finally(() => { locked.current = false; setSubmitting(false); setPending(null) })
  }, [pending, options.nodes, options.generate])
  return { draft, setDraft, config, references, addReferences, generate, busy: submitting || Boolean(options.runningId) }
}

export function WorkspaceComposer({ workspace, nodes, onUpload, onStop }: { workspace: ReturnType<typeof useSimpleWorkspace>; nodes: CanvasNodeData[]; onUpload: () => void; onStop: () => void }) {
  const { draft, setDraft, config, references, addReferences, generate, busy } = workspace
  const [picker, setPicker] = useState(false)
  const [settings, setSettings] = useState(false)
  const theme = canvasThemes[useThemeStore(state => state.theme)]
  const input = useRef<HTMLTextAreaElement>(null)
  const available = nodes.filter(node => node.type === CanvasNodeType.Image && node.metadata?.content)
  const change = (patch: Partial<Draft>) => setDraft(current => ({ ...current, ...patch }))
  const ratios = ['1:1', '3:2', '2:3', '16:9', '9:16', 'auto']
  return <>
    <form className='workspace-composer' data-canvas-no-zoom aria-label={label('图片创作', 'Image composer')} onSubmit={event => { event.preventDefault(); generate() }} onPointerDown={event => event.stopPropagation()} onMouseDown={event => event.stopPropagation()}>
      <div className='workspace-references'>
        {references.map(node => <div className='workspace-reference' key={node.id} title={node.title}>
          <img src={node.metadata!.content} alt={node.title} />
          <button type='button' aria-label={label('移除参考图', 'Remove reference')} onClick={() => change({ references: draft.references.filter(id => id !== node.id) })}><X size={12} /></button>
        </div>)}
        <button type='button' className='workspace-reference-add' onClick={() => setPicker(true)}><Plus size={16} /><span>{label('添加参考', 'Add reference')}</span></button>
        <Tooltip title={label('上传图片到画布', 'Upload to canvas')}><button type='button' className='workspace-icon-button' aria-label={label('上传图片', 'Upload image')} onClick={onUpload}><Upload size={16} /></button></Tooltip>
        {references.length > 0 && <span className='workspace-reference-count'>{label(`${references.length} 张参考图`, `${references.length} references`)}</span>}
      </div>
      <textarea ref={input} value={draft.prompt} onChange={event => change({ prompt: event.target.value })} placeholder={label('描述你想生成的图片，或添加参考图继续创作…', 'Describe an image, or add references to keep creating…')} aria-label={label('图片描述', 'Image prompt')} onKeyDown={event => { if ((event.metaKey || event.ctrlKey) && event.key === 'Enter' && !event.nativeEvent.isComposing) { event.preventDefault(); generate() } }} />
      <div className='workspace-composer-footer'>
        <ModelPicker config={config} value={config.model} capability='image' onChange={model => change({ model })} className='workspace-model' />
        <select aria-label={label('图片比例', 'Aspect ratio')} value={inferMediaRatio(draft.size)} onChange={event => change({ size: computeMediaSize(inferMediaScale(draft.size), event.target.value) })}>{ratios.map(ratio => <option key={ratio} value={ratio}>{ratio === 'auto' ? label('自动', 'Auto') : ratio}</option>)}</select>
        <button type='button' className='workspace-icon-button' aria-label={label('更多生成设置', 'More generation settings')} title={label('更多生成设置', 'More generation settings')} onClick={() => setSettings(true)}><Settings2 size={16} />{draft.count > 1 && <span>{draft.count}</span>}</button>
        {busy && <button type='button' className='workspace-icon-button workspace-stop' aria-label={label('停止生成', 'Stop generation')} title={label('停止生成', 'Stop generation')} onClick={onStop}><X size={16} /></button>}
        <button type='submit' aria-label={busy ? label('正在生成', 'Generating') : label('生成图片', 'Generate image')} className={`workspace-generate ${busy ? 'is-running' : ''}`} disabled={busy || !draft.prompt.trim()}>
          {busy ? <><LoaderCircle size={16} className='animate-spin' /><span>{label('正在生成', 'Generating')}</span></> : <><Sparkles size={16} /><span>{label('生成图片', 'Generate')}</span></>}
        </button>
      </div>
    </form>
    <Modal title={label('添加参考图', 'Add reference images')} open={picker} onCancel={() => setPicker(false)} footer={<Button type='primary' onClick={() => { setPicker(false); input.current?.focus() }}>{label('完成', 'Done')}</Button>} width={620}>
      {available.length ? <div className='workspace-image-picker'>{available.map(node => <button type='button' key={node.id} aria-pressed={draft.references.includes(node.id)} aria-label={node.title} onClick={() => draft.references.includes(node.id) ? change({ references: draft.references.filter(id => id !== node.id) }) : addReferences([node.id])}>
        <img src={node.metadata!.content} alt='' />{draft.references.includes(node.id) && <span><Check size={16} /></span>}<small>{node.title}</small>
      </button>)}</div> : <div className='workspace-picker-empty'><ImagePlus size={32} /><p>{label('先生成一张图片，或将图片拖入画布', 'Generate an image or drag one onto the canvas first')}</p><Button onClick={() => { setPicker(false); onUpload() }}>{label('上传图片', 'Upload image')}</Button></div>}
    </Modal>
    <Modal title={label('生成设置', 'Generation settings')} open={settings} onCancel={() => setSettings(false)} footer={<Button type='primary' onClick={() => setSettings(false)}>{label('完成', 'Done')}</Button>} width={420}>
      <ImageSettingsPanel config={config} theme={theme} showTitle={false} className='workspace-image-settings' maxCount={4} quickCount={4} onConfigChange={(key, value) => change({ [key]: key === 'count' ? Math.min(4, Math.max(1, Number(value) || 1)) : value })} />
    </Modal>
  </>
}

export function WorkspaceTopBar(props: ComponentProps<typeof CanvasTopBar> & { busy?: boolean }) {
  const navigate = useNavigate()
  return <header className='workspace-topbar' data-canvas-no-zoom>
    <button className='workspace-icon-button' aria-label={label('返回画布列表', 'Back to canvases')} title={label('返回画布列表', 'Back to canvases')} disabled={props.busy} onClick={props.onProjects}><ArrowLeft size={17} /></button>
    {props.isTitleEditing ? <input aria-label={label('画布名称', 'Canvas name')} autoFocus value={props.titleDraft} onChange={event => props.onTitleDraftChange(event.target.value)} onBlur={props.onFinishTitleEditing} onKeyDown={event => { if (event.key === 'Enter') props.onFinishTitleEditing(); if (event.key === 'Escape') props.onCancelTitleEditing() }} /> : <Dropdown trigger={['click']} menu={{ items: [
      { key: 'rename', label: label('重命名', 'Rename'), onClick: props.onStartTitleEditing },
      { key: 'new', label: label('新建画布', 'New canvas'), disabled: props.busy, icon: <Plus size={15} />, onClick: props.onCreateProject },
      { key: 'list', label: label('所有画布', 'All canvases'), disabled: props.busy, icon: <Layers size={15} />, onClick: props.onProjects },
      { type: 'divider' },
      { key: 'export', label: label('导出画布', 'Export canvas'), icon: <Download size={15} />, onClick: props.onExportProject },
      { key: 'delete', label: label('删除画布', 'Delete canvas'), danger: true, disabled: props.busy, onClick: props.onDeleteProject },
    ] }}><button className='workspace-project-name'>{props.title}<ChevronDown size={14} /></button></Dropdown>}
    <div className='workspace-history'><button className='workspace-icon-button' title={label('撤销', 'Undo')} aria-label={label('撤销', 'Undo')} disabled={!props.canUndo} onClick={props.onUndo}><Undo2 size={16} /></button><button className='workspace-icon-button' title={label('重做', 'Redo')} aria-label={label('重做', 'Redo')} disabled={!props.canRedo} onClick={props.onRedo}><Redo2 size={16} /></button></div>
    <button className='workspace-library-button' aria-label={label('作品库', 'Library')} disabled={props.busy} onClick={() => navigate('/assets')}><FolderOpen size={16} /><span>{label('作品库', 'Library')}</span></button>
  </header>
}

export function WorkspaceTools(props: ComponentProps<typeof CanvasToolbar>) {
  const [more, setMore] = useState(false)
  const panel = useCanvasSidePanelStore()
  return <div className='workspace-tools' data-canvas-no-zoom>
    <Tooltip title={label('选择 / 移动画布', 'Select / move canvas')} placement='right'><button aria-label={label('切换选择和移动', 'Toggle select and pan')} className='workspace-icon-button is-active' onClick={() => props.onCanvasToolChange(props.canvasTool === 'pan' ? 'select' : 'pan')}>{props.canvasTool === 'pan' ? <Hand size={18} /> : <MousePointer2 size={18} />}</button></Tooltip>
    <Tooltip title={label('上传图片', 'Upload image')} placement='right'><button className='workspace-icon-button' aria-label={label('上传图片到画布', 'Upload image to canvas')} onClick={props.onUpload}><ImagePlus size={18} /></button></Tooltip>
    <Tooltip title={label('画布元素', 'Canvas elements')} placement='right'><button className={`workspace-icon-button ${panel.panelOpen ? 'is-active' : ''}`} aria-label={label('展开画布元素', 'Toggle canvas elements')} onClick={panel.togglePanel}><Layers size={18} /></button></Tooltip>
    <Dropdown open={more} onOpenChange={setMore} trigger={['click']} placement='bottomLeft' menu={{ items: [
      { key: 'text', label: label('添加文字', 'Add text'), icon: <Type size={16} />, onClick: props.onAddText },
      { key: 'group', label: label('添加分组', 'Add group'), onClick: props.onAddGroup },
      { key: 'config', label: label('高级生成节点', 'Advanced generation node'), onClick: props.onAddConfig },
      { key: 'focus', label: label('聚焦选中', 'Focus selection'), disabled: !props.selectedCount, onClick: props.onFocusSelected },
      { type: 'divider' },
      { key: 'grid', label: label('切换网格', 'Toggle grid'), onClick: () => props.onBackgroundModeChange(props.backgroundMode === 'dots' ? 'blank' : 'dots') },
      { key: 'delete', label: label('删除选中', 'Delete selection'), disabled: !props.selectedCount, danger: true, onClick: props.onDelete },
      { key: 'clear', label: label('清空画布', 'Clear canvas'), danger: true, onClick: props.onClear },
    ] }}><button className='workspace-icon-button' aria-label={label('更多画布工具', 'More canvas tools')}><Ellipsis size={18} /></button></Dropdown>
  </div>
}

export function WorkspaceNodeToolbar({ onUseReference, ...props }: ComponentProps<typeof CanvasNodeHoverToolbar> & { onUseReference: (id: string) => void }) {
  const node = props.node
  if (!node) return null
  const hasImage = node.type === CanvasNodeType.Image && Boolean(node.metadata?.content)
  const left = props.viewport.x + (node.position.x + node.width / 2) * props.viewport.k
  const top = Math.max(60, props.viewport.y + node.position.y * props.viewport.k - 48)
  return <div className='workspace-node-toolbar' data-node-toolbar={node.id} style={{ left: Math.min(window.innerWidth - 125, Math.max(125, left)), top }} onMouseDown={event => event.stopPropagation()} onPointerDown={event => event.stopPropagation()} onMouseEnter={() => props.onKeep(node.id)}>
    {hasImage ? <><button className='workspace-reference-action' onClick={() => onUseReference(node.id)}><ImagePlus size={15} />{label('用作参考', 'Use as reference')}</button><button className='workspace-icon-button' aria-label={label('下载图片', 'Download image')} title={label('下载图片', 'Download image')} onClick={() => props.onDownload(node)}><Download size={16} /></button></> : <button className='workspace-icon-button' aria-label={label('编辑节点', 'Edit node')} onClick={() => props.onToggleDialog(node)}><Settings2 size={16} /></button>}
    <Dropdown trigger={['click']} menu={{ items: [
      ...(hasImage ? [
        { key: 'preview', label: label('放大预览', 'Preview'), onClick: () => props.onViewImage(node) },
        { key: 'save', label: label('保存到作品库', 'Save to library'), onClick: () => props.onSaveAsset(node) },
        { key: 'crop', label: label('裁剪', 'Crop'), onClick: () => props.onCrop(node) },
        { key: 'mask', label: label('局部重绘', 'Inpaint'), onClick: () => props.onMaskEdit(node) },
      ] : []),
      ...(node.metadata?.status === 'error' ? [{ key: 'retry', label: label('重试生成', 'Retry'), onClick: () => props.onRetry(node) }] : []),
      { key: 'info', label: label('查看详情', 'Details'), onClick: () => props.onInfo(node) },
      { key: 'delete', label: label('删除', 'Delete'), danger: true, icon: <Trash2 size={15} />, onClick: () => props.onDelete(node) },
    ] }}><button className='workspace-icon-button' aria-label={label('更多图片操作', 'More image actions')}><Ellipsis size={16} /></button></Dropdown>
  </div>
}

export function WorkspaceEmpty({ onUpload }: { onUpload: () => void }) {
  return <div className='workspace-empty'><div className='workspace-empty-symbol'><Images size={30} /></div><h2>{label('从一个想法开始', 'Start with an idea')}</h2><p>{label('在下方描述画面，或上传图片继续创作', 'Describe your image below, or upload one to continue')}</p><button onClick={onUpload}><Upload size={15} />{label('上传图片', 'Upload image')}</button></div>
}

export function WorkspaceZoom(props: ComponentProps<typeof CanvasZoomControls>) {
  return <div className='workspace-zoom' data-canvas-no-zoom>
    <button className='workspace-icon-button' aria-label={label('缩小画布', 'Zoom out')} onClick={() => props.onScaleChange(Math.max(.05, props.scale / 1.2))}><Minus size={14} /></button>
    <span>{Math.round(props.scale * 100)}%</span>
    <button className='workspace-icon-button' aria-label={label('放大画布', 'Zoom in')} onClick={() => props.onScaleChange(Math.min(5, props.scale * 1.2))}><Plus size={14} /></button>
    <i />
    <button className='workspace-icon-button' aria-label={label('适应画布', 'Fit canvas')} title={label('适应画布', 'Fit canvas')} onClick={props.onReset}><Focus size={15} /></button>
  </div>
}
