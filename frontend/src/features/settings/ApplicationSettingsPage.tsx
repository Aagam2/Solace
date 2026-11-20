import { useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { DatabaseBackup, Info, Keyboard, Monitor, Moon, PenLine, Settings2, Sun } from 'lucide-react'
import { DataManagementPage } from '../data/DataManagementPage'
import type { AppInfo, Settings, Theme } from '../../types'

type SettingsTab = 'general' | 'writing' | 'data' | 'about'

const tabs: Array<{ id:SettingsTab; label:string; icon:typeof Settings2 }> = [
  { id:'general', label:'General', icon:Settings2 },
  { id:'writing', label:'Writing', icon:PenLine },
  { id:'data', label:'Data & safety', icon:DatabaseBackup },
  { id:'about', label:'About', icon:Info },
]

export function ApplicationSettingsPage({ settings, appInfo, updateSettings }: {
  settings:Settings
  appInfo:AppInfo | null
  updateSettings:(patch:Partial<Settings>)=>void
}) {
  const [tab, setTab] = useState<SettingsTab>('general')
  const tabRefs = useRef<Record<SettingsTab, HTMLButtonElement | null>>({ general:null, writing:null, data:null, about:null })
  const selectTab = (next:SettingsTab) => { setTab(next); tabRefs.current[next]?.focus() }
  const onTabKey = (event:KeyboardEvent, current:SettingsTab) => {
    if (!['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return
    event.preventDefault()
    const index = tabs.findIndex((item) => item.id === current)
    const nextIndex = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (index + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length
    selectTab(tabs[nextIndex].id)
  }
  return <section className="settings-page" aria-label="Application settings">
    <div className="settings-workspace">
      <div className="settings-tabs" role="tablist" aria-label="Settings sections">
        {tabs.map(({ id, label, icon:Icon }) => <button key={id} ref={(node) => { tabRefs.current[id] = node }} role="tab" aria-selected={tab === id} aria-controls={`settings-panel-${id}`} tabIndex={tab === id ? 0 : -1} onKeyDown={(event) => onTabKey(event,id)} onClick={() => setTab(id)}><Icon size={16}/>{label}</button>)}
      </div>
      <GeneralPanel hidden={tab !== 'general'} settings={settings} updateSettings={updateSettings}/>
      <WritingPanel hidden={tab !== 'writing'} settings={settings} updateSettings={updateSettings}/>
      <div id="settings-panel-data" role="tabpanel" hidden={tab !== 'data'}><DataManagementPage compact settings={settings} updateSettings={updateSettings}/></div>
      <AboutPanel hidden={tab !== 'about'} appInfo={appInfo}/>
    </div>
  </section>
}

function GeneralPanel({ hidden, settings, updateSettings }: PanelProps) {
  const themes:Array<{ id:Theme; label:string; note:string; icon:typeof Sun }> = [
    { id:'light', label:'Light', note:'Warm paper surfaces', icon:Sun },
    { id:'system', label:'System', note:'Follow your desktop', icon:Monitor },
    { id:'dark', label:'Dark', note:'Quiet charcoal tones', icon:Moon },
  ]
  return <div id="settings-panel-general" role="tabpanel" hidden={hidden} className="settings-panel">
    <div className="settings-panel-heading"><h3>Appearance & dates</h3><p>These preferences apply immediately and stay on this device.</p></div>
    <fieldset className="settings-fieldset"><legend>Theme</legend><div className="setting-choice-grid theme-choices">{themes.map(({ id,label,note,icon:Icon }) => <button type="button" key={id} aria-pressed={settings.theme === id} onClick={() => updateSettings({ theme:id })}><Icon size={19}/><span><strong>{label}</strong><small>{note}</small></span></button>)}</div></fieldset>
    <label className="settings-select-row"><span><strong>Date display</strong><small>Changes journal dates without changing stored ISO dates.</small></span><select aria-label="Date display format" value={settings.dateFormat} onChange={(event) => updateSettings({ dateFormat:event.target.value as Settings['dateFormat'] })}><option value="system">System preference</option><option value="long">Long · November 18, 2026</option><option value="iso">ISO · 2026-11-18</option></select></label>
  </div>
}

function WritingPanel({ hidden, settings, updateSettings }: PanelProps) {
  return <div id="settings-panel-writing" role="tabpanel" hidden={hidden} className="settings-panel writing-settings">
    <div className="settings-panel-heading"><h3>Writing</h3><p>Adjust the page without changing the content saved in your journal.</p></div>
    <div className="writing-settings-grid">
      <div className="setting-control"><label htmlFor="editor-font-size"><strong>Text size</strong><small>{settings.editorFontSize}px</small></label><input id="editor-font-size" type="range" min="14" max="24" step="1" value={settings.editorFontSize} onChange={(event) => updateSettings({ editorFontSize:Number(event.target.value) })}/></div>
      <label className="setting-control"><span><strong>Line spacing</strong><small>Give longer entries room to breathe.</small></span><select aria-label="Editor line spacing" value={settings.editorLineSpacing} onChange={(event) => updateSettings({ editorLineSpacing:Number(event.target.value) })}><option value="1.5">Compact</option><option value="1.75">Comfortable</option><option value="1.9">Airy</option><option value="2.1">Very airy</option></select></label>
      <label className="setting-control"><span><strong>Autosave pace</strong><small>Changes still save immediately when you press the shortcut.</small></span><select aria-label="Autosave pace" value={settings.autosaveDelayMs} onChange={(event) => updateSettings({ autosaveDelayMs:Number(event.target.value) })}><option value="400">Quick · 0.4 seconds</option><option value="700">Balanced · 0.7 seconds</option><option value="1500">Relaxed · 1.5 seconds</option></select></label>
    </div>
    <fieldset className="settings-fieldset"><legend>Page width</legend><div className="setting-choice-grid width-choices">{(['narrow','comfortable','wide'] as const).map((width) => <button type="button" key={width} aria-pressed={settings.editorWidth === width} onClick={() => updateSettings({ editorWidth:width })}><span className={`width-swatch ${width}`}/><span><strong>{capitalize(width)}</strong><small>{width === 'narrow' ? 'Focused, book-like line length' : width === 'wide' ? 'More room for large windows' : 'Balanced for everyday writing'}</small></span></button>)}</div></fieldset>
    <div className="writing-preview" style={{ fontSize:settings.editorFontSize, lineHeight:settings.editorLineSpacing }}><span>Preview</span><p>A quiet page leaves enough room for the thought itself.</p></div>
  </div>
}

function AboutPanel({ hidden, appInfo }: { hidden:boolean; appInfo:AppInfo | null }) {
  const shortcuts = shortcutRows()
  return <div id="settings-panel-about" role="tabpanel" hidden={hidden} className="settings-panel about-settings">
    <div className="settings-panel-heading"><h3>About Solace</h3><p>A private, local-first journal that works without an internet connection.</p></div>
    <dl className="app-facts"><div><dt>Version</dt><dd>{appInfo?.version ?? '0.1.0'}</dd></div><div><dt>Runtime</dt><dd>{appInfo?.initialized ? 'Native desktop' : 'Browser preview'}</dd></div><div><dt>Data policy</dt><dd>Local and offline by default</dd></div></dl>
    <div className="storage-location"><div><DatabaseBackup size={18}/><span><strong>Storage location</strong><small>Shown for backup and troubleshooting. Moving this folder here is intentionally disabled because relocation requires a verified migration.</small></span></div><code>{appInfo?.dataPaths.root ?? 'Loading local data path…'}</code>{appInfo?.dataPaths.database && <code>{appInfo.dataPaths.database}</code>}</div>
    <div className="shortcut-reference"><div><Keyboard size={18}/><span><strong>Keyboard shortcuts</strong><small>Platform-appropriate shortcuts work from anywhere in Solace.</small></span></div><dl>{shortcuts.map(([label,keys]) => <div key={label}><dt>{label}</dt><dd>{keys.split('+').map((key) => <kbd key={key}>{key}</kbd>)}</dd></div>)}</dl></div>
  </div>
}

type PanelProps = { hidden:boolean; settings:Settings; updateSettings:(patch:Partial<Settings>)=>void }
function capitalize(value:string) { return value.charAt(0).toUpperCase() + value.slice(1) }
function shortcutRows():Array<[string,string]> {
  const mod = /Mac|iPhone|iPad/.test(navigator.platform) ? '⌘' : 'Ctrl'
  return [['New entry',`${mod}+N`],['Save entry',`${mod}+S`],['Search',`${mod}+K`],['Today',`${mod}+Shift+T`],['Toggle sidebar',`${mod}+B`],['Reading mode',`${mod}+Shift+R`],['Settings',`${mod}+,`]]
}
