import { useEffect } from 'react'
import { createHashRouter, Navigate } from 'react-router-dom'
import { notifyHost } from './context'
import HostLayout from './host-layout'
import CanvasPage from '@/pages/canvas'
import CanvasProjectPage from '@/pages/canvas/project'
import AssetsPage from '@/pages/assets'

function CanvasError() {
  useEffect(() => { notifyHost('sub2api:canvas:error') }, [])
  return null
}

export const router = createHashRouter([{
  element: <HostLayout />,
  errorElement: <CanvasError />,
  children: [
    { path: '/canvas', element: <CanvasPage /> },
    { path: '/canvas/:id', element: <CanvasProjectPage /> },
    { path: '/assets', element: <AssetsPage /> },
    { path: '*', element: <Navigate to='/canvas?mode=recent' replace /> }
  ]
}])
