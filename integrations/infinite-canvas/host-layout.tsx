import { NavLink, Outlet, useLocation } from 'react-router-dom'
import { Images, LayoutGrid, ExternalLink } from 'lucide-react'

export default function HostLayout() {
  const zh = window.__SUB2API_CANVAS__.locale.startsWith('zh')
  const isProject = /^\/canvas\/.+/.test(useLocation().pathname)
  return <div className='flex h-dvh min-w-0 flex-col overflow-hidden bg-background text-foreground'>
    {!isProject && <nav className='canvas-navigation' aria-label={zh ? '画布导航' : 'Canvas navigation'}>
      <NavLink to='/canvas'><LayoutGrid size={15} />{zh ? '我的画布' : 'My Canvases'}</NavLink>
      <NavLink to='/assets'><Images size={15} />{zh ? '素材库' : 'Assets'}</NavLink>
      <a className='canvas-attribution' href='https://github.com/basketikun/infinite-canvas' target='_blank' rel='noreferrer' title='Infinite Canvas · MIT' aria-label='Infinite Canvas · MIT'><ExternalLink size={14} /><span>Infinite Canvas</span></a>
    </nav>}
    <div className='min-h-0 flex-1 overflow-hidden'><Outlet /></div>
  </div>
}
