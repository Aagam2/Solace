export type Theme = 'light' | 'dark' | 'system'
export type EditorWidth = 'narrow' | 'comfortable' | 'wide'
export type DateFormat = 'system' | 'long' | 'iso'
export type ExportFormat = 'markdown' | 'html' | 'json' | 'archive'
export interface Settings {
  theme: Theme
  sidebarCollapsed: boolean
  editorFontSize: number
  editorWidth: EditorWidth
  editorLineSpacing: number
  autosaveDelayMs: number
  defaultExportFormat: ExportFormat
  exportIncludeAttachments: boolean
  dateFormat: DateFormat
}
export interface DataPaths {
  root: string
  database?: string
  media?: string
  backups?: string
  logs?: string
}
export interface AppInfo { name: string; version: string; initialized: boolean; dataPaths: DataPaths }
