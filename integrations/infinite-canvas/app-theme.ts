import { theme, type ThemeConfig } from 'antd'

export function getAntThemeConfig(dark: boolean): ThemeConfig {
  return {
    algorithm: dark ? theme.darkAlgorithm : theme.defaultAlgorithm,
    cssVar: { key: dark ? 'sub2api-canvas-dark' : 'sub2api-canvas-light' },
    token: {
      colorPrimary: dark ? '#2dd4bf' : '#0d9488',
      colorInfo: '#0d9488',
      colorText: dark ? '#f1f5f9' : '#111827',
      colorTextSecondary: dark ? '#94a3b8' : '#6b7280',
      colorBgBase: dark ? '#0f172a' : '#ffffff',
      colorBgContainer: dark ? '#1e293b' : '#ffffff',
      colorBgElevated: dark ? '#1e293b' : '#ffffff',
      colorBorder: dark ? '#475569' : '#d1d5db',
      colorBorderSecondary: dark ? '#334155' : '#e5e7eb',
      borderRadius: 6,
      borderRadiusLG: 8,
      controlHeight: 34,
      fontFamily: 'system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif',
      boxShadowSecondary: dark ? '0 8px 24px rgba(0,0,0,.24)' : '0 8px 24px rgba(17,24,39,.08)',
    },
    components: {
      Button: { primaryShadow: 'none', fontWeight: 500 },
      Select: { optionSelectedBg: dark ? '#134e4a' : '#f0fdfa', optionSelectedColor: dark ? '#5eead4' : '#0f766e' },
      Segmented: { itemSelectedBg: dark ? '#134e4a' : '#ccfbf1', itemSelectedColor: dark ? '#5eead4' : '#0f766e' },
    },
  }
}
