import { useEffect, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { ArchiveRestore, Check, DatabaseBackup, Download, FileArchive, FileCode2, FileJson2, FileText, FolderOpen, ShieldCheck, ShieldEllipsis, X } from 'lucide-react'
import type { ArchiveSummary, BackupSettings, OperationResult } from '../../../bindings/solace/internal/portable/models'
import { diaryBridge } from '../entries/diary-bridge'
import type { Entry } from '../entries/types'
import { dataBridge } from './data-bridge'
import { PrivacyPanel } from '../privacy/PrivacyPanel'
import type { ExportFormat, Settings } from '../../types'
import { friendlyError } from '../../lib/user-errors'

type Tab = 'export' | 'restore' | 'backups' | 'privacy'
type ExportScope = 'entire' | 'date' | 'selected'

const tabs: Array<{ id:Tab; label:string; icon:typeof Download }> = [
  { id:'export', label:'Export', icon:Download }, { id:'restore', label:'Restore', icon:ArchiveRestore }, { id:'backups', label:'Backups', icon:DatabaseBackup }, { id:'privacy', label:'Privacy & safety', icon:ShieldEllipsis },
]
const formats: Array<{ id:ExportFormat; label:string; note:string; icon:typeof FileText }> = [
  { id:'markdown', label:'Markdown', note:'Readable files and media', icon:FileText },
  { id:'html', label:'HTML', note:'Offline pages for a browser', icon:FileCode2 },
  { id:'json', label:'JSON', note:'Structured journal data', icon:FileJson2 },
  { id:'archive', label:'Complete archive', note:'Restorable database and files', icon:FileArchive },
]

export function DataManagementPage({ compact = false, settings, updateSettings }: {
  compact?:boolean
  settings?:Settings
  updateSettings?:(patch:Partial<Settings>)=>void
}) {
  const [tab, setTab] = useState<Tab>('export')
  const tabRefs = useRef<Record<Tab, HTMLButtonElement | null>>({ export:null, restore:null, backups:null, privacy:null })
  const selectTab = (next: Tab) => { setTab(next); tabRefs.current[next]?.focus() }
  const onTabKey = (event: KeyboardEvent, current: Tab) => {
    const index = tabs.findIndex((item) => item.id === current)
    if (event.key === 'ArrowRight') { event.preventDefault(); selectTab(tabs[(index + 1) % tabs.length].id) }
    if (event.key === 'ArrowLeft') { event.preventDefault(); selectTab(tabs[(index + tabs.length - 1) % tabs.length].id) }
  }
  return <section className={compact ? 'data-page compact' : 'data-page'} aria-label="Data management">
    {!compact && <header className="data-intro"><div><span className="eyebrow">Your journal, in your hands</span><h2>Data management</h2><p>Manage portable copies, verified recovery, local backups, and privacy safeguards.</p></div><ShieldCheck size={31}/></header>}
    <div className="data-workspace">
      <div className="data-tabs" role="tablist" aria-label="Data management sections">
        {tabs.map(({ id, label, icon:Icon }) => <button key={id} ref={(node) => { tabRefs.current[id] = node }} role="tab" aria-selected={tab === id} aria-controls={`data-panel-${id}`} tabIndex={tab === id ? 0 : -1} onKeyDown={(event) => onTabKey(event, id)} onClick={() => setTab(id)}><Icon size={16}/>{label}</button>)}
      </div>
      <ExportPanel hidden={tab !== 'export'} settings={settings} updateSettings={updateSettings}/>
      <RestorePanel hidden={tab !== 'restore'}/>
      <BackupsPanel hidden={tab !== 'backups'}/>
      <PrivacyPanel hidden={tab !== 'privacy'}/>
    </div>
  </section>
}

function ExportPanel({ hidden, settings, updateSettings }: { hidden:boolean; settings?:Settings; updateSettings?:(patch:Partial<Settings>)=>void }) {
  const [format, setFormat] = useState<ExportFormat>(settings?.defaultExportFormat ?? 'archive')
  const [scope, setScope] = useState<ExportScope>('entire')
  const [entries, setEntries] = useState<Entry[]>([])
  const [selected, setSelected] = useState<string[]>([])
  const [startDate, setStartDate] = useState(localDate())
  const [endDate, setEndDate] = useState(localDate())
  const [includeAttachments, setIncludeAttachments] = useState(settings?.exportIncludeAttachments ?? true)
  const operation = useOperation()
  useEffect(() => { if (!hidden) void diaryBridge.list({ limit:100, sort:'newest' }).then(setEntries) }, [hidden])
  const run = async () => {
    operation.clear()
    if (scope === 'selected' && !selected.length) return operation.fail('Select at least one entry to export.')
    if (scope === 'date' && (!startDate || !endDate || startDate > endDate)) return operation.fail('Choose a valid date range.')
    const destination = await dataBridge.chooseExportDestination(format)
    if (!destination) return
    await operation.run(() => dataBridge.export({ format, scope, startDate, endDate, entryIDs:selected, includeAttachments, destination }))
  }
  return <div id="data-panel-export" role="tabpanel" hidden={hidden} className="data-panel">
    <div className="data-panel-heading"><div><h3>Create a portable copy</h3><p>Choose a format based on how you want to read or restore it.</p></div><span>Step 1 of 1</span></div>
    <fieldset disabled={operation.busy}>
      <legend>Format</legend><div className="format-grid">{formats.map(({ id, label, note, icon:Icon }) => <label key={id} className={format === id ? 'format-option selected' : 'format-option'}><input type="radio" name="format" value={id} checked={format === id} onChange={() => { setFormat(id); updateSettings?.({ defaultExportFormat:id }) }}/><Icon size={20}/><span><strong>{label}</strong><small>{note}</small></span></label>)}</div>
      <div className="data-form-grid">
        <label>Entries<select value={scope} onChange={(event) => setScope(event.target.value as ExportScope)}><option value="entire">Entire journal</option><option value="date">A date range</option><option value="selected">Selected entries</option></select></label>
        {scope === 'date' && <><label>From<input aria-label="Export from date" type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)}/></label><label>To<input aria-label="Export to date" type="date" value={endDate} onChange={(event) => setEndDate(event.target.value)}/></label></>}
        <label className="data-check"><input type="checkbox" checked={includeAttachments} onChange={(event) => { setIncludeAttachments(event.target.checked); updateSettings?.({ exportIncludeAttachments:event.target.checked }) }}/><span><strong>Include attachments</strong><small>Copy original photos, audio, video, PDFs, and files.</small></span></label>
      </div>
      {scope === 'selected' && <div className="entry-picker" aria-label="Entries to export">{entries.length ? entries.map((entry) => <label key={entry.id}><input type="checkbox" checked={selected.includes(entry.id)} onChange={(event) => setSelected((current) => event.target.checked ? [...current, entry.id] : current.filter((id) => id !== entry.id))}/><span><strong>{entry.title || 'Untitled entry'}</strong><small>{formatDate(entry.startDate)}</small></span></label>) : <p>No entries are available yet.</p>}</div>}
    </fieldset>
    <OperationFooter operation={operation} action="Choose destination & export" onAction={() => void run()} nativeOnly/>
  </div>
}

function RestorePanel({ hidden }: { hidden:boolean }) {
  const [path, setPath] = useState('')
  const [summary, setSummary] = useState<ArchiveSummary | null>(null)
  const [replace, setReplace] = useState(false)
  const operation = useOperation()
  const choose = async () => {
    operation.clear(); setSummary(null)
    const next = await dataBridge.chooseArchive(); if (!next) return
    setPath(next)
    try { setSummary(await dataBridge.inspectArchive(next)) } catch (reason) { operation.fail(friendlyError(reason, 'That archive couldn’t be verified. It was not opened and your journal is unchanged.', 'inspect archive')) }
  }
  const restore = async () => {
    if (!path || !summary?.valid) return
    if (replace && !window.confirm('Replace the current journal with this archive? Solace will create a recovery backup first.')) return
    await operation.run(() => dataBridge.restore(path, replace))
  }
  return <div id="data-panel-restore" role="tabpanel" hidden={hidden} className="data-panel">
    <div className="data-panel-heading"><div><h3>Restore a complete archive</h3><p>Solace verifies its version, paths, and file checksums before changing anything.</p></div><ShieldCheck size={23}/></div>
    <div className="restore-picker"><button className="secondary-action" onClick={() => void choose()} disabled={operation.busy || !dataBridge.isNative()}><FolderOpen size={16}/>Choose archive</button><span title={path}>{path || 'No archive selected'}</span></div>
    {summary && <div className="archive-summary"><div><Check size={19}/><span><strong>Archive verified</strong><small>Created {formatTimestamp(summary.createdAt)}</small></span></div><dl><div><dt>Entries</dt><dd>{summary.entryCount}</dd></div><div><dt>Attachments</dt><dd>{summary.attachmentCount}</dd></div><div><dt>Missing files</dt><dd>{summary.missingAttachments?.length ?? 0}</dd></div></dl></div>}
    <label className="data-check restore-check"><input type="checkbox" checked={replace} onChange={(event) => setReplace(event.target.checked)}/><span><strong>Replace the current journal</strong><small>Required when Solace already contains entries. A recovery archive is created automatically.</small></span></label>
    <OperationFooter operation={operation} action="Restore verified archive" onAction={() => void restore()} disabled={!summary?.valid} nativeOnly/>
  </div>
}

function BackupsPanel({ hidden }: { hidden:boolean }) {
  const [settings, setSettings] = useState<BackupSettings>({ schedule:'manual', destination:'', retention:7, includeAttachments:true, lastBackupAt:'' })
  const [saved, setSaved] = useState('')
  const [loadError, setLoadError] = useState('')
  const operation = useOperation()
  useEffect(() => { if (!hidden) void dataBridge.loadBackupSettings().then(setSettings).catch((reason) => setLoadError(friendlyError(reason, 'Backup preferences couldn’t be loaded. Please try again.', 'load backup settings'))) }, [hidden])
  const choose = async () => { const destination = await dataBridge.chooseBackupDirectory(); if (destination) setSettings((current) => ({ ...current, destination })) }
  const save = async () => {
    operation.clear(); setSaved('')
    try { await dataBridge.saveBackupSettings(settings); setSaved('Backup preferences saved.') }
    catch (reason) { operation.fail(friendlyError(reason, 'Backup preferences couldn’t be saved. Please try again.', 'save backup settings')) }
  }
  const backup = async () => {
    operation.clear(); setSaved('')
    try { await dataBridge.saveBackupSettings(settings) } catch (reason) { operation.fail(friendlyError(reason, 'Backup preferences couldn’t be saved. Please try again.', 'save backup before run')); return }
    await operation.run(() => dataBridge.createBackup())
  }
  return <div id="data-panel-backups" role="tabpanel" hidden={hidden} className="data-panel">
    <div className="data-panel-heading"><div><h3>Keep a quiet safety net</h3><p>Backups are local, restorable archives. Scheduled backups run when Solace starts.</p></div>{settings.lastBackupAt && <span>Last: {formatTimestamp(settings.lastBackupAt)}</span>}</div>
    <fieldset disabled={operation.busy} className="backup-form">
      <label>Schedule<select value={settings.schedule} onChange={(event) => setSettings({ ...settings, schedule:event.target.value })}><option value="manual">Manual only</option><option value="daily">Daily</option><option value="weekly">Weekly</option></select></label>
      <label>Keep<input type="number" min="1" max="100" value={settings.retention} onChange={(event) => setSettings({ ...settings, retention:Number(event.target.value) })}/><small>most recent backups</small></label>
      <label className="backup-destination">Backup folder<span><input value={settings.destination} readOnly aria-label="Backup folder"/><button type="button" onClick={() => void choose()} disabled={!dataBridge.isNative()}><FolderOpen size={15}/>Choose</button></span></label>
      <label className="data-check"><input type="checkbox" checked={settings.includeAttachments} onChange={(event) => setSettings({ ...settings, includeAttachments:event.target.checked })}/><span><strong>Include attachments</strong><small>Recommended for complete recovery.</small></span></label>
    </fieldset>
    {saved && <p className="operation-success" role="status"><Check size={15}/>{saved}</p>}
    <div className="backup-actions"><button className="secondary-action" onClick={() => void save()} disabled={operation.busy}>Save preferences</button><button className="primary-action" onClick={() => void backup()} disabled={operation.busy || !dataBridge.isNative()}><DatabaseBackup size={16}/>Back up now</button></div>
    {loadError && <p className="operation-error" role="alert">{loadError}</p>}<OperationMessage operation={operation}/>{!dataBridge.isNative() && <PreviewNotice/>}
  </div>
}

function useOperation() {
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [result, setResult] = useState<OperationResult | null>(null)
  return { busy, error, result, clear:() => { setError(''); setResult(null) }, fail:setError, run:async (work:() => Promise<OperationResult>) => { setBusy(true); setError(''); setResult(null); try { setResult(await work()) } catch (reason) { setError(friendlyError(reason, 'Solace couldn’t finish that file operation. Your journal was not changed.', 'journal file operation')) } finally { setBusy(false) } } }
}
type Operation = ReturnType<typeof useOperation>
function OperationFooter({ operation, action, onAction, disabled, nativeOnly }: { operation:Operation; action:string; onAction:()=>void; disabled?:boolean; nativeOnly?:boolean }) {
  return <>{operation.busy && <div className="operation-progress" role="progressbar" aria-label="Operation progress" aria-valuetext="Working"><span/></div>}<div className="operation-footer"><span>{operation.busy ? 'Working safely…' : 'Exports never alter your journal.'}</span>{operation.busy && <button className="secondary-action" onClick={() => void dataBridge.cancel()}><X size={15}/>Cancel</button>}<button className="primary-action" onClick={onAction} disabled={operation.busy || disabled || (nativeOnly && !dataBridge.isNative())}><Download size={16}/>{action}</button></div><OperationMessage operation={operation}/>{nativeOnly && !dataBridge.isNative() && <PreviewNotice/>}</>
}
function OperationMessage({ operation }: { operation:Operation }) {
  if (operation.error) return <p className="operation-error" role="alert">{operation.error}</p>
  if (operation.result) return <div className="operation-result" role="status"><Check size={17}/><span><strong>{operation.result.operationMessage}</strong><small>{operation.result.outputPath}{operation.result.recoveryBackup ? ` · Recovery: ${operation.result.recoveryBackup}` : ''}</small></span></div>
  return null
}
function PreviewNotice() { return <p className="preview-notice">The controls are shown for preview. File operations become available in the native Solace desktop window.</p> }
function localDate() { const date=new Date(); return `${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,'0')}-${String(date.getDate()).padStart(2,'0')}` }
function formatDate(value:string) { if (!value) return 'No date'; return new Intl.DateTimeFormat(undefined,{ dateStyle:'medium' }).format(new Date(`${value}T12:00:00`)) }
function formatTimestamp(value:string) { if (!value) return 'Never'; const date=new Date(value); return Number.isNaN(date.getTime()) ? value : new Intl.DateTimeFormat(undefined,{ dateStyle:'medium', timeStyle:'short' }).format(date) }
