import * as NativeDiary from '../../../bindings/solace/internal/diary/service'
import type { EntryInput as NativeInput, ListOptions as NativeOptions } from '../../../bindings/solace/internal/diary/models'
import { isWailsHost } from '../../lib/bridge'
import type { Entry, EntryInput, ListOptions, Revision } from './types'

const storageKey = 'solace.entries'
const revisionStorageKey = 'solace.revisions'

function readLocal(): Entry[] {
  try {
    return (JSON.parse(localStorage.getItem(storageKey) ?? '[]') as Entry[]).map((entry) => ({ ...entry, pinned:Boolean(entry.pinned), tags:entry.tags ?? [] }))
  } catch {
    return []
  }
}

function writeLocal(entries: Entry[]) {
  localStorage.setItem(storageKey, JSON.stringify(entries))
}

function readLocalRevisions(): Revision[] {
  try { return JSON.parse(localStorage.getItem(revisionStorageKey) ?? '[]') as Revision[] } catch { return [] }
}
function writeLocalRevisions(revisions: Revision[]) { localStorage.setItem(revisionStorageKey, JSON.stringify(revisions)) }

function localID() {
  return globalThis.crypto?.randomUUID?.() ?? `preview-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function normalize(entry: Omit<Entry, 'tags'> & { tags?: string[] | null }): Entry {
  return { ...entry, pinned:Boolean(entry.pinned), tags: entry.tags ?? [] }
}

function nativeInput(input: EntryInput): NativeInput {
  return { ...input, tags: input.tags }
}

function nativeOptions(options: ListOptions): NativeOptions {
  return {
    startDate: options.startDate ?? '', endDate: options.endDate ?? '', sort: options.sort ?? 'newest',
    favoritesOnly: options.favoritesOnly ?? false, draftsOnly: options.draftsOnly ?? false,
    includeTrash: options.includeTrash ?? false, limit: options.limit ?? 100, offset: options.offset ?? 0,
  }
}

export const diaryBridge = {
  async create(input: EntryInput): Promise<Entry> {
    if (isWailsHost()) return normalize(await NativeDiary.CreateEntry(nativeInput(input)))
    const now = new Date().toISOString()
    const entry: Entry = { ...input, id: localID(), createdAt: now, updatedAt: now, deletedAt: '', version: 1 }
    writeLocal([entry, ...readLocal()])
    return entry
  },
  async update(id: string, input: EntryInput, expectedVersion: number): Promise<Entry> {
    if (isWailsHost()) return normalize(await NativeDiary.UpdateEntry(id, nativeInput(input), expectedVersion))
    const entries = readLocal()
    const index = entries.findIndex((entry) => entry.id === id && !entry.deletedAt)
    if (index < 0) throw new Error('Entry not found')
    if (entries[index].version !== expectedVersion) throw new Error('Entry was changed by another operation')
    const previous = entries[index]
    const revision: Revision = { ...previous, entryID:previous.id, id:localID(), sourceVersion:previous.version, savedAt:new Date().toISOString() }
    const revisions = [revision, ...readLocalRevisions().filter((item) => item.entryID !== id).slice(0, 19), ...readLocalRevisions().filter((item) => item.entryID === id).slice(0, 19)]
    writeLocalRevisions(revisions.filter((item, index, all) => item.entryID !== id || all.slice(0, index).filter((candidate) => candidate.entryID === id).length < 20))
    entries[index] = { ...entries[index], ...input, updatedAt: new Date().toISOString(), version: expectedVersion + 1 }
    writeLocal(entries)
    return entries[index]
  },
  async list(options: ListOptions = {}): Promise<Entry[]> {
    if (isWailsHost()) return (await NativeDiary.ListEntries(nativeOptions(options)) ?? []).map(normalize)
    const start = options.startDate ?? ''
    const end = options.endDate ?? start
    const entries = readLocal().filter((entry) => {
      if (options.includeTrash ? !entry.deletedAt : entry.deletedAt) return false
      if (options.favoritesOnly && !entry.favorite) return false
      return !start || (entry.startDate <= end && entry.endDate >= start)
    })
    const sort = options.sort ?? 'newest'
    entries.sort((a, b) => sort === 'oldest' ? a.startDate.localeCompare(b.startDate) : sort === 'updated' ? b.updatedAt.localeCompare(a.updatedAt) : b.startDate.localeCompare(a.startDate))
    return entries.slice(options.offset ?? 0, (options.offset ?? 0) + (options.limit ?? 100))
  },
  async trash(id: string) {
    if (isWailsHost()) return NativeDiary.TrashEntry(id)
    writeLocal(readLocal().map((entry) => entry.id === id ? { ...entry, deletedAt: new Date().toISOString(), version: entry.version + 1 } : entry))
  },
  async restore(id: string) {
    if (isWailsHost()) return NativeDiary.RestoreEntry(id)
    writeLocal(readLocal().map((entry) => entry.id === id ? { ...entry, deletedAt: '', version: entry.version + 1 } : entry))
  },
  async permanentlyDelete(id: string) {
    if (isWailsHost()) return NativeDiary.PermanentlyDeleteEntry(id)
    writeLocal(readLocal().filter((entry) => entry.id !== id))
  },
  async duplicate(id: string): Promise<Entry> {
    if (isWailsHost()) return normalize(await NativeDiary.DuplicateEntry(id))
    const entry = readLocal().find((candidate) => candidate.id === id)
    if (!entry) throw new Error('Entry not found')
    return this.create({ ...toInput(entry), title: entry.title ? `${entry.title} (copy)` : 'Copy', favorite: false, draft: true })
  },
  async listRevisions(entryID: string): Promise<Revision[]> {
    if (isWailsHost()) return (await NativeDiary.ListRevisions(entryID) ?? []).map((revision) => ({ ...revision, tags:revision.tags ?? [] }))
    return readLocalRevisions().filter((revision) => revision.entryID === entryID).sort((a, b) => b.savedAt.localeCompare(a.savedAt)).slice(0, 20)
  },
  async restoreRevision(entryID: string, revisionID: string, expectedVersion: number): Promise<Entry> {
    if (isWailsHost()) return normalize(await NativeDiary.RestoreRevision(entryID, revisionID, expectedVersion))
    const revision = readLocalRevisions().find((item) => item.id === revisionID && item.entryID === entryID)
    if (!revision) throw new Error('Revision not found')
    return diaryBridge.update(entryID, {
      title:revision.title, contentJSON:revision.contentJSON, contentText:revision.contentText, startDate:revision.startDate, endDate:revision.endDate,
      mood:revision.mood, favorite:revision.favorite, pinned:revision.pinned, draft:revision.draft, locationText:revision.locationText,
      displayOrder:revision.displayOrder, tags:revision.tags,
    }, expectedVersion)
  },
}

export function toInput(entry: Entry): EntryInput {
  return {
    title: entry.title, contentJSON: entry.contentJSON, contentText: entry.contentText, startDate: entry.startDate, endDate: entry.endDate,
    mood: entry.mood, favorite: entry.favorite, pinned:entry.pinned, draft: entry.draft, locationText: entry.locationText,
    displayOrder: entry.displayOrder, tags: entry.tags,
  }
}
