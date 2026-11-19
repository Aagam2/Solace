import { useCallback, useEffect, useMemo, useState } from 'react'
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react'
import type { TimelineOccurrence } from '../entries/types'
import { navigationBridge } from './navigation-bridge'
import { friendlyError } from '../../lib/user-errors'

const PAGE_SIZE = 100

export function TimelineView({ onOpenEntry }: { onOpenEntry: (id: string) => void }) {
  const [anchor, setAnchor] = useState(today())
  const [scope, setScope] = useState<'month' | 'year'>('month')
  const [items, setItems] = useState<TimelineOccurrence[]>([])
  const [total, setTotal] = useState(0)
  const [active, setActive] = useState(0)
  const [error, setError] = useState('')
  const bounds = useMemo(() => periodBounds(anchor, scope), [anchor, scope])
  const load = useCallback(async (offset = 0) => {
    try {
      const page = await navigationBridge.timeline(bounds.start, bounds.end, PAGE_SIZE, offset)
      setItems((current) => offset ? [...current, ...page.occurrences] : page.occurrences); setTotal(page.total); setError('')
    } catch (reason) { setError(friendlyError(reason, 'The timeline couldn’t be loaded. Please try again.', 'timeline')) }
  }, [bounds.end, bounds.start])
  useEffect(() => { setActive(0); void load() }, [load])

  const stepEntry = (amount: number) => {
    if (!items.length) return
    const next = Math.max(0, Math.min(items.length - 1, active + amount)); setActive(next)
    document.getElementById(`timeline-${next}`)?.scrollIntoView({ behavior:'smooth', block:'center' })
  }
  const shift = (amount: number) => setAnchor(shiftPeriod(anchor, scope, amount))
  let lastYear = ''; let lastMonth = ''; let lastDay = ''
  return <section className="timeline-panel">
    <div className="browser-toolbar timeline-toolbar">
      <button onClick={() => shift(-1)} aria-label="Previous period"><ChevronLeft size={16}/></button>
      <button onClick={() => setAnchor(today())}>Today</button>
      <input type="date" value={anchor} onChange={(event) => event.target.value && setAnchor(event.target.value)} aria-label="Jump to date" />
      {scope === 'month' ? <input type="month" value={anchor.slice(0, 7)} onChange={(event) => event.target.value && setAnchor(`${event.target.value}-01`)} aria-label="Jump to month" /> : <input type="number" min="1" max="9999" value={anchor.slice(0, 4)} onChange={(event) => event.target.value && setAnchor(`${event.target.value.padStart(4, '0')}-01-01`)} aria-label="Jump to year" />}
      <select value={scope} onChange={(event) => setScope(event.target.value as 'month'|'year')} aria-label="Timeline scope"><option value="month">Month</option><option value="year">Year</option></select>
      <button onClick={() => shift(1)} aria-label="Next period"><ChevronRight size={16}/></button>
      <span className="toolbar-spacer" />
      <button onClick={() => stepEntry(-1)} disabled={!items.length || active === 0}>Previous entry</button><button onClick={() => stepEntry(1)} disabled={!items.length || active === items.length - 1}>Next entry</button>
    </div>
    <div className="timeline-heading"><div><span className="eyebrow">Chronological browsing</span><h2>{periodLabel(anchor, scope)}</h2></div><span>{total} appearances</span></div>
    {error && <div className="editor-error" role="alert">{error}</div>}
    {!items.length && !error && <div className="empty-state"><h2>No entries in this {scope}.</h2><p>Jump to another period or create a dated entry.</p></div>}
    <div className="timeline-list">
      {items.map((item, index) => {
        const year = item.date.slice(0, 4), month = item.date.slice(0, 7), day = item.date
        const showYear = year !== lastYear, showMonth = month !== lastMonth, showDay = day !== lastDay
        lastYear = year; lastMonth = month; lastDay = day
        return <div key={`${item.date}-${item.entry.id}`}>
          {showYear && <h3 className="timeline-year">{year}</h3>}
          {showMonth && <h4 className="timeline-month">{monthName(month)}</h4>}
          {showDay && <div className="timeline-day">{dayLabel(day)}</div>}
          <button id={`timeline-${index}`} className={`timeline-entry ${active === index ? 'active' : ''}`} onFocus={() => setActive(index)} onClick={() => onOpenEntry(item.entry.id)}>
            <div><strong>{item.entry.title || 'Untitled entry'}</strong><p>{item.entry.preview || 'A blank page'}</p></div>
            <span>{item.entry.pinned ? 'Pinned · ' : ''}{item.entry.mood || (item.entry.draft ? 'Draft' : 'Published')}{item.entry.startDate !== item.entry.endDate ? ' · Multi-day' : ''}</span>
          </button>
        </div>
      })}
    </div>
    {items.length < total && <button className="load-more" onClick={() => void load(items.length)}><ChevronDown size={15}/>Load more</button>}
  </section>
}

function periodBounds(anchor: string, scope: 'month'|'year') { const year = Number(anchor.slice(0, 4)), month = Number(anchor.slice(5, 7)); return scope === 'year' ? { start:`${year}-01-01`, end:`${year}-12-31` } : { start:`${anchor.slice(0, 7)}-01`, end:dateOnly(new Date(year, month, 0)) } }
function shiftPeriod(anchor: string, scope: 'month'|'year', amount: number) { const [year, month] = anchor.split('-').map(Number); const date = scope === 'year' ? new Date(year + amount, 0, 1) : new Date(year, month - 1 + amount, 1); return dateOnly(date) }
function today() { return dateOnly(new Date()) }
function dateOnly(date: Date) { return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
function localDate(value: string) { const [year, month, day] = value.split('-').map(Number); return new Date(year, month - 1, day) }
function periodLabel(anchor: string, scope: 'month'|'year') { return scope === 'year' ? anchor.slice(0, 4) : new Intl.DateTimeFormat(undefined, { month:'long', year:'numeric' }).format(localDate(anchor)) }
function monthName(month: string) { return new Intl.DateTimeFormat(undefined, { month:'long' }).format(localDate(`${month}-01`)) }
function dayLabel(day: string) { return new Intl.DateTimeFormat(undefined, { weekday:'long', month:'short', day:'numeric' }).format(localDate(day)) }
