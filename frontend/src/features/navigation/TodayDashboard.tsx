import { useEffect, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { Heart, Paperclip, Pin, Plus } from 'lucide-react'
import type { EntrySummary, TodayDashboard as DashboardData } from '../entries/types'
import { navigationBridge } from './navigation-bridge'
import { friendlyError } from '../../lib/user-errors'

type DashboardTab = 'today' | 'on-this-day' | 'recent'

const dashboardTabs: Array<{ id: DashboardTab; label: string }> = [
  { id:'today', label:'Today’s pages' },
  { id:'on-this-day', label:'On this day' },
  { id:'recent', label:'Recent' },
]

export function TodayDashboard({ onNew, onOpenEntry }: { onNew: () => void; onOpenEntry: (id: string) => void }) {
  const [dashboard, setDashboard] = useState<DashboardData | null>(null)
  const [activeTab, setActiveTab] = useState<DashboardTab>('today')
  const [error, setError] = useState('')
  useEffect(() => { navigationBridge.dashboard(today()).then(setDashboard).catch((reason) => setError(friendlyError(reason, 'Your journal summary couldn’t be loaded. Please try again.', 'today summary'))) }, [])
  return <>
    <section className="welcome-card">
      <span className="ornament" aria-hidden="true">✦</span>
      <div><p className="kicker">A quiet place for what matters</p><h2>Good evening.</h2>
      <p>When you’re ready, Solace will be here to hold the moments you want to remember.</p></div>
      <button onClick={onNew}><Plus size={17} aria-hidden="true"/>Write today’s entry</button>
    </section>
    {error && <div className="editor-error" role="alert">{error}</div>}
    {dashboard && <section className="dashboard-tabs" aria-label="Journal overview">
      <div className="dashboard-tab-list" role="tablist" aria-label="Journal overview sections">
        {dashboardTabs.map((tab) => <button key={tab.id} id={`dashboard-tab-${tab.id}`} role="tab" aria-selected={activeTab === tab.id} aria-controls="dashboard-tab-panel" tabIndex={activeTab === tab.id ? 0 : -1} onClick={() => setActiveTab(tab.id)} onKeyDown={(event) => moveTab(event, tab.id, setActiveTab)}>
          {tab.label}<span>{tabCount(tab.id, dashboard)}</span>
        </button>)}
      </div>
      <div id="dashboard-tab-panel" className={activeTab === 'recent' ? 'dashboard-tab-panel recent' : 'dashboard-tab-panel'} role="tabpanel" aria-labelledby={`dashboard-tab-${activeTab}`}>
        {activeTab === 'today' && <DashboardSection title="Today’s pages" items={dashboard.todayEntries} empty="Nothing written for today yet." onOpen={onOpenEntry}/>} 
        {activeTab === 'on-this-day' && <DashboardSection title="On this day" items={dashboard.onThisDay} empty="Past years will appear here." onOpen={onOpenEntry}/>} 
        {activeTab === 'recent' && <><DashboardSection title="Recently edited" items={dashboard.recentlyEdited} empty="Edited pages will appear here." onOpen={onOpenEntry}/><DashboardSection title="Recent memories" items={dashboard.recentMemories} empty="Older memories will appear here." onOpen={onOpenEntry}/></>}
      </div>
    </section>}
  </>
}

function DashboardSection({ title, items, empty, onOpen }: { title:string; items:EntrySummary[]; empty:string; onOpen:(id:string)=>void }) {
  return <section className="dashboard-section"><div className="section-heading"><h3>{title}</h3><span className="count">{items.length}</span></div>
    {!items.length && <p className="muted-copy">{empty}</p>}
    {items.map((entry) => <button key={entry.id} onClick={() => onOpen(entry.id)}><div><strong>{entry.title || 'Untitled entry'}</strong><p>{entry.preview || formatRange(entry)}</p></div><span className="summary-icons">{entry.pinned && <Pin size={12}/>} {entry.favorite && <Heart size={12}/>} {entry.attachmentCount > 0 && <Paperclip size={12}/>}</span></button>)}
  </section>
}
function formatRange(entry: EntrySummary) { return entry.startDate === entry.endDate ? entry.startDate : `${entry.startDate} – ${entry.endDate}` }
function today() { const date = new Date(); return `${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,'0')}-${String(date.getDate()).padStart(2,'0')}` }
function tabCount(tab: DashboardTab, dashboard: DashboardData) {
  if (tab === 'today') return dashboard.todayEntries.length
  if (tab === 'on-this-day') return dashboard.onThisDay.length
  return dashboard.recentlyEdited.length + dashboard.recentMemories.length
}
function moveTab(event: KeyboardEvent<HTMLButtonElement>, current: DashboardTab, select: (tab: DashboardTab) => void) {
  if (!['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return
  event.preventDefault()
  const currentIndex = dashboardTabs.findIndex((tab) => tab.id === current)
  const nextIndex = event.key === 'Home' ? 0 : event.key === 'End' ? dashboardTabs.length - 1 : (currentIndex + (event.key === 'ArrowRight' ? 1 : -1) + dashboardTabs.length) % dashboardTabs.length
  const next = dashboardTabs[nextIndex].id
  select(next)
  window.requestAnimationFrame(() => document.getElementById(`dashboard-tab-${next}`)?.focus())
}
