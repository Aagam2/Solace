import type { AppInfo, Settings } from '../types'
import * as NativeService from '../../bindings/solace/internal/app/service'
import { DateFormat as NativeDateFormat, EditorWidth as NativeEditorWidth, Theme as NativeTheme } from '../../bindings/solace/internal/settings/models'
import type { DataHealth } from '../../bindings/solace/internal/app/models'

export interface AppBridge {
  getInfo(): Promise<AppInfo>
  loadSettings(): Promise<Settings>
  saveSettings(settings: Settings): Promise<void>
  checkDataHealth(): Promise<DataHealth>
}

export const defaultSettings: Settings = {
  theme:'system', sidebarCollapsed:false, editorFontSize:17, editorWidth:'comfortable', editorLineSpacing:1.75,
  autosaveDelayMs:700, defaultExportFormat:'archive', exportIncludeAttachments:true, dateFormat:'system',
}

export function isWailsHost() {
  return Boolean((window as Window & { _wails?: { environment?: { OS?: string } } })._wails?.environment?.OS)
}

const browserBridge: AppBridge = {
  async getInfo() { return { name: 'Solace', version: '0.1.0', initialized: false, dataPaths: { root: 'Available in desktop app' } } },
  async loadSettings() {
    const value = localStorage.getItem('solace.settings')
    if (!value) return defaultSettings
    try {
      const parsed = JSON.parse(value) as Partial<Settings>
      const validTheme = parsed.theme === 'light' || parsed.theme === 'dark' || parsed.theme === 'system'
      return validTheme ? normalizeSettings(parsed) : defaultSettings
    } catch {
      return defaultSettings
    }
  },
  async saveSettings(settings) { localStorage.setItem('solace.settings', JSON.stringify(settings)) },
  async checkDataHealth() { return { healthy:true, checkedAt:new Date().toISOString(), message:'Browser preview does not open SQLite.' } },
}

const nativeBridge: AppBridge = {
  async getInfo() {
    return NativeService.GetInfo()
  },
  async loadSettings() {
    const value = await NativeService.LoadSettings()
    return normalizeSettings({
      theme:value.theme as Settings['theme'], sidebarCollapsed:value.sidebarCollapsed,
      editorFontSize:value.editorFontSize, editorWidth:value.editorWidth as Settings['editorWidth'],
      editorLineSpacing:value.editorLineSpacing, autosaveDelayMs:value.autosaveDelayMs,
      defaultExportFormat:value.defaultExportFormat as Settings['defaultExportFormat'],
      exportIncludeAttachments:value.exportIncludeAttachments, dateFormat:value.dateFormat as Settings['dateFormat'],
    })
  },
  async saveSettings(settings) {
    await NativeService.SaveSettings({
      theme: settings.theme as NativeTheme,
      sidebarCollapsed: settings.sidebarCollapsed,
      editorFontSize: settings.editorFontSize,
      editorWidth: settings.editorWidth as NativeEditorWidth,
      editorLineSpacing: settings.editorLineSpacing,
      autosaveDelayMs: settings.autosaveDelayMs,
      defaultExportFormat: settings.defaultExportFormat,
      exportIncludeAttachments: settings.exportIncludeAttachments,
      dateFormat: settings.dateFormat as NativeDateFormat,
    })
  },
  async checkDataHealth() { return NativeService.CheckDataHealth() },
}

export const appBridge: AppBridge = {
  getInfo: () => (isWailsHost() ? nativeBridge : browserBridge).getInfo(),
  loadSettings: () => (isWailsHost() ? nativeBridge : browserBridge).loadSettings(),
  saveSettings: (settings) => (isWailsHost() ? nativeBridge : browserBridge).saveSettings(settings),
  checkDataHealth: () => (isWailsHost() ? nativeBridge : browserBridge).checkDataHealth(),
}

function normalizeSettings(value: Partial<Settings>): Settings {
  const fontSize = Number(value.editorFontSize)
  const lineSpacing = Number(value.editorLineSpacing)
  const autosaveDelay = Number(value.autosaveDelayMs)
  return {
    ...defaultSettings,
    ...value,
    editorFontSize:fontSize >= 14 && fontSize <= 24 ? fontSize : defaultSettings.editorFontSize,
    editorWidth:['narrow','comfortable','wide'].includes(value.editorWidth ?? '') ? value.editorWidth! : defaultSettings.editorWidth,
    editorLineSpacing:lineSpacing >= 1.4 && lineSpacing <= 2.2 ? lineSpacing : defaultSettings.editorLineSpacing,
    autosaveDelayMs:autosaveDelay >= 300 && autosaveDelay <= 5000 ? autosaveDelay : defaultSettings.autosaveDelayMs,
    defaultExportFormat:['markdown','html','json','archive'].includes(value.defaultExportFormat ?? '') ? value.defaultExportFormat! : defaultSettings.defaultExportFormat,
    dateFormat:['system','long','iso'].includes(value.dateFormat ?? '') ? value.dateFormat! : defaultSettings.dateFormat,
    sidebarCollapsed:Boolean(value.sidebarCollapsed),
    exportIncludeAttachments:value.exportIncludeAttachments ?? defaultSettings.exportIncludeAttachments,
  }
}
