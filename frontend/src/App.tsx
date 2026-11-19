import { useCallback, useEffect, useState } from 'react'
import { Archive, CalendarDays, ChevronLeft, Clock3, Heart, LockKeyhole, Menu, Moon, Plus, Search, Settings, Sun, Tag, Trash2 } from 'lucide-react'
import { SolaceMark } from './components/SolaceMark'
import { EntryWorkspace } from './features/entries/EntryWorkspace'
import type { EntryView } from './features/entries/EntryWorkspace'
import { CalendarView } from './features/navigation/CalendarView'
import { TimelineView } from './features/navigation/TimelineView'
import { TodayDashboard } from './features/navigation/TodayDashboard'
import { SearchView } from './features/search/SearchView'
import { TagsView } from './features/search/TagsView'
import { ApplicationSettingsPage } from './features/settings/ApplicationSettingsPage'
import { LockScreen, usePrivacyGuard } from './features/privacy/PrivacyGuard'
import { useSettings } from './hooks/use-settings'
import { appBridge } from './lib/bridge'
import type { AppInfo, DateFormat } from './types'
import type { Theme } from './types'

const navigation = [
  { label: 'Today', icon: Sun }, { label: 'Timeline', icon: Clock3 },
  { label: 'Calendar', icon: CalendarDays }, { label: 'All entries', icon: Archive },
  { label: 'Favorites', icon: Heart }, { label: 'Tags', icon: Tag }, { label: 'Search', icon: Search },
  { label: 'Trash', icon: Trash2 },
]

export default function App() {
  const { settings, ready, setTheme, toggleSidebar, updateSettings } = useSettings()
  const [page, setPage] = useState('Today')
  const [newEntrySignal, setNewEntrySignal] = useState(0)
  const [openEntryID, setOpenEntryID] = useState('')
  const [searchTag, setSearchTag] = useState('')
  const [appInfo, setAppInfo] = useState<AppInfo | null>(null)
  const privacy = usePrivacyGuard()
  const collapsed = settings.sidebarCollapsed
  const now = new Date()
  const dayLabel = formatTopDate(now, settings.dateFormat)
  useEffect(() => { void appBridge.getInfo().then(setAppInfo) }, [])
  const startEntry = useCallback(() => {
    setPage('All entries')
    setOpenEntryID('')
    setNewEntrySignal((value) => value + 1)
  }, [])
  const openEntry = useCallback((id: string) => { setOpenEntryID(id); setPage('All entries') }, [])
  const filterTag = useCallback((tag: string) => { setSearchTag(tag); setPage('Search') }, [])
  const showSearch = useCallback(() => { setSearchTag(''); setPage('Search') }, [])
  const showToday = useCallback(() => setPage('Today'), [])
  const showSettings = useCallback(() => setPage('Settings'), [])
  useGlobalShortcuts({ enabled:!privacy.status.locked, startEntry, toggleSidebar, showSearch, showToday, showSettings })
  const entryView = pageToEntryView(page)
  return (
    <div className="app-shell" data-ready={ready}>
      <aside className={collapsed ? 'sidebar collapsed' : 'sidebar'} aria-label="Primary navigation">
        <div className="brand-row">
          <div className="brand-mark"><SolaceMark /></div>
          {!collapsed && <div><strong>Solace</strong><span>Your days, kept close.</span></div>}
          <button className="icon-button collapse-button" onClick={toggleSidebar} aria-label="Toggle sidebar">{collapsed ? <Menu size={18} /> : <ChevronLeft size={18} />}</button>
        </div>
        <button className="new-entry" onClick={startEntry} aria-keyshortcuts="Control+N Meta+N" title="New entry (Ctrl/⌘+N)"><Plus size={17} />{!collapsed && 'New entry'}</button>
        <nav>
          {navigation.map(({ label, icon: Icon }) => (
            <button key={label} className={page === label ? 'nav-item active' : 'nav-item'} onClick={() => { setOpenEntryID(''); setSearchTag(''); setPage(label) }} title={collapsed ? label : undefined}>
              <Icon size={18} />{!collapsed && <span>{label}</span>}
            </button>
          ))}
        </nav>
        <button className={page === 'Settings' ? 'nav-item settings-link active' : 'nav-item settings-link'} onClick={() => setPage('Settings')}><Settings size={18} />{!collapsed && <span>Settings</span>}</button>
        {!collapsed && <div className="runtime-status"><span className={appInfo?.initialized ? 'status-dot connected' : 'status-dot'} />{appInfo?.initialized ? 'Local database ready' : 'Browser preview'}</div>}
      </aside>
      <main>
        <header className="topbar">
          <div><span className="eyebrow">{dayLabel}</span><h1>{page}</h1></div>
          <div className="global-actions"><div className="theme-switcher" aria-label="Theme">
            {(['light', 'system', 'dark'] as Theme[]).map((theme) => (
              <button key={theme} className={settings.theme === theme ? 'selected' : ''} onClick={() => setTheme(theme)} aria-label={`${theme} theme`}>
                {theme === 'light' ? <Sun size={16} /> : theme === 'dark' ? <Moon size={16} /> : <span className="system-icon">A</span>}
              </button>
            ))}
          </div>
          {privacy.status.enabled && <button className="topbar-lock" onClick={() => void privacy.lock()} title="Lock Solace (Ctrl+Shift+L)"><LockKeyhole size={16}/>Lock</button>}
          </div>
        </header>
        <section className={entryView ? 'content entry-content' : ['Calendar','Timeline','Search'].includes(page) ? 'content browse-content' : 'content'}>
          {page === 'Today' && <TodayDashboard onNew={startEntry} onOpenEntry={openEntry} />}
          {entryView && <EntryWorkspace key={`${entryView}-${openEntryID}`} view={entryView} preferredEntryID={openEntryID} newEntrySignal={newEntrySignal} onNewEntryHandled={() => setNewEntrySignal(0)} autosaveDelayMs={settings.autosaveDelayMs} dateFormat={settings.dateFormat} />}
          {page === 'Calendar' && <CalendarView onOpenEntry={openEntry} />}
          {page === 'Timeline' && <TimelineView onOpenEntry={openEntry} />}
          {page === 'Search' && <SearchView key={searchTag} initialTag={searchTag} onOpenEntry={openEntry} />}
          {page === 'Tags' && <TagsView onFilter={filterTag} />}
          {page === 'Settings' && <ApplicationSettingsPage settings={settings} appInfo={appInfo} updateSettings={updateSettings}/>} 
          {!entryView && !['Today','Calendar','Timeline','Search','Tags','Settings'].includes(page) && <EmptyPage page={page} />}
        </section>
      </main>
      <LockScreen status={privacy.status} ready={privacy.ready} onUnlocked={privacy.apply}/>
    </div>
  )
}

function useGlobalShortcuts({ enabled, startEntry, toggleSidebar, showSearch, showToday, showSettings }: {
  enabled:boolean; startEntry:()=>void; toggleSidebar:()=>void; showSearch:()=>void; showToday:()=>void; showSettings:()=>void
}) {
  useEffect(() => {
    const onKeyDown = (event:KeyboardEvent) => {
      if (!enabled || !(event.ctrlKey || event.metaKey) || event.altKey) return
      const key = event.key.toLowerCase()
      let handled = true
      if (key === 'n' && !event.shiftKey) startEntry()
      else if (key === 's' && !event.shiftKey) window.dispatchEvent(new Event('solace:save-entry'))
      else if (key === 'k' && !event.shiftKey) showSearch()
      else if (key === 't' && event.shiftKey) showToday()
      else if (key === 'b' && !event.shiftKey) toggleSidebar()
      else if (key === 'r' && event.shiftKey) window.dispatchEvent(new Event('solace:toggle-reading'))
      else if (key === ',' && !event.shiftKey) showSettings()
      else handled = false
      if (handled) event.preventDefault()
    }
    window.addEventListener('keydown',onKeyDown)
    return () => window.removeEventListener('keydown',onKeyDown)
  }, [enabled, showSearch, showSettings, showToday, startEntry, toggleSidebar])
}

function formatTopDate(date:Date, format:DateFormat) {
  if (format === 'iso') return `${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,'0')}-${String(date.getDate()).padStart(2,'0')}`
  if (format === 'long') return new Intl.DateTimeFormat(undefined,{ weekday:'long', year:'numeric', month:'long', day:'numeric' }).format(date)
  return new Intl.DateTimeFormat(undefined,{ weekday:'long', month:'long', day:'numeric' }).format(date)
}

function pageToEntryView(page: string): EntryView | null {
  if (page === 'All entries') return 'all'
  if (page === 'Favorites') return 'favorites'
  if (page === 'Trash') return 'trash'
  return null
}

function EmptyPage({ page }: { page: string }) {
  return <section className="empty-state page-empty"><div className="empty-icon"><Archive size={24} /></div><h2>{page} is ready for its next chapter.</h2><p>The application shell and navigation are in place. This feature arrives in its scheduled development part.</p></section>
}
