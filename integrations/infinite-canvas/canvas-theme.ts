export type CanvasColorTheme = 'light' | 'dark'
export type CanvasBackgroundMode = 'dots' | 'lines' | 'blank'

export const canvasThemes = {
  light: {
    canvas: { background: '#f9fafb', dot: 'rgba(107,114,128,.22)', line: 'rgba(107,114,128,.10)', selectionStroke: '#0d9488', selectionFill: 'rgba(20,184,166,.08)' },
    node: { label: '#4b5563', fill: '#ffffff', panel: '#ffffff', stroke: '#d1d5db', activeStroke: '#0d9488', placeholder: '#9ca3af', text: '#111827', muted: '#6b7280', faint: '#9ca3af' },
    toolbar: { panel: '#ffffff', border: '#e5e7eb', item: '#4b5563', itemHover: '#f3f4f6', activeBg: '#ccfbf1', activeText: '#0f766e' },
  },
  dark: {
    canvas: { background: '#0f172a', dot: 'rgba(148,163,184,.22)', line: 'rgba(148,163,184,.10)', selectionStroke: '#2dd4bf', selectionFill: 'rgba(45,212,191,.10)' },
    node: { label: '#cbd5e1', fill: '#1e293b', panel: '#1e293b', stroke: '#475569', activeStroke: '#2dd4bf', placeholder: '#64748b', text: '#f1f5f9', muted: '#94a3b8', faint: '#64748b' },
    toolbar: { panel: '#1e293b', border: '#334155', item: '#cbd5e1', itemHover: '#334155', activeBg: '#134e4a', activeText: '#5eead4' },
  },
} as const

export type CanvasTheme = (typeof canvasThemes)[CanvasColorTheme]
