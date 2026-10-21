import { useEffect, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { canvasThemes } from '@/lib/canvas-theme'
import { useThemeStore } from '@/stores/use-theme-store'

export function EmbeddedNodePanel({ children, nodeId, onClose }: { children: ReactNode; nodeId: string; scale: number; onClose?: () => void }) {
  const [mobile, setMobile] = useState(window.innerWidth < 768)
  const theme = canvasThemes[useThemeStore(state => state.theme)]
  const { t } = useTranslation()
  useEffect(() => {
    const update = () => setMobile(window.innerWidth < 768)
    window.addEventListener('resize', update)
    return () => window.removeEventListener('resize', update)
  }, [])
  return createPortal(<div className={mobile ? 'canvas-composer canvas-composer-mobile' : 'canvas-composer'} data-node-panel={nodeId} onMouseDown={event => event.stopPropagation()} onPointerDown={event => event.stopPropagation()}>
    <div className='flex shrink-0 justify-end pb-1'>
      <button className='grid size-8 place-items-center rounded-md' style={{ background: theme.toolbar.panel, color: theme.node.text }} onClick={onClose} title={t('canvas.createMenu.close')} aria-label={t('canvas.createMenu.close')}><X className='size-4' /></button>
    </div>
    <div className='min-h-0 overflow-auto'>{children}</div>
  </div>, document.body)
}
