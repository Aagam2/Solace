import { useCallback, useEffect, useState } from 'react'
import { Image, RefreshCw, Search, Video } from 'lucide-react'
import type { SearchOptions, SearchPage, TagSummary } from '../entries/types'
import { searchBridge } from './search-bridge'
import { friendlyError } from '../../lib/user-errors'

const emptySearch = (tag = ''): SearchOptions => ({ query:'', startDate:'', endDate:'', tags:tag ? [tag] : [], mood:'', favoritesOnly:false, hasImages:false, hasVideos:false, draftStatus:'any', sort:'relevance', limit:50, offset:0 })

export function SearchView({ initialTag, onOpenEntry }: { initialTag?: string; onOpenEntry:(id:string)=>void }) {
  const [options, setOptions] = useState<SearchOptions>(() => emptySearch(initialTag))
  const [page, setPage] = useState<SearchPage>({ results:[], total:0, hasMore:false })
  const [tags, setTags] = useState<TagSummary[]>([])
  const [error, setError] = useState('')
  const [searching, setSearching] = useState(false)
  const run = useCallback(async (next: SearchOptions, append = false) => {
    setSearching(true)
    try { const result = await searchBridge.search(next); setPage((current) => append ? { ...result, results:[...current.results, ...result.results] } : result); setError('') }
    catch (reason) { setError(friendlyError(reason, 'Search is unavailable right now. Please try again.', 'journal search')) }
    finally { setSearching(false) }
  }, [])
  useEffect(() => { void searchBridge.listTags().then(setTags).catch((reason) => setError(friendlyError(reason, 'Your tags couldn’t be loaded. Please try again.', 'search tags'))) }, [])
  useEffect(() => { const timer = window.setTimeout(() => void run({ ...options, offset:0 }), 250); return () => window.clearTimeout(timer) }, [options, run])
  const edit = <K extends keyof SearchOptions>(key: K, value: SearchOptions[K]) => setOptions((current) => ({ ...current, [key]:value, offset:0 }))
  const toggleTag = (name: string) => edit('tags', options.tags.some((tag) => same(tag,name)) ? options.tags.filter((tag) => !same(tag,name)) : [...options.tags, name])
  const rebuild = async () => { try { await searchBridge.rebuild(); await run({ ...options, offset:0 }); setError('') } catch (reason) { setError(friendlyError(reason, 'The search index couldn’t be rebuilt. Your entries are unchanged.', 'rebuild search')) } }
  return <section className="search-page">
    <div className="search-box"><Search size={20}/><input autoFocus value={options.query} onChange={(event) => edit('query',event.target.value)} placeholder="Search titles, writing, tags, and captions" aria-label="Search journal"/>{searching && <span>Searching…</span>}</div>
    <div className="search-layout">
      <aside className="search-filters">
        <div className="filter-heading"><strong>Filters</strong><button onClick={() => setOptions(emptySearch())}>Clear</button></div>
        <label>From<input type="date" value={options.startDate} onChange={(event) => edit('startDate',event.target.value)}/></label>
        <label>To<input type="date" min={options.startDate} value={options.endDate} onChange={(event) => edit('endDate',event.target.value)}/></label>
        <label>Mood<select value={options.mood} onChange={(event) => edit('mood',event.target.value)}><option value="">Any mood</option><option>Very Happy</option><option>Happy</option><option>Neutral</option><option>Sad</option><option>Very Sad</option></select></label>
        <label>Status<select value={options.draftStatus} onChange={(event) => edit('draftStatus',event.target.value as SearchOptions['draftStatus'])}><option value="any">Drafts and published</option><option value="draft">Draft only</option><option value="published">Published only</option></select></label>
        <label>Order<select value={options.sort} onChange={(event) => edit('sort',event.target.value as SearchOptions['sort'])}><option value="relevance">Relevance</option><option value="newest">Newest journal date</option><option value="oldest">Oldest journal date</option></select></label>
        <label className="filter-check"><input type="checkbox" checked={options.favoritesOnly} onChange={(event) => edit('favoritesOnly',event.target.checked)}/>Favorites only</label>
        <label className="filter-check"><input type="checkbox" checked={options.hasImages} onChange={(event) => edit('hasImages',event.target.checked)}/><Image size={14}/>Contains images</label>
        <label className="filter-check"><input type="checkbox" checked={options.hasVideos} onChange={(event) => edit('hasVideos',event.target.checked)}/><Video size={14}/>Contains videos</label>
        <div className="tag-filter"><span>Tags</span>{tags.length === 0 && <p>No tags yet.</p>}{tags.map((tag) => <button key={tag.id} className={options.tags.some((name) => same(name,tag.name)) ? 'selected' : ''} onClick={() => toggleTag(tag.name)}>{tag.name}<small>{tag.entryCount}</small></button>)}</div>
        <button className="rebuild-button" onClick={() => void rebuild()}><RefreshCw size={14}/>Rebuild search index</button>
      </aside>
      <div className="search-results">
        <div className="results-heading"><span>{page.total} result{page.total === 1 ? '' : 's'}</span>{options.startDate || options.endDate ? <small>Date filters match overlapping entry ranges.</small> : null}</div>
        {error && <div className="editor-error" role="alert">{error}</div>}
        {!searching && !page.results.length && <div className="empty-state"><h2>No matching entries.</h2><p>Try a partial word or remove one of the filters.</p></div>}
        {page.results.map((result) => <button className="search-result" key={result.entry.id} onClick={() => onOpenEntry(result.entry.id)}>
          <div className="search-result-top"><h3><Highlighted value={result.highlightedTitle || result.entry.title || 'Untitled entry'}/></h3><span>{formatRange(result.entry.startDate,result.entry.endDate)}</span></div>
          <p><Highlighted value={result.snippet || 'A blank page'}/></p>
          <div className="result-meta">{result.entry.mood && <span>{result.entry.mood}</span>}{result.entry.favorite && <span>Favorite</span>}{result.entry.draft && <span>Draft</span>}{result.tags.map((tag) => <span key={tag}>#{tag}</span>)}</div>
        </button>)}
        {page.hasMore && <button className="load-more" onClick={() => void run({...options,offset:page.results.length},true)}>Load more</button>}
      </div>
    </div>
  </section>
}

function Highlighted({ value }: { value:string }) { let marked=false; return <>{value.split(/(<mark>|<\/mark>)/g).filter(Boolean).map((part,index) => { if(part === '<mark>'){marked=true;return null} if(part === '</mark>'){marked=false;return null} return marked ? <mark key={index}>{part}</mark> : <span key={index}>{part}</span> })}</> }
function formatRange(start:string,end:string){return start===end?start:`${start} – ${end}`}
function same(left:string,right:string){return left.localeCompare(right,undefined,{sensitivity:'accent'})===0}
