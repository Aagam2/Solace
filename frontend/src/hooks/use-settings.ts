import { useCallback, useEffect, useRef, useState } from 'react'
import { appBridge, defaultSettings } from '../lib/bridge'
import type { Settings, Theme } from '../types'

export function useSettings() {
  const [settings, setSettings] = useState(defaultSettings)
  const settingsRef = useRef(defaultSettings)
  const [ready, setReady] = useState(false)
  useEffect(() => { appBridge.loadSettings().then((value) => { settingsRef.current=value; setSettings(value) }).finally(() => setReady(true)) }, [])
  useEffect(() => {
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const apply = () => {
      const dark = settings.theme === 'dark' || (settings.theme === 'system' && media.matches)
      document.documentElement.dataset.theme = dark ? 'dark' : 'light'
    }
    apply(); media.addEventListener('change', apply)
    return () => media.removeEventListener('change', apply)
  }, [settings.theme])
  useEffect(() => {
    const widths = { narrow:'620px', comfortable:'720px', wide:'860px' }
    document.documentElement.style.setProperty('--editor-prose-width', widths[settings.editorWidth])
    document.documentElement.style.setProperty('--editor-font-size', `${settings.editorFontSize}px`)
    document.documentElement.style.setProperty('--editor-line-spacing', String(settings.editorLineSpacing))
  }, [settings.editorFontSize, settings.editorLineSpacing, settings.editorWidth])
  const updateSettings = useCallback((patch: Partial<Settings>) => {
    const next = { ...settingsRef.current, ...patch }
    settingsRef.current = next
    setSettings(next)
    void appBridge.saveSettings(next)
  }, [])
  const setTheme = useCallback((theme:Theme) => updateSettings({ theme }), [updateSettings])
  const toggleSidebar = useCallback(() => updateSettings({ sidebarCollapsed:!settingsRef.current.sidebarCollapsed }), [updateSettings])
  return {
    settings, ready,
    setTheme,
    toggleSidebar,
    updateSettings,
  }
}
