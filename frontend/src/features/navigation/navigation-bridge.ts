import * as NativeDiary from '../../../bindings/solace/internal/diary/service'
import { isWailsHost } from '../../lib/bridge'
import { diaryBridge } from '../entries/diary-bridge'
import type { CalendarDay, Entry, EntrySummary, TimelineOccurrence, TimelinePage, TodayDashboard } from '../entries/types'

function summary(entry: Entry): EntrySummary {
  return {
    id:entry.id, title:entry.title, preview:entry.contentText.slice(0, 180), startDate:entry.startDate, endDate:entry.endDate,
    updatedAt:entry.updatedAt, mood:entry.mood, favorite:entry.favorite, pinned:entry.pinned, draft:entry.draft, attachmentCount:0,
  }
}

export const navigationBridge = {
  async calendarMonth(month: string): Promise<CalendarDay[]> {
    if (isWailsHost()) return await NativeDiary.CalendarMonth(month) ?? []
    const start = `${month}-01`; const end = monthEnd(month)
    const entries = await diaryBridge.list({ startDate:start, endDate:end, limit:10000 })
    const days = new Map<string, CalendarDay>()
    for (const entry of entries) for (let date = max(entry.startDate, start); date <= min(entry.endDate, end); date = addDays(date, 1)) {
      const day = days.get(date) ?? { date, entryCount:0, mood:'', hasAttachments:false }
      day.entryCount++; day.mood ||= entry.mood; days.set(date, day)
    }
    return [...days.values()].sort((a, b) => a.date.localeCompare(b.date))
  },
  async timeline(startDate: string, endDate: string, limit: number, offset: number): Promise<TimelinePage> {
    if (isWailsHost()) {
      const page = await NativeDiary.Timeline({ startDate, endDate, limit, offset })
      return { ...page, occurrences:page.occurrences ?? [] }
    }
    const entries = await diaryBridge.list({ startDate, endDate, limit:10000 })
    const occurrences: TimelineOccurrence[] = []
    for (const entry of entries) for (let date = max(entry.startDate, startDate); date <= min(entry.endDate, endDate); date = addDays(date, 1)) occurrences.push({ date, entry:summary(entry) })
    occurrences.sort((a, b) => b.date.localeCompare(a.date) || Number(b.entry.pinned) - Number(a.entry.pinned) || b.entry.updatedAt.localeCompare(a.entry.updatedAt))
    return { occurrences:occurrences.slice(offset, offset + limit), total:occurrences.length, hasMore:offset + limit < occurrences.length }
  },
  async dashboard(today: string): Promise<TodayDashboard> {
    if (isWailsHost()) {
      const value = await NativeDiary.TodayDashboard(today)
      return { ...value, todayEntries:value.todayEntries ?? [], recentlyEdited:value.recentlyEdited ?? [], recentMemories:value.recentMemories ?? [], onThisDay:value.onThisDay ?? [] }
    }
    const entries = await diaryBridge.list({ limit:10000 })
    const summaries = entries.map(summary)
    const currentYear = Number(today.slice(0, 4)); const monthDay = today.slice(5)
    return {
      today,
      todayEntries:summaries.filter((entry) => entry.startDate <= today && entry.endDate >= today).slice(0, 20),
      recentlyEdited:[...summaries].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)).slice(0, 6),
      recentMemories:summaries.filter((entry) => entry.endDate < today).sort((a, b) => b.endDate.localeCompare(a.endDate)).slice(0, 6),
      onThisDay:summaries.filter((entry) => coversMonthDay(entry, currentYear, monthDay)).slice(0, 8),
    }
  },
}

function coversMonthDay(entry: EntrySummary, currentYear: number, monthDay: string) {
  for (let year = Number(entry.startDate.slice(0, 4)); year < currentYear && year <= Number(entry.endDate.slice(0, 4)); year++) {
    const date = `${year}-${monthDay}`
    if (validDate(date) && date >= entry.startDate && date <= entry.endDate) return true
  }
  return false
}
function addDays(value: string, amount: number) { const date = localDate(value); date.setDate(date.getDate() + amount); return dateOnly(date) }
function monthEnd(month: string) { const [year, value] = month.split('-').map(Number); return dateOnly(new Date(year, value, 0)) }
function localDate(value: string) { const [year, month, day] = value.split('-').map(Number); return new Date(year, month - 1, day) }
function dateOnly(date: Date) { return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
function validDate(value: string) { const date = localDate(value); return dateOnly(date) === value }
function max(left: string, right: string) { return left > right ? left : right }
function min(left: string, right: string) { return left < right ? left : right }
