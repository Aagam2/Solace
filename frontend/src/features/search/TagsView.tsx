import { useEffect, useState } from 'react'
import { Check, Pencil, Plus, Search, Trash2, X } from 'lucide-react'
import type { TagSummary } from '../entries/types'
import { searchBridge } from './search-bridge'
import { friendlyError } from '../../lib/user-errors'
import { useConfirmation } from '../../components/ConfirmationDialog'

export function TagsView({ onFilter }: { onFilter:(tag:string)=>void }) {
  const confirmAction=useConfirmation()
  const [tags,setTags]=useState<TagSummary[]>([]), [newName,setNewName]=useState(''), [editing,setEditing]=useState(''), [editName,setEditName]=useState(''), [error,setError]=useState('')
  const load=async()=>{try{setTags(await searchBridge.listTags());setError('')}catch(reason){setError(friendlyError(reason,'Your tags couldn’t be loaded. Please try again.','tags'))}}
  useEffect(()=>{void load()},[])
  const create=async()=>{try{await searchBridge.createTag(newName);setNewName('');await load()}catch(reason){setError(friendlyError(reason,'That tag couldn’t be created. Check its name and try again.','create tag'))}}
  const rename=async(id:string)=>{try{await searchBridge.renameTag(id,editName);setEditing('');await load()}catch(reason){setError(friendlyError(reason,'That tag couldn’t be renamed. Check its name and try again.','rename tag'))}}
  const remove=async(tag:TagSummary)=>{if(!await confirmAction({title:'Delete this tag?',message:`Delete the “${tag.name}” tag? Entries using it will remain.`,confirmLabel:'Delete tag',tone:'danger'}))return;try{await searchBridge.deleteTag(tag.id);await load()}catch(reason){setError(friendlyError(reason,'That tag couldn’t be deleted. Please try again.','delete tag'))}}
  return <section className="tags-page"><div className="tag-create"><div><span className="eyebrow">Organize your archive</span><h2>Tags</h2><p>Tags are optional and never hardcoded. Renaming or deleting a tag updates every assigned entry.</p></div><div><input value={newName} onChange={(event)=>setNewName(event.target.value)} onKeyDown={(event)=>{if(event.key==='Enter')void create()}} placeholder="New tag name" aria-label="New tag name"/><button onClick={()=>void create()} disabled={!newName.trim()}><Plus size={15}/>Create tag</button></div></div>
    {error&&<div className="editor-error" role="alert">{error}</div>}
    <div className="tag-manager-list">{tags.length===0&&<div className="empty-state"><h2>No tags yet.</h2><p>Create one here or type tags while editing an entry.</p></div>}{tags.map((tag)=><div key={tag.id}>
      {editing===tag.id?<input autoFocus aria-label={`Rename ${tag.name}`} value={editName} onChange={(event)=>setEditName(event.target.value)} onKeyDown={(event)=>{if(event.key==='Enter')void rename(tag.id)}}/>:<><strong>{tag.name}</strong><span>{tag.entryCount} {tag.entryCount===1?'entry':'entries'}</span></>}
      <div>{editing===tag.id?<><button aria-label="Save tag name" onClick={()=>void rename(tag.id)}><Check size={15}/></button><button aria-label="Cancel rename" onClick={()=>setEditing('')}><X size={15}/></button></>:<><button onClick={()=>onFilter(tag.name)}><Search size={14}/>Filter</button><button aria-label={`Rename ${tag.name}`} onClick={()=>{setEditing(tag.id);setEditName(tag.name)}}><Pencil size={14}/></button><button className="danger" aria-label={`Delete ${tag.name}`} onClick={()=>void remove(tag)}><Trash2 size={14}/></button></>}</div>
    </div>)}</div>
  </section>
}
