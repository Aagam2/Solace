import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { beforeEach, expect, test } from 'vitest'
import App from './App'
import { LockScreen } from './features/privacy/PrivacyGuard'

beforeEach(() => {
  localStorage.clear()
  window.matchMedia = () => ({ matches:false, addEventListener(){}, removeEventListener(){} }) as unknown as MediaQueryList
})

test('renders the initial application shell and navigates', async () => {
  render(<App />)
  expect(screen.getByRole('heading', { name:'Good evening.' })).toBeInTheDocument()
  expect(await screen.findByRole('tab', { name:/Today’s pages/ })).toHaveAttribute('aria-selected', 'true')
  expect(screen.queryByRole('region', { name:'Journal entries' })).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name:'Timeline' }))
  expect(await screen.findByRole('heading', { name:/No entries in this month/ })).toBeInTheDocument()
})

test('switches between writing and entry details without leaving the editor', async () => {
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'New entry' }))
  expect(await screen.findByRole('tab', { name:'Writing' })).toHaveAttribute('aria-selected', 'true')
  expect(screen.getByRole('textbox', { name:'Entry content' })).toBeVisible()

  fireEvent.click(screen.getByRole('tab', { name:'Details & media' }))
  expect(screen.getByRole('tab', { name:'Details & media' })).toHaveAttribute('aria-selected', 'true')
  expect(screen.getByLabelText('Start date')).toBeVisible()
  expect(screen.queryByRole('textbox', { name:'Entry content' })).not.toBeInTheDocument()

  fireEvent.click(screen.getByRole('tab', { name:'Writing' }))
  expect(screen.getByRole('textbox', { name:'Entry content' })).toBeVisible()
})

test('shows a multi-day entry in calendar and opens it from the timeline', async () => {
  const date = new Date()
  const start = localDate(date)
  date.setDate(date.getDate() + 1)
  const end = localDate(date)
  localStorage.setItem('solace.entries', JSON.stringify([{ id:'memory-1', title:'Two day memory', contentJSON:'{"type":"doc","content":[{"type":"paragraph"}]}', contentText:'A journey across two days.', startDate:start, endDate:end, createdAt:new Date().toISOString(), updatedAt:new Date().toISOString(), mood:'Happy', favorite:false, pinned:true, draft:false, locationText:'', displayOrder:null, deletedAt:'', version:1, tags:[] }]))
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'Calendar' }))
  expect(await screen.findByText('2 dated pages')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name:'Timeline' }))
  const memory = await screen.findAllByRole('button', { name:/Two day memory/ })
  fireEvent.click(memory[0])
  expect(await screen.findByRole('textbox', { name:'Entry title' })).toHaveValue('Two day memory')
})

test('persists theme selection', async () => {
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'dark theme' }))
  await waitFor(() => expect(document.documentElement.dataset.theme).toBe('dark'))
  expect(localStorage.getItem('solace.settings')).toContain('dark')
})

test('creates and autosaves an entry in browser preview', async () => {
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'New entry' }))
  const title = await screen.findByRole('textbox', { name:'Entry title' })
  const content = screen.getByRole('textbox', { name:'Entry content' })
  fireEvent.change(title, { target:{ value:'A remembered morning' } })
  content.innerHTML = '<p>Coffee and rain.</p>'
  fireEvent.input(content, { inputType:'insertText', data:'Coffee and rain.' })

  await waitFor(() => expect(screen.getByText('Saved')).toBeInTheDocument(), { timeout:2000 })
  const stored = JSON.parse(localStorage.getItem('solace.entries') ?? '[]') as Array<{ title:string; contentText:string; version:number }>
  expect(stored[0]).toMatchObject({ title:'A remembered morning', contentText:'Coffee and rain.', version:2 })
  expect(stored[0]).toMatchObject({ contentJSON:expect.stringContaining('Coffee and rain.') })
})

test('manual save cancels the pending autosave without a stale write', async () => {
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'New entry' }))
  const title = await screen.findByRole('textbox', { name:'Entry title' })
  fireEvent.change(title, { target:{ value:'Saved once' } })
  fireEvent.click(screen.getByRole('button', { name:'Save' }))
  await waitFor(() => expect(screen.getByText('Saved')).toBeInTheDocument())
  await new Promise((resolve) => window.setTimeout(resolve, 800))
  const stored = JSON.parse(localStorage.getItem('solace.entries') ?? '[]') as Array<{ version:number }>
  expect(stored[0].version).toBe(2)
  expect(screen.queryByText('Save failed')).not.toBeInTheDocument()
})

test('searches partial words and opens the matching entry', async () => {
  localStorage.setItem('solace.entries', JSON.stringify([
    previewEntry('career-1','Career day','Had an interview today.',['Work']),
    previewEntry('career-2','Preparation','Prepared for a technical interview.',['Work']),
    previewEntry('book-1','Bookstore','Went to the bookstore.',['Books']),
  ]))
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'Search' }))
  fireEvent.change(screen.getByRole('textbox', { name:'Search journal' }), { target:{ value:'interv' } })
  expect(await screen.findByText('2 results')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name:/Career day/ }))
  expect(await screen.findByRole('textbox', { name:'Entry title' })).toHaveValue('Career day')
})

test('renames a tag across assigned browser-preview entries', async () => {
  localStorage.setItem('solace.entries', JSON.stringify([previewEntry('book-1','Bookstore','Went to the bookstore.',['Books'])]))
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'Tags' }))
  fireEvent.click(await screen.findByRole('button', { name:'Rename Books' }))
  fireEvent.change(screen.getByRole('textbox', { name:'Rename Books' }), { target:{ value:'Reading' } })
  fireEvent.click(screen.getByRole('button', { name:'Save tag name' }))
  expect(await screen.findByText('Reading')).toBeInTheDocument()
  const stored = JSON.parse(localStorage.getItem('solace.entries') ?? '[]') as Array<{ tags:string[] }>
  expect(stored[0].tags).toEqual(['Reading'])
})

test('imports a pasted image into the current entry', async () => {
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'New entry' }))
  const content = await screen.findByRole('textbox', { name:'Entry content' })
  const image = new File(['image bytes'], 'pasted-memory.png', { type:'image/png' })
  fireEvent.paste(content, { clipboardData:{ files:[image], types:['Files'], getData:() => '', setData:() => true } })

  await screen.findByText('pasted-memory.png')
  const stored = JSON.parse(localStorage.getItem('solace.attachments') ?? '[]') as Array<{ originalFilename:string; mimeType:string }>
  expect(stored[0]).toMatchObject({ originalFilename:'pasted-memory.png', mimeType:'image/png' })
})

test('imports files dropped onto the media drop zone', async () => {
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'New entry' }))
  fireEvent.click(await screen.findByRole('tab', { name:'Details & media' }))
  const dropZone = screen.getByRole('region', { name:'Attachment drop zone' })
  const documentFile = new File(['remember this'], 'letter.txt', { type:'text/plain' })
  fireEvent.drop(dropZone, { dataTransfer:{ files:[documentFile], types:['Files'] } })

  await screen.findByText('letter.txt')
  expect(screen.getByText('1 attachment imported')).toBeInTheDocument()
})

test('keeps data management compact with export, restore, and backup tabs', async () => {
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'Settings' }))
  expect(await screen.findByRole('heading', { name:'Application settings' })).toBeInTheDocument()
  fireEvent.click(screen.getByRole('tab', { name:'Data & safety' }))
  expect(screen.getByRole('tab', { name:'Export' })).toHaveAttribute('aria-selected', 'true')
  fireEvent.click(screen.getByRole('tab', { name:'Restore' }))
  expect(screen.getByRole('heading', { name:'Restore a complete archive' })).toBeVisible()
  fireEvent.click(screen.getByRole('tab', { name:'Backups' }))
  expect(screen.getByRole('heading', { name:'Keep a quiet safety net' })).toBeVisible()
  fireEvent.click(screen.getByRole('tab', { name:'Privacy & safety' }))
  expect(screen.getByRole('heading', { name:'Privacy & data safety' })).toBeVisible()
  fireEvent.click(screen.getByRole('button', { name:'Run check' }))
  expect(await screen.findByText('Everything looks healthy')).toBeVisible()
})

test('supports global desktop shortcuts for creating, saving, searching, and settings', async () => {
  render(<App />)
  fireEvent.keyDown(window, { key:'n', ctrlKey:true })
  const title = await screen.findByRole('textbox', { name:'Entry title' })
  fireEvent.change(title, { target:{ value:'Keyboard memory' } })
  fireEvent.keyDown(window, { key:'s', ctrlKey:true })
  await waitFor(() => expect(screen.getByText('Saved')).toBeInTheDocument())

  fireEvent.keyDown(window, { key:'k', ctrlKey:true })
  expect(await screen.findByRole('textbox', { name:'Search journal' })).toHaveFocus()
  fireEvent.keyDown(window, { key:',', ctrlKey:true })
  expect(await screen.findByRole('heading', { name:'Application settings' })).toBeVisible()
})

test('persists writing preferences and applies them to the editor surface', async () => {
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'Settings' }))
  fireEvent.click(await screen.findByRole('tab', { name:'Writing' }))
  fireEvent.change(screen.getByLabelText(/Text size/), { target:{ value:'20' } })
  fireEvent.click(screen.getByRole('button', { name:/Wide/ }))
  fireEvent.change(screen.getByLabelText('Autosave pace'), { target:{ value:'1500' } })

  await waitFor(() => expect(document.documentElement.style.getPropertyValue('--editor-font-size')).toBe('20px'))
  const stored = JSON.parse(localStorage.getItem('solace.settings') ?? '{}') as { editorFontSize:number; editorWidth:string; autosaveDelayMs:number }
  expect(stored).toMatchObject({ editorFontSize:20, editorWidth:'wide', autosaveDelayMs:1500 })
})

test('shows live writing counts and exits distraction-free mode with Escape', async () => {
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'New entry' }))
  const content = await screen.findByRole('textbox', { name:'Entry content' })
  content.innerHTML = '<p>Two words</p>'
  fireEvent.input(content, { inputType:'insertText', data:'Two words' })
  expect(await screen.findByText(/2 words · 9 characters/)).toBeVisible()
  fireEvent.click(screen.getByRole('button', { name:/Focus/ }))
  expect(screen.getByRole('region', { name:'Journal entries' })).toHaveClass('focus-mode')
  fireEvent.keyDown(window, { key:'Escape' })
  expect(screen.getByRole('region', { name:'Journal entries' })).not.toHaveClass('focus-mode')
})

test('keeps bounded saved versions and restores an earlier entry revision', async () => {
  window.confirm = () => true
  render(<App />)
  fireEvent.click(screen.getByRole('button', { name:'New entry' }))
  const title = await screen.findByRole('textbox', { name:'Entry title' })
  fireEvent.change(title, { target:{ value:'First saved title' } })
  fireEvent.click(screen.getByRole('button', { name:'Save' }))
  await waitFor(() => expect(screen.getByText('Saved')).toBeInTheDocument())
  fireEvent.change(title, { target:{ value:'Second saved title' } })
  fireEvent.click(screen.getByRole('button', { name:'Save' }))
  await waitFor(() => expect(screen.getByText('Saved')).toBeInTheDocument())

  fireEvent.click(screen.getByRole('tab', { name:'History' }))
  const earlier = await screen.findByText('First saved title')
  const revision = earlier.closest('article')
  expect(revision).not.toBeNull()
  fireEvent.click(within(revision as HTMLElement).getByRole('button', { name:'Restore' }))
  expect(await screen.findByRole('textbox', { name:'Entry title' })).toHaveValue('First saved title')
})

test('labels the application lock as a privacy screen rather than encryption', () => {
  render(<LockScreen status={{ enabled:true, locked:true, lockOnStart:true, inactivityMinutes:5 }} ready onUnlocked={() => undefined}/>)
  expect(screen.getByRole('dialog', { name:'Solace is locked' })).toBeInTheDocument()
  expect(screen.getByText(/does not encrypt the database or media files/i)).toBeVisible()
  expect(screen.getByLabelText('Password')).toHaveAttribute('type', 'password')
})

function localDate(date: Date) { return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
function previewEntry(id:string,title:string,contentText:string,tags:string[]) { const date=localDate(new Date()); return { id,title,contentJSON:'{"type":"doc","content":[{"type":"paragraph"}]}',contentText,startDate:date,endDate:date,createdAt:new Date().toISOString(),updatedAt:new Date().toISOString(),mood:'',favorite:false,pinned:false,draft:false,locationText:'',displayOrder:null,deletedAt:'',version:1,tags } }
