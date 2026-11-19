import * as NativeDiary from '../../../bindings/solace/internal/diary/service'
import { isWailsHost } from '../../lib/bridge'
import { diaryBridge } from '../entries/diary-bridge'
import type { Attachment, Entry, EntrySummary, SearchOptions, SearchPage, SearchResult, TagSummary } from '../entries/types'

const tagsKey = 'solace.tags'

export const searchBridge = {
  async search(options: SearchOptions): Promise<SearchPage> {
    if (isWailsHost()) {
      const page = await NativeDiary.Search(options)
      return { ...page, results:(page.results ?? []).map((result) => ({ ...result, tags:result.tags ?? [] })) }
    }
    const entries = await diaryBridge.list({ limit:10000 })
    const attachments = readAttachments()
    const terms = tokenize(options.query)
    let results = entries.filter((entry) => matches(entry, attachments, options, terms)).map((entry) => localResult(entry, attachments, terms))
    if (options.sort === 'oldest') results.sort((a, b) => a.entry.startDate.localeCompare(b.entry.startDate))
    else if (options.sort === 'newest' || !terms.length) results.sort((a, b) => b.entry.startDate.localeCompare(a.entry.startDate))
    else results.sort((a, b) => score(b, terms) - score(a, terms) || b.entry.startDate.localeCompare(a.entry.startDate))
    return { results:results.slice(options.offset, options.offset + options.limit), total:results.length, hasMore:options.offset + options.limit < results.length }
  },
  async listTags(): Promise<TagSummary[]> {
    if (isWailsHost()) return await NativeDiary.ListTags() ?? []
    const entries = await diaryBridge.list({ limit:10000 })
    const explicit = readTags()
    const names = new Map(explicit.map((tag) => [tag.name.toLocaleLowerCase(), tag]))
    for (const entry of entries) for (const name of entry.tags) if (!names.has(name.toLocaleLowerCase())) names.set(name.toLocaleLowerCase(), { id:localID(), name, entryCount:0 })
    const result = [...names.values()].map((tag) => ({ ...tag, entryCount:entries.filter((entry) => entry.tags.some((name) => same(name, tag.name))).length })).sort((a, b) => a.name.localeCompare(b.name))
    writeTags(result.map(({ id, name }) => ({ id, name, entryCount:0 })))
    return result
  },
  async createTag(name: string): Promise<TagSummary> {
    if (isWailsHost()) return await NativeDiary.CreateTag(name)
    const clean = tagName(name), tags = await this.listTags()
    if (tags.some((tag) => same(tag.name, clean))) throw new Error('A tag with that name already exists')
    const tag = { id:localID(), name:clean, entryCount:0 }; writeTags([...readTags(), tag]); return tag
  },
  async renameTag(id: string, name: string) {
    if (isWailsHost()) return NativeDiary.RenameTag(id, name)
    const clean = tagName(name), tags = await this.listTags(), current = tags.find((tag) => tag.id === id)
    if (!current) throw new Error('Tag not found')
    if (tags.some((tag) => tag.id !== id && same(tag.name, clean))) throw new Error('A tag with that name already exists')
    writeTags(readTags().map((tag) => tag.id === id ? { ...tag, name:clean } : tag))
    await rewriteEntryTags(current.name, clean)
  },
  async deleteTag(id: string) {
    if (isWailsHost()) return NativeDiary.DeleteTag(id)
    const tag = (await this.listTags()).find((candidate) => candidate.id === id)
    if (!tag) throw new Error('Tag not found')
    writeTags(readTags().filter((candidate) => candidate.id !== id)); await rewriteEntryTags(tag.name, '')
  },
  async rebuild() { if (isWailsHost()) return NativeDiary.RebuildSearchIndex() },
}

function matches(entry: Entry, attachments: Attachment[], options: SearchOptions, terms: string[]) {
  const entryAttachments = attachments.filter((item) => item.entryID === entry.id)
  const haystack = [entry.title, entry.contentText, entry.tags.join(' '), entryAttachments.map((item) => item.caption).join(' ')].join(' ').toLocaleLowerCase()
  if (!terms.every((term) => haystack.includes(term))) return false
  if ((options.startDate || options.endDate) && !(entry.startDate <= (options.endDate || options.startDate) && entry.endDate >= (options.startDate || options.endDate))) return false
  if (!options.tags.every((tag) => entry.tags.some((name) => same(name, tag)))) return false
  if (options.mood && entry.mood !== options.mood) return false
  if (options.favoritesOnly && !entry.favorite) return false
  if (options.hasImages && !entryAttachments.some((item) => item.mimeType.startsWith('image/'))) return false
  if (options.hasVideos && !entryAttachments.some((item) => item.mimeType.startsWith('video/'))) return false
  if (options.draftStatus === 'draft' && !entry.draft || options.draftStatus === 'published' && entry.draft) return false
  return true
}
function localResult(entry: Entry, attachments: Attachment[], terms: string[]): SearchResult {
  const captions = attachments.filter((item) => item.entryID === entry.id).map((item) => item.caption).join(' ')
  const text = entry.contentText || captions || entry.tags.join(' ')
  return { entry:summary(entry, attachments), highlightedTitle:highlight(entry.title || 'Untitled entry', terms), snippet:highlight(snippet(text, terms), terms), tags:entry.tags }
}
function summary(entry: Entry, attachments: Attachment[]): EntrySummary { return { id:entry.id, title:entry.title, preview:entry.contentText.slice(0,180), startDate:entry.startDate, endDate:entry.endDate, updatedAt:entry.updatedAt, mood:entry.mood, favorite:entry.favorite, pinned:entry.pinned, draft:entry.draft, attachmentCount:attachments.filter((item) => item.entryID === entry.id).length } }
function score(result: SearchResult, terms: string[]) { const value = `${result.entry.title} ${result.snippet}`.toLocaleLowerCase(); return terms.reduce((total, term) => total + (value.split(term).length - 1), 0) }
function snippet(value: string, terms: string[]) { const lower = value.toLocaleLowerCase(); const index = terms.reduce((best, term) => { const found = lower.indexOf(term); return found >= 0 && (best < 0 || found < best) ? found : best }, -1); const start = Math.max(0, index - 70); return `${start ? '…' : ''}${value.slice(start, start + 220)}${start + 220 < value.length ? '…' : ''}` }
function highlight(value: string, terms: string[]) { if (!terms.length) return value; const escaped = terms.map((term) => term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|'); return value.replace(new RegExp(`(${escaped})`, 'giu'), '<mark>$1</mark>') }
function tokenize(value: string) { return value.toLocaleLowerCase().match(/[\p{L}\p{N}_]+/gu) ?? [] }
function readAttachments() { try { return JSON.parse(localStorage.getItem('solace.attachments') ?? '[]') as Attachment[] } catch { return [] } }
function readTags() { try { return JSON.parse(localStorage.getItem(tagsKey) ?? '[]') as TagSummary[] } catch { return [] } }
function writeTags(tags: TagSummary[]) { localStorage.setItem(tagsKey, JSON.stringify(tags)) }
async function rewriteEntryTags(oldName: string, newName: string) { const entries = await diaryBridge.list({ limit:10000 }); for (const entry of entries) if (entry.tags.some((tag) => same(tag, oldName))) await diaryBridge.update(entry.id, { ...entry, tags:entry.tags.flatMap((tag) => same(tag, oldName) ? (newName ? [newName] : []) : [tag]) }, entry.version) }
function tagName(value: string) { const clean = value.trim(); if (!clean) throw new Error('Tag name is required'); if ([...clean].length > 80) throw new Error('Tag name must be at most 80 characters'); return clean }
function same(left: string, right: string) { return left.localeCompare(right, undefined, { sensitivity:'accent' }) === 0 }
function localID() { return globalThis.crypto?.randomUUID?.() ?? `tag-${Date.now()}-${Math.random().toString(16).slice(2)}` }
