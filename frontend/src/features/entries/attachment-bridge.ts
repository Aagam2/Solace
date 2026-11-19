import { Dialogs, Events } from '@wailsio/runtime'
import * as NativeAttachments from '../../../bindings/solace/internal/attachments/service'
import { isWailsHost } from '../../lib/bridge'
import { diaryBridge } from './diary-bridge'
import type { Attachment } from './types'

const storageKey = 'solace.attachments'

function readLocal(): Attachment[] {
  try { return JSON.parse(localStorage.getItem(storageKey) ?? '[]') as Attachment[] } catch { return [] }
}
function writeLocal(attachments: Attachment[]) { localStorage.setItem(storageKey, JSON.stringify(attachments)) }
function localID() { return globalThis.crypto?.randomUUID?.() ?? `attachment-${Date.now()}-${Math.random().toString(16).slice(2)}` }

export const attachmentBridge = {
  async list(entryID: string): Promise<Attachment[]> {
    if (isWailsHost()) return await NativeAttachments.ListAttachments(entryID) ?? []
    return readLocal().filter((attachment) => attachment.entryID === entryID).sort((a, b) => a.displayOrder - b.displayOrder)
  },
  async pick(entryID: string): Promise<Attachment[]> {
    if (!isWailsHost()) return []
    const paths = await Dialogs.OpenFile({
      AllowsMultipleSelection:true, CanChooseFiles:true, CanChooseDirectories:false,
      Title:'Attach files to this entry', ButtonText:'Attach', AllowsOtherFiletypes:true,
      Filters:[
        { DisplayName:'Images and media', Pattern:'*.jpg;*.jpeg;*.png;*.webp;*.gif;*.mp4;*.webm;*.mov;*.mp3;*.wav;*.m4a;*.pdf' },
        { DisplayName:'All files', Pattern:'*' },
      ],
    })
    return paths.length ? await NativeAttachments.ImportFiles(entryID, paths) ?? [] : []
  },
  async importFiles(entryID: string, files: File[]): Promise<Attachment[]> {
    const imported: Attachment[] = []
    for (const file of files) {
      if (isWailsHost()) {
        imported.push(await NativeAttachments.ImportBase64(entryID, file.name, file.type, await fileBase64(file)))
      } else {
        const current = readLocal().filter((item) => item.entryID === entryID)
        const id = localID()
        imported.push({
          id, entryID, originalFilename:file.name, storedFilename:id, relativePath:'browser-preview',
          mimeType:file.type || 'application/octet-stream', fileSize:file.size, createdAt:new Date().toISOString(),
          caption:'', displayOrder:current.length + imported.length, checksum:'preview', metadataJSON:'{}',
          available:true, url:await fileDataURL(file), thumbnailURL:'',
        })
        imported[imported.length - 1].thumbnailURL = imported[imported.length - 1].url
      }
    }
    if (!isWailsHost()) writeLocal([...readLocal(), ...imported])
    return imported
  },
  async importPaths(entryID: string, paths: string[]) {
    if (!isWailsHost() || !paths.length) return []
    return await NativeAttachments.ImportFiles(entryID, paths) ?? []
  },
  async caption(id: string, caption: string) {
    if (isWailsHost()) return NativeAttachments.UpdateCaption(id, caption)
    writeLocal(readLocal().map((item) => item.id === id ? { ...item, caption } : item))
  },
  async reorder(entryID: string, ids: string[]) {
    if (isWailsHost()) return NativeAttachments.ReorderAttachments(entryID, ids)
    const order = new Map(ids.map((id, index) => [id, index]))
    writeLocal(readLocal().map((item) => item.entryID === entryID ? { ...item, displayOrder:order.get(item.id) ?? item.displayOrder } : item))
  },
  async remove(id: string) {
    if (isWailsHost()) return NativeAttachments.RemoveAttachment(id)
    writeLocal(readLocal().filter((item) => item.id !== id))
  },
  async permanentlyDeleteEntry(entryID: string) {
    if (isWailsHost()) return NativeAttachments.PermanentlyDeleteEntry(entryID)
    writeLocal(readLocal().filter((item) => item.entryID !== entryID))
    return diaryBridge.permanentlyDelete(entryID)
  },
  async open(attachment: Attachment) {
    if (isWailsHost()) return NativeAttachments.OpenAttachment(attachment.id)
    window.open(attachment.url, '_blank', 'noopener,noreferrer')
  },
  onNativeDrop(callback: (paths: string[]) => void) {
    if (!isWailsHost()) return () => undefined
    return Events.On('solace:files-dropped', (event) => {
      const data = event.data as string[] | [string[]]
      callback(Array.isArray(data[0]) ? data[0] : data as string[])
    })
  },
}

function fileDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(reader.error ?? new Error('Unable to read file'))
    reader.readAsDataURL(file)
  })
}

async function fileBase64(file: File) {
  const dataURL = await fileDataURL(file)
  return dataURL.slice(dataURL.indexOf(',') + 1)
}
