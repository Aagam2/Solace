import { useCallback, useEffect, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { BookOpen, Copy, Heart, History, Maximize2, Minimize2, Pin, RotateCcw, Save, Trash2, X } from 'lucide-react'
import { AttachmentGallery } from './AttachmentGallery'
import { RichTextEditor } from './RichTextEditor'
import { attachmentBridge } from './attachment-bridge'
import { diaryBridge, toInput } from './diary-bridge'
import type { Attachment, Entry, EntryInput, ListOptions, Revision } from './types'
import type { DateFormat } from '../../types'
import { friendlyError } from '../../lib/user-errors'
import { useConfirmation } from '../../components/ConfirmationDialog'

export type EntryView = 'today' | 'all' | 'favorites' | 'trash'
type EditorTab = 'writing' | 'details' | 'history'

const emptyInput = (date: string): EntryInput => ({
  title:'', contentJSON:'{"type":"doc","content":[{"type":"paragraph"}]}', contentText:'', startDate:date, endDate:date, mood:'', favorite:false,
  pinned:false, draft:true, locationText:'', displayOrder:null, tags:[],
})

export function EntryWorkspace({ view, preferredEntryID, newEntrySignal, onNewEntryHandled, autosaveDelayMs, dateFormat }: {
  view:EntryView; preferredEntryID?:string; newEntrySignal:number; onNewEntryHandled:()=>void; autosaveDelayMs:number; dateFormat:DateFormat
}) {
  const confirmAction = useConfirmation()
  const [entries, setEntries] = useState<Entry[]>([])
  const [selected, setSelected] = useState<Entry | null>(null)
  const [form, setForm] = useState<EntryInput>(() => emptyInput(today()))
  const [dirty, setDirty] = useState(false)
  const [sort, setSort] = useState<ListOptions['sort']>(view === 'favorites' ? 'favorites' : 'newest')
  const [status, setStatus] = useState<'saved' | 'unsaved' | 'saving' | 'error'>('saved')
  const [error, setError] = useState('')
  const [attachmentRevision, setAttachmentRevision] = useState(0)
  const [insertImage, setInsertImage] = useState<{ key: number; url: string; alt: string } | null>(null)
  const [readingMode, setReadingMode] = useState(false)
  const [focusMode, setFocusMode] = useState(false)
  const [editorTab, setEditorTab] = useState<EditorTab>('writing')
  const [revisions, setRevisions] = useState<Revision[]>([])
  const selectedID = selected?.id
  const initialSignal = useRef(0)
  const saveTimer = useRef<number | null>(null)
  const saveInFlight = useRef<Promise<Entry | null> | null>(null)
  const editRevision = useRef(0)
  const pendingSave = useRef<{ entry: Entry | null; input: EntryInput; dirty: boolean }>({ entry:null, input:form, dirty:false })
  pendingSave.current = { entry:selected, input:form, dirty }

  const options = useCallback((): ListOptions => {
    if (view === 'today') return { startDate:today(), endDate:today() }
    if (view === 'favorites') return { favoritesOnly:true, sort:'favorites' }
    if (view === 'trash') return { includeTrash:true }
    return { sort }
  }, [sort, view])

  const reload = useCallback(async (preferredID?: string) => {
    try {
      const next = await diaryBridge.list(options())
      setEntries(next)
      const target = next.find((entry) => entry.id === preferredID) ?? next[0] ?? null
      setSelected(target)
      setForm(target ? toInput(target) : emptyInput(today()))
      setEditorTab('writing')
      setDirty(false); setStatus('saved'); setError('')
    } catch (reason) { setError(friendlyError(reason, 'Your entries couldn’t be loaded. Please try again.', 'load entries')); setStatus('error') }
  }, [options])

  useEffect(() => { void reload(preferredEntryID) }, [preferredEntryID, reload])
  useEffect(() => {
    if (editorTab === 'history' && selectedID) void diaryBridge.listRevisions(selectedID).then(setRevisions).catch((reason) => setError(friendlyError(reason, 'Saved versions couldn’t be loaded. Please try again.', 'entry history')))
  }, [editorTab, selectedID])

  const createEntry = useCallback(async () => {
    try {
      const entry = await diaryBridge.create(emptyInput(today()))
      await reload(entry.id)
    } catch (reason) { setError(friendlyError(reason, 'A new entry couldn’t be created. Please try again.', 'create entry')); setStatus('error') }
  }, [reload])

  useEffect(() => {
    if (newEntrySignal > 0 && newEntrySignal !== initialSignal.current) {
      initialSignal.current = newEntrySignal
      void createEntry().finally(onNewEntryHandled)
    }
  }, [createEntry, newEntrySignal, onNewEntryHandled])

  const save = useCallback((): Promise<Entry | null> => {
    if (saveInFlight.current) return saveInFlight.current
    if (saveTimer.current !== null) { window.clearTimeout(saveTimer.current); saveTimer.current = null }
    if (!selected || !dirty || view === 'trash') return Promise.resolve(selected)
    const revision = editRevision.current
    setStatus('saving'); setError('')
    const operation = (async () => {
      try {
        const updated = await diaryBridge.update(selected.id, form, selected.version)
        setSelected(updated); setEntries((current) => current.map((entry) => entry.id === updated.id ? updated : entry))
        if (editRevision.current === revision) { setDirty(false); setStatus('saved') }
        return updated
      } catch (reason) { setError(friendlyError(reason, 'Your latest changes couldn’t be saved. They remain on this screen—please try again.', 'save entry')); setStatus('error'); return null }
    })()
    saveInFlight.current = operation
    void operation.finally(() => { if (saveInFlight.current === operation) saveInFlight.current = null })
    return operation
  }, [dirty, form, selected, view])

  useEffect(() => {
    if (!dirty || !selected) return
    saveTimer.current = window.setTimeout(() => { saveTimer.current = null; void save() }, autosaveDelayMs)
    return () => { if (saveTimer.current !== null) { window.clearTimeout(saveTimer.current); saveTimer.current = null } }
  }, [autosaveDelayMs, dirty, save, selected])

  useEffect(() => {
    const saveEntry = () => { void save() }
    const toggleReading = () => {
      if (!selectedID || view === 'trash') return
      setEditorTab('writing'); setFocusMode(false); setReadingMode((current) => !current)
    }
    const escapeModes = (event:globalThis.KeyboardEvent) => {
      if (event.key !== 'Escape') return
      setReadingMode(false); setFocusMode(false)
    }
    window.addEventListener('solace:save-entry',saveEntry)
    window.addEventListener('solace:toggle-reading',toggleReading)
    window.addEventListener('keydown',escapeModes)
    return () => {
      window.removeEventListener('solace:save-entry',saveEntry)
      window.removeEventListener('solace:toggle-reading',toggleReading)
      window.removeEventListener('keydown',escapeModes)
    }
  }, [save, selectedID, view])

  useEffect(() => () => {
    const pending = pendingSave.current
    if (pending.entry && pending.dirty && view !== 'trash') {
      void diaryBridge.update(pending.entry.id, pending.input, pending.entry.version)
    }
  }, [view])

  const edit = <K extends keyof EntryInput>(key: K, value: EntryInput[K]) => {
    editRevision.current++
    setForm((current) => ({ ...current, [key]: value })); setDirty(true); setStatus('unsaved')
  }

  const editContent = useCallback((contentJSON: string, contentText: string) => {
    editRevision.current++
    setForm((current) => ({ ...current, contentJSON, contentText })); setDirty(true); setStatus('unsaved')
  }, [])

  const insertImportedImages = useCallback((attachments: Attachment[]) => {
    const image = attachments.at(-1)
    if (image) setInsertImage({ key:Date.now(), url:image.url, alt:image.caption || image.originalFilename })
  }, [])

  const handleEditorFiles = useCallback(async (files: File[]) => {
    if (!selectedID || view === 'trash') return
    try {
      const imported = await attachmentBridge.importFiles(selectedID, files)
      setAttachmentRevision((current) => current + 1)
      insertImportedImages(imported.filter((item) => item.mimeType.startsWith('image/')))
    } catch (reason) { setError(friendlyError(reason, 'Those files couldn’t be attached. Check that they are available, then try again.', 'attach editor files')) }
  }, [insertImportedImages, selectedID, view])

  const select = async (entry: Entry) => {
    if (dirty && await save() === null) return
    setSelected(entry); setForm(toInput(entry)); setEditorTab('writing'); setDirty(false); setStatus('saved'); setError('')
  }

  const trash = async () => { if (selected) { try { await diaryBridge.trash(selected.id); await reload() } catch (reason) { setError(friendlyError(reason, 'This entry couldn’t be moved to Trash. Please try again.', 'trash entry')) } } }
  const restore = async () => { if (selected) { try { await diaryBridge.restore(selected.id); await reload() } catch (reason) { setError(friendlyError(reason, 'This entry couldn’t be restored. Please try again.', 'restore entry')) } } }
  const remove = async () => {
    if (selected && await confirmAction({ title:'Delete this entry forever?', message:'This permanently removes the entry and its attachments. This cannot be undone.', confirmLabel:'Delete forever', tone:'danger' })) {
      try { await attachmentBridge.permanentlyDeleteEntry(selected.id); await reload() } catch (reason) { setError(friendlyError(reason, 'This entry couldn’t be permanently deleted. Nothing was changed.', 'delete entry')) }
    }
  }
  const duplicate = async () => { if (selected) { try { const entry = await diaryBridge.duplicate(selected.id); await reload(entry.id) } catch (reason) { setError(friendlyError(reason, 'This entry couldn’t be duplicated. Please try again.', 'duplicate entry')) } } }
  const restoreRevision = async (revision: Revision) => {
    if (!selected || dirty || !await confirmAction({ title:'Restore this version?', message:`Restore the version saved ${formatTimestamp(revision.savedAt)}? Your current saved version will remain in recovery history.`, confirmLabel:'Restore version' })) return
    try { const restored = await diaryBridge.restoreRevision(selected.id, revision.id, selected.version); await reload(restored.id) }
    catch (reason) { setError(friendlyError(reason, 'That saved version couldn’t be restored. Your current entry is unchanged.', 'restore entry version')); setStatus('error') }
  }
  const deleteRevision = async (revision:Revision) => {
    if (!selected || !await confirmAction({ title:'Delete this recovery version?', message:`Delete the version saved ${formatTimestamp(revision.savedAt)}? Your current entry will not change.`, confirmLabel:'Delete version', tone:'danger' })) return
    try { await diaryBridge.deleteRevision(selected.id, revision.id); setRevisions((current) => current.filter((item) => item.id !== revision.id)) }
    catch (reason) { setError(friendlyError(reason, 'That saved version couldn’t be deleted. Please try again.', 'delete entry version')) }
  }
  const clearRevisions = async () => {
    if (!selected || !await confirmAction({ title:'Clear recovery history?', message:'Delete every saved version for this entry? Your current entry will not change, but this history cannot be recovered.', confirmLabel:'Clear history', tone:'danger' })) return
    try { await diaryBridge.clearRevisions(selected.id); setRevisions([]) }
    catch (reason) { setError(friendlyError(reason, 'Saved versions couldn’t be cleared. Please try again.', 'clear entry history')) }
  }
  const changeSort = async (next: ListOptions['sort']) => {
    if (dirty && await save() === null) return
    setSort(next)
  }

  const wordCount = countWords(form.contentText)
  const workspaceClass = focusMode ? 'entry-workspace focus-mode' : 'entry-workspace'
  const editorClass = readingMode ? 'entry-editor reading-mode' : focusMode ? 'entry-editor focus-writing' : 'entry-editor'
  return <section className={workspaceClass} aria-label="Journal entries">
    <aside className="entry-list">
      <div className="entry-list-header"><span>{viewLabel(view)}</span><strong>{entries.length}</strong></div>
      {(view === 'all' || view === 'today') && <label className="entry-sort">Order
        <select value={sort} onChange={(event) => void changeSort(event.target.value as ListOptions['sort'])}>
          <option value="newest">Newest first</option><option value="oldest">Oldest first</option>
          <option value="updated">Recently updated</option><option value="favorites">Favorites first</option>
        </select>
      </label>}
      {entries.length === 0 && <div className="entry-list-empty">No entries here yet.</div>}
      {entries.map((entry) => <button key={entry.id} className={selected?.id === entry.id ? 'entry-card selected' : 'entry-card'} onClick={() => void select(entry)}>
        <span className="entry-card-date">{dateRange(entry,dateFormat)}</span>
        <strong>{entry.title || 'Untitled entry'}</strong>
        <p>{entry.contentText || 'A blank page'}</p>
        <span className="entry-card-meta">{entry.pinned ? 'Pinned · ' : ''}{entry.draft ? 'Draft' : 'Published'}{entry.favorite ? ' · Favorite' : ''}{entry.mood ? ` · ${entry.mood}` : ''}{entry.tags.length ? ` · ${entry.tags.join(', ')}` : ''}</span>
      </button>)}
    </aside>
    <div className={editorClass}>
      {!selected ? <div className="editor-empty"><h3>Your next page starts here.</h3><button onClick={() => void createEntry()}>Create an entry</button></div> : <>
        <div className="editor-toolbar">
          <div className="editor-status"><div className={`save-status ${status}`}>{status === 'saving' ? 'Saving…' : status === 'unsaved' ? 'Unsaved changes' : status === 'error' ? 'Save failed' : 'Saved'}</div><span>{wordCount} {wordCount === 1 ? 'word' : 'words'} · {form.contentText.length} characters</span></div>
          <div className="editor-actions">
            {readingMode ? <button onClick={() => setReadingMode(false)}><X size={15}/>Exit reading</button> : <>
            {view === 'trash' ? <><button onClick={() => void restore()}><RotateCcw size={15}/>Restore</button><button className="danger" onClick={() => void remove()}><Trash2 size={15}/>Delete forever</button></> : <>
              <button onClick={() => edit('favorite', !form.favorite)} aria-label="Toggle favorite"><Heart size={15} fill={form.favorite ? 'currentColor' : 'none'}/></button>
              <button onClick={() => edit('pinned', !form.pinned)} aria-label="Toggle pinned"><Pin size={15} fill={form.pinned ? 'currentColor' : 'none'}/></button>
              <button aria-pressed={focusMode} onClick={() => { setEditorTab('writing'); setFocusMode((current) => !current) }} title="Distraction-free writing"><>{focusMode ? <Minimize2 size={15}/> : <Maximize2 size={15}/>}</>{focusMode ? 'Exit focus' : 'Focus'}</button>
              <button onClick={() => { setEditorTab('writing'); setFocusMode(false); setReadingMode(true) }} aria-keyshortcuts="Control+Shift+R Meta+Shift+R"><BookOpen size={15}/>Read</button>
              <button onClick={() => void duplicate()}><Copy size={15}/>Duplicate</button>
              <button onClick={() => void save()} aria-keyshortcuts="Control+S Meta+S"><Save size={15}/>Save</button>
              <button className="danger" onClick={() => void trash()}><Trash2 size={15}/>Trash</button>
            </>}</>}
          </div>
        </div>
        {error && <div className="editor-error" role="alert">{error}</div>}
        {!readingMode && <div className="editor-section-tabs" role="tablist" aria-label="Entry sections">
          <button id="editor-tab-writing" type="button" role="tab" aria-selected={editorTab === 'writing'} aria-controls="editor-panel-writing" tabIndex={editorTab === 'writing' ? 0 : -1} onClick={() => setEditorTab('writing')} onKeyDown={(event) => moveEditorTab(event, 'writing', setEditorTab)}>Writing</button>
          <button id="editor-tab-details" type="button" role="tab" aria-selected={editorTab === 'details'} aria-controls="editor-panel-details" tabIndex={editorTab === 'details' ? 0 : -1} onClick={() => setEditorTab('details')} onKeyDown={(event) => moveEditorTab(event, 'details', setEditorTab)}>Details &amp; media</button>
          <button id="editor-tab-history" type="button" role="tab" aria-selected={editorTab === 'history'} aria-controls="editor-panel-history" tabIndex={editorTab === 'history' ? 0 : -1} onClick={() => setEditorTab('history')} onKeyDown={(event) => moveEditorTab(event, 'history', setEditorTab)}>History</button>
        </div>}
        <fieldset disabled={view === 'trash' || readingMode}>
          <input className="title-input" value={form.title} onChange={(event) => edit('title', event.target.value)} placeholder="Untitled entry" aria-label="Entry title" />
          <div id="editor-panel-writing" role="tabpanel" aria-labelledby="editor-tab-writing" hidden={!readingMode && editorTab !== 'writing'}>
            <RichTextEditor contentJSON={form.contentJSON} readOnly={view === 'trash' || readingMode} insertImage={insertImage} onChange={editContent} onFiles={handleEditorFiles} />
          </div>
          <div id="editor-panel-details" className="editor-details-panel" role="tabpanel" aria-labelledby="editor-tab-details" hidden={readingMode || editorTab !== 'details'}>
            <div className="entry-fields">
              <label>Start date<input type="date" value={form.startDate} onChange={(event) => edit('startDate', event.target.value)} /></label>
              <label>End date<input type="date" min={form.startDate} value={form.endDate} onChange={(event) => edit('endDate', event.target.value)} /></label>
              <label>Mood<select value={form.mood} onChange={(event) => edit('mood', event.target.value)}><option value="">Not set</option><option>Very Happy</option><option>Happy</option><option>Neutral</option><option>Sad</option><option>Very Sad</option></select></label>
              <label>Location<input value={form.locationText} onChange={(event) => edit('locationText', event.target.value)} placeholder="Optional" /></label>
            </div>
            <div className="entry-footer-fields">
              <label>Tags<input value={form.tags.join(', ')} onChange={(event) => edit('tags', event.target.value.split(',').map((tag) => tag.trim()).filter(Boolean))} placeholder="Personal, Travel" /></label>
              <label className="check-field"><input type="checkbox" checked={form.draft} onChange={(event) => edit('draft', event.target.checked)} />Keep as draft</label>
            </div>
            <AttachmentGallery key={`${selected.id}-${attachmentRevision}`} entryID={selected.id} readOnly={view === 'trash' || readingMode} onImagesAdded={insertImportedImages} />
          </div>
          <div id="editor-panel-history" className="revision-panel" role="tabpanel" aria-labelledby="editor-tab-history" hidden={readingMode || editorTab !== 'history'}>
            <div className="revision-heading"><div><History size={17}/><span><strong>Recovery history</strong><small>Up to 5 meaningful versions, spaced at least 10 minutes apart.</small></span></div><div className="revision-heading-actions">{dirty && <em>Save current changes before restoring.</em>}{revisions.length > 0 && <button type="button" className="revision-clear" onClick={() => void clearRevisions()}><Trash2 size={13}/>Clear history</button>}</div></div>
            {revisions.length === 0 ? <div className="revision-empty">Recovery versions will appear here after meaningful changes to your writing or entry details.</div> : <div className="revision-list">{revisions.map((revision) => <article key={revision.id}><div><time>{formatTimestamp(revision.savedAt)}</time><strong>{revision.title || 'Untitled entry'}</strong><p>{revision.contentText || 'A blank page'}</p><small>Version {revision.sourceVersion}{revision.tags.length ? ` · ${revision.tags.join(', ')}` : ''}</small></div><div className="revision-actions"><button type="button" disabled={dirty || view === 'trash'} onClick={() => void restoreRevision(revision)}><RotateCcw size={14}/>Restore</button><button type="button" className="revision-delete" onClick={() => void deleteRevision(revision)} aria-label={`Delete version saved ${formatTimestamp(revision.savedAt)}`} title="Delete this saved version"><Trash2 size={14}/></button></div></article>)}</div>}
          </div>
        </fieldset>
      </>}
    </div>
  </section>
}

function today() {
  const date = new Date()
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
function viewLabel(view: EntryView) { return view === 'today' ? 'Today' : view === 'favorites' ? 'Favorites' : view === 'trash' ? 'Trash' : 'All entries' }
function displayDate(value: string, format:DateFormat) {
  if (format === 'iso') return value
  const [year, month, day] = value.split('-').map(Number)
  return new Intl.DateTimeFormat(undefined, format === 'long' ? { month:'long', day:'numeric', year:'numeric' } : { month:'short', day:'numeric', year:'numeric' }).format(new Date(year, month - 1, day))
}
function dateRange(entry: Entry, format:DateFormat) { return entry.startDate === entry.endDate ? displayDate(entry.startDate,format) : `${displayDate(entry.startDate,format)} – ${displayDate(entry.endDate,format)}` }
function formatTimestamp(value:string) { return new Intl.DateTimeFormat(undefined,{ dateStyle:'medium', timeStyle:'short' }).format(new Date(value)) }
function countWords(value:string) { const trimmed=value.trim(); return trimmed ? trimmed.split(/\s+/u).length : 0 }
function moveEditorTab(event: KeyboardEvent<HTMLButtonElement>, current: EditorTab, select: (tab: EditorTab) => void) {
  if (!['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return
  event.preventDefault()
  const tabs: EditorTab[] = ['writing','details','history']
  const currentIndex = tabs.indexOf(current)
  const nextIndex = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (currentIndex + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length
  const next = tabs[nextIndex]
  select(next)
  window.requestAnimationFrame(() => document.getElementById(`editor-tab-${next}`)?.focus())
}
