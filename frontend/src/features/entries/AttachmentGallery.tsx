import { useCallback, useEffect, useRef, useState } from 'react'
import { ChevronLeft, ChevronRight, File, FileText, FolderOpen, ImagePlus, Maximize2, Paperclip, Play, Trash2, Upload } from 'lucide-react'
import { isWailsHost } from '../../lib/bridge'
import { attachmentBridge } from './attachment-bridge'
import { friendlyError } from '../../lib/user-errors'
import type { Attachment } from './types'

export function AttachmentGallery({ entryID, readOnly, onImagesAdded }: {
  entryID: string
  readOnly: boolean
  onImagesAdded: (attachments: Attachment[]) => void
}) {
  const [attachments, setAttachments] = useState<Attachment[]>([])
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [enlarged, setEnlarged] = useState<Attachment | null>(null)
  const [playbackErrors, setPlaybackErrors] = useState<Record<string, boolean>>({})
  const input = useRef<HTMLInputElement>(null)

  const reload = useCallback(async () => setAttachments(await attachmentBridge.list(entryID)), [entryID])
  useEffect(() => { void reload() }, [reload])

  const finishImport = useCallback(async (operation: Promise<Attachment[]>) => {
    setStatus('Importing…'); setError('')
    try {
      const imported = await operation
      await reload(); onImagesAdded(imported.filter((item) => item.mimeType.startsWith('image/')))
      setStatus(imported.length ? `${imported.length} attachment${imported.length === 1 ? '' : 's'} imported` : '')
    } catch (reason) { setError(friendlyError(reason, 'Those files couldn’t be attached. Check that they are available, then try again.', 'attach files')); setStatus('') }
  }, [onImagesAdded, reload])

  useEffect(() => attachmentBridge.onNativeDrop((paths) => { if (!readOnly) void finishImport(attachmentBridge.importPaths(entryID, paths)) }), [entryID, finishImport, readOnly])

  const choose = async () => {
    if (isWailsHost()) await finishImport(attachmentBridge.pick(entryID))
    else input.current?.click()
  }
  const browserFiles = (files: FileList | null) => {
    if (!readOnly && files?.length) void finishImport(attachmentBridge.importFiles(entryID, Array.from(files)))
  }
  const move = async (index: number, offset: number) => {
    const target = index + offset
    if (target < 0 || target >= attachments.length) return
    const next = [...attachments]; [next[index], next[target]] = [next[target], next[index]]
    setAttachments(next)
    try { await attachmentBridge.reorder(entryID, next.map((item) => item.id)) } catch (reason) { setError(friendlyError(reason, 'The attachment order couldn’t be saved. Please try again.', 'reorder attachments')); await reload() }
  }
  const remove = async (attachment: Attachment) => {
    if (!window.confirm(`Remove “${attachment.originalFilename}” from this entry?`)) return
    try { await attachmentBridge.remove(attachment.id); await reload() } catch (reason) { setError(friendlyError(reason, 'That attachment couldn’t be removed. Please try again.', 'remove attachment')) }
  }
  const caption = async (attachment: Attachment, value: string) => {
    setAttachments((current) => current.map((item) => item.id === attachment.id ? { ...item, caption:value } : item))
    try { await attachmentBridge.caption(attachment.id, value) } catch (reason) { setError(friendlyError(reason, 'The caption couldn’t be saved. Please try again.', 'save attachment caption')) }
  }

  return <section
    id="entry-attachment-drop-zone" className="attachment-section" aria-label="Attachment drop zone" data-file-drop-target={readOnly ? undefined : ''}
    onDragOver={(event) => event.preventDefault()}
    onDrop={(event) => { event.preventDefault(); browserFiles(event.dataTransfer.files) }}
  >
    <div className="attachment-heading">
      <div><Paperclip size={16}/><strong>Attachments</strong><span>{attachments.length}</span></div>
      {!readOnly && <button type="button" onClick={() => void choose()}><Upload size={15}/>Attach files</button>}
      <input ref={input} hidden type="file" multiple onChange={(event) => { browserFiles(event.target.files); event.target.value = '' }} />
    </div>
    <p className="drop-hint">Drop files here, paste an image into the editor, or choose files. Originals are copied into Solace.</p>
    {status && <div className="attachment-status">{status}</div>}
    {error && <div className="editor-error" role="alert">{error}</div>}
    <div className="attachment-grid">
      {attachments.map((attachment, index) => <article className={!attachment.available ? 'attachment-card missing' : 'attachment-card'} key={attachment.id}>
        <AttachmentPreview attachment={attachment} playbackError={playbackErrors[attachment.id]} onPlaybackError={() => setPlaybackErrors((current) => ({ ...current, [attachment.id]:true }))} onEnlarge={() => setEnlarged(attachment)} onOpen={() => void attachmentBridge.open(attachment)}/>
        <div className="attachment-info"><strong title={attachment.originalFilename}>{attachment.originalFilename}</strong><span>{formatSize(attachment.fileSize)} · {typeLabel(attachment)}</span></div>
        <input aria-label={`Caption for ${attachment.originalFilename}`} placeholder="Add a caption" value={attachment.caption} disabled={readOnly} onChange={(event) => setAttachments((current) => current.map((item) => item.id === attachment.id ? { ...item, caption:event.target.value } : item))} onBlur={(event) => void caption(attachment, event.target.value)} />
        {!readOnly && <div className="attachment-actions">
          {attachment.mimeType.startsWith('image/') && <button aria-label="Insert image into entry" onClick={() => onImagesAdded([attachment])}><ImagePlus size={14}/></button>}
          <button aria-label="Move attachment left" disabled={index === 0} onClick={() => void move(index, -1)}><ChevronLeft size={14}/></button>
          <button aria-label="Move attachment right" disabled={index === attachments.length - 1} onClick={() => void move(index, 1)}><ChevronRight size={14}/></button>
          <button aria-label="Open attachment externally" onClick={() => void attachmentBridge.open(attachment)}><FolderOpen size={14}/></button>
          <button className="danger" aria-label="Remove attachment" onClick={() => void remove(attachment)}><Trash2 size={14}/></button>
        </div>}
      </article>)}
    </div>
    {enlarged && <div className="image-lightbox" role="dialog" aria-label={enlarged.originalFilename} onClick={() => setEnlarged(null)}><img src={enlarged.url} alt={enlarged.caption || enlarged.originalFilename}/><p>{enlarged.caption || enlarged.originalFilename}</p></div>}
  </section>
}

function AttachmentPreview({ attachment, playbackError, onPlaybackError, onEnlarge, onOpen }: {
  attachment: Attachment; playbackError?: boolean; onPlaybackError: () => void; onEnlarge: () => void; onOpen: () => void
}) {
  if (!attachment.available) return <div className="file-preview unavailable"><File size={30}/><span>File missing</span></div>
  if (attachment.mimeType.startsWith('image/')) return <button className="image-preview" onClick={onEnlarge}><img src={attachment.thumbnailURL || attachment.url} alt={attachment.caption || attachment.originalFilename}/><Maximize2 size={16}/></button>
  if (attachment.mimeType.startsWith('video/')) return playbackError
    ? <button className="file-preview" onClick={onOpen}><Play size={30}/><span>Playback unsupported — open externally</span></button>
    : <video src={attachment.url} controls preload="metadata" onError={onPlaybackError}/>
  if (attachment.mimeType.startsWith('audio/')) return playbackError
    ? <button className="file-preview" onClick={onOpen}><Play size={30}/><span>Playback unsupported — open externally</span></button>
    : <div className="audio-preview"><audio src={attachment.url} controls preload="metadata" onError={onPlaybackError}/></div>
  if (attachment.mimeType === 'application/pdf') return <button className="file-preview" onClick={onOpen}><FileText size={30}/><span>Open PDF</span></button>
  return <button className="file-preview" onClick={onOpen}><File size={30}/><span>Open attachment</span></button>
}

function typeLabel(attachment: Attachment) { return attachment.mimeType.split('/')[1]?.toUpperCase() || 'FILE' }
function formatSize(size: number) { return size < 1024 ? `${size} B` : size < 1024 * 1024 ? `${(size / 1024).toFixed(1)} KB` : `${(size / 1024 / 1024).toFixed(1)} MB` }
