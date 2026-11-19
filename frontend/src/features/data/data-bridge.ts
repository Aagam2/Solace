import { Dialogs } from '@wailsio/runtime'
import * as NativePortable from '../../../bindings/solace/internal/portable/service'
import type { ArchiveSummary, BackupSettings, ExportRequest, OperationResult } from '../../../bindings/solace/internal/portable/models'
import { isWailsHost } from '../../lib/bridge'

const previewSettingsKey = 'solace.backup-settings'
const defaultPreviewSettings: BackupSettings = {
  schedule:'manual', destination:'Solace backups', retention:7, includeAttachments:true, lastBackupAt:'',
}

export const dataBridge = {
  isNative: isWailsHost,
  async chooseExportDestination(format: ExportRequest['format']) {
    if (!isWailsHost()) return ''
    const extension = format === 'json' ? 'json' : 'zip'
    return Dialogs.SaveFile({
      Title:'Export from Solace', ButtonText:'Export', CanCreateDirectories:true,
      Filename:`solace-export-${localDate()}.${extension}`,
      Filters:[{ DisplayName:extension === 'json' ? 'JSON export' : 'ZIP archive', Pattern:`*.${extension}` }],
    })
  },
  async export(request: ExportRequest): Promise<OperationResult> {
    if (!isWailsHost()) throw new Error('Exports are available in the Solace desktop app.')
    return NativePortable.Export(request)
  },
  async chooseArchive() {
    if (!isWailsHost()) return ''
    return Dialogs.OpenFile({
      Title:'Choose a Solace archive', ButtonText:'Inspect archive', CanChooseFiles:true, CanChooseDirectories:false,
      Filters:[{ DisplayName:'Solace archive', Pattern:'*.zip' }],
    })
  },
  async inspectArchive(path: string): Promise<ArchiveSummary> {
    if (!isWailsHost()) throw new Error('Archive inspection is available in the Solace desktop app.')
    return NativePortable.InspectArchive(path)
  },
  async restore(path: string, replaceExisting: boolean): Promise<OperationResult> {
    if (!isWailsHost()) throw new Error('Restore is available in the Solace desktop app.')
    return NativePortable.RestoreArchive(path, replaceExisting)
  },
  async loadBackupSettings(): Promise<BackupSettings> {
    if (isWailsHost()) return NativePortable.LoadBackupSettings()
    try { return { ...defaultPreviewSettings, ...JSON.parse(localStorage.getItem(previewSettingsKey) ?? '{}') } }
    catch { return defaultPreviewSettings }
  },
  async chooseBackupDirectory() {
    if (!isWailsHost()) return ''
    return Dialogs.OpenFile({
      Title:'Choose a backup folder', ButtonText:'Use this folder', CanChooseFiles:false, CanChooseDirectories:true, CanCreateDirectories:true,
    })
  },
  async saveBackupSettings(settings: BackupSettings) {
    if (isWailsHost()) return NativePortable.SaveBackupSettings(settings)
    localStorage.setItem(previewSettingsKey, JSON.stringify(settings))
  },
  async createBackup(): Promise<OperationResult> {
    if (!isWailsHost()) throw new Error('Backups are available in the Solace desktop app.')
    return NativePortable.CreateBackup()
  },
  async cancel() { if (isWailsHost()) await NativePortable.CancelCurrentOperation() },
}

function localDate() {
  const date = new Date()
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}
