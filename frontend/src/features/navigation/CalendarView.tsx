import { useEffect, useMemo, useState } from 'react'
import { ChevronLeft, ChevronRight, Paperclip } from 'lucide-react'
import { diaryBridge } from '../entries/diary-bridge'
import type { CalendarDay, Entry } from '../entries/types'
import { navigationBridge } from './navigation-bridge'
import { friendlyError } from '../../lib/user-errors'

export function CalendarView({ onOpenEntry }: { onOpenEntry: (id: string) => void }) {
  const [month, setMonth] = useState(currentMonth())
  const [days, setDays] = useState<CalendarDay[]>([])
  const [selectedDate, setSelectedDate] = useState(today())
  const [entries, setEntries] = useState<Entry[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    navigationBridge.calendarMonth(month).then((value) => { if (active) { setDays(value); setError('') } }).catch((reason) => active && setError(friendlyError(reason, 'This month couldn’t be loaded. Please try again.', 'calendar month')))
    return () => { active = false }
  }, [month])
  useEffect(() => {
    let active = true
    diaryBridge.list({ startDate:selectedDate, endDate:selectedDate, limit:200 }).then((value) => active && setEntries(value)).catch((reason) => active && setError(friendlyError(reason, 'Entries for this date couldn’t be loaded. Please try again.', 'calendar entries')))
    return () => { active = false }
  }, [selectedDate])

  const dayMap = useMemo(() => new Map(days.map((day) => [day.date, day])), [days])
  const cells = calendarCells(month)
  const chooseDate = (date: string) => { setSelectedDate(date); setMonth(date.slice(0, 7)) }
  const jumpToday = () => { const value = today(); setSelectedDate(value); setMonth(value.slice(0, 7)) }
  return <div className="calendar-layout">
    <section className="calendar-panel">
      <div className="browser-toolbar">
        <button onClick={() => setMonth(shiftMonth(month, -1))} aria-label="Previous month"><ChevronLeft size={16}/></button>
        <button onClick={jumpToday}>Today</button>
        <input type="month" value={month} onChange={(event) => event.target.value && setMonth(event.target.value)} aria-label="Jump to month" />
        <button onClick={() => setMonth(shiftMonth(month, 1))} aria-label="Next month"><ChevronRight size={16}/></button>
      </div>
      <div className="calendar-title"><h2>{monthLabel(month)}</h2><span>{days.reduce((count, day) => count + day.entryCount, 0)} dated pages</span></div>
      {error && <div className="editor-error" role="alert">{error}</div>}
      <div className="calendar-grid" role="grid" aria-label={monthLabel(month)}>
        {['Sun','Mon','Tue','Wed','Thu','Fri','Sat'].map((label) => <div className="calendar-weekday" key={label}>{label}</div>)}
        {cells.map((date) => {
          const day = dayMap.get(date)
          const outside = date.slice(0, 7) !== month
          return <button key={date} role="gridcell" className={`${outside ? 'outside ' : ''}${selectedDate === date ? 'selected ' : ''}${date === today() ? 'today' : ''}`} onClick={() => chooseDate(date)}>
            <span className="calendar-day-number">{Number(date.slice(8))}</span>
            {day && <span className="calendar-indicators"><span className="entry-count">{day.entryCount}</span>{day.mood && <span title={day.mood}>{moodMark(day.mood)}</span>}{day.hasAttachments && <Paperclip size={11}/>}</span>}
          </button>
        })}
      </div>
    </section>
    <aside className="date-entries">
      <span className="eyebrow">Selected date</span><h3>{longDate(selectedDate)}</h3>
      {entries.length === 0 && <p className="muted-copy">No pages cover this date.</p>}
      {entries.map((entry) => <button key={entry.id} onClick={() => onOpenEntry(entry.id)}><strong>{entry.title || 'Untitled entry'}</strong><span>{entry.mood || (entry.draft ? 'Draft' : 'Published')}{entry.startDate !== entry.endDate ? ' · Multi-day' : ''}</span></button>)}
    </aside>
  </div>
}

function calendarCells(month: string) { const [year, value] = month.split('-').map(Number); const first = new Date(year, value - 1, 1); first.setDate(first.getDate() - first.getDay()); return Array.from({ length:42 }, (_, index) => { const date = new Date(first); date.setDate(first.getDate() + index); return dateOnly(date) }) }
function shiftMonth(month: string, amount: number) { const [year, value] = month.split('-').map(Number); const date = new Date(year, value - 1 + amount, 1); return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}` }
function currentMonth() { return today().slice(0, 7) }
function today() { return dateOnly(new Date()) }
function dateOnly(date: Date) { return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
function localDate(value: string) { const [year, month, day] = value.split('-').map(Number); return new Date(year, month - 1, day) }
function longDate(value: string) { return new Intl.DateTimeFormat(undefined, { weekday:'long', month:'long', day:'numeric', year:'numeric' }).format(localDate(value)) }
function monthLabel(value: string) { return new Intl.DateTimeFormat(undefined, { month:'long', year:'numeric' }).format(localDate(`${value}-01`)) }
function moodMark(mood: string) { return mood === 'Very Happy' ? '●' : mood === 'Happy' ? '●' : mood === 'Sad' || mood === 'Very Sad' ? '○' : '·' }
