import { useEffect } from 'react'
import type { ReactNode } from 'react'
import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import Image from '@tiptap/extension-image'
import {
  Bold, Heading2, ImageIcon, Italic, Link, List, ListChecks, ListOrdered, Minus,
  Quote, Redo2, Strikethrough, Underline, Undo2,
} from 'lucide-react'

interface InsertImageRequest { key: number; url: string; alt: string }

export function RichTextEditor({ contentJSON, readOnly, insertImage, onChange, onFiles }: {
  contentJSON: string
  readOnly: boolean
  insertImage: InsertImageRequest | null
  onChange: (contentJSON: string, contentText: string) => void
  onFiles: (files: File[]) => void
}) {
  const editor = useEditor({
    immediatelyRender:false,
    editable:!readOnly,
    extensions:[
      StarterKit.configure({
        heading:{ levels:[1, 2, 3] },
        link:{ openOnClick:false, autolink:true, defaultProtocol:'https' },
      }),
      TaskList,
      TaskItem.configure({ nested:true }),
      Image.configure({ allowBase64:false, inline:false }),
    ],
    content:parseDocument(contentJSON),
    editorProps:{
      attributes:{ class:'rich-text-surface', 'aria-label':'Entry content' },
      handlePaste:(_view, event) => {
        const files = Array.from(event.clipboardData?.files ?? [])
        if (!files.length) return false
        event.preventDefault(); onFiles(files); return true
      },
      handleDrop:(_view, event) => {
        const files = Array.from(event.dataTransfer?.files ?? [])
        if (!files.length) return false
        event.preventDefault(); onFiles(files); return true
      },
    },
    onUpdate:({ editor:updated }) => onChange(JSON.stringify(updated.getJSON()), updated.getText({ blockSeparator:'\n' })),
  }, [onFiles])

  useEffect(() => { editor?.setEditable(!readOnly) }, [editor, readOnly])
  useEffect(() => {
    if (!editor || editor.isDestroyed) return
    const next = parseDocument(contentJSON)
    if (JSON.stringify(editor.getJSON()) !== JSON.stringify(next)) editor.commands.setContent(next, { emitUpdate:false })
  }, [contentJSON, editor])
  useEffect(() => {
    if (editor && insertImage) editor.chain().focus().setImage({ src:insertImage.url, alt:insertImage.alt, title:insertImage.alt }).run()
  }, [editor, insertImage])

  if (!editor) return <div className="rich-text-loading">Preparing editor…</div>

  const setLink = () => {
    const current = editor.getAttributes('link').href as string | undefined
    const href = window.prompt('Link address', current ?? 'https://')
    if (href === null) return
    if (!href.trim()) editor.chain().focus().extendMarkRange('link').unsetLink().run()
    else editor.chain().focus().extendMarkRange('link').setLink({ href:href.trim() }).run()
  }
  const button = (label: string, active: boolean, action: () => void, icon: ReactNode) => (
    <button type="button" aria-label={label} title={label} className={active ? 'active' : ''} onClick={action} disabled={readOnly}>{icon}</button>
  )

  return <div className="rich-text-editor">
    <div className="rich-text-toolbar" aria-label="Text formatting">
      {button('Heading', editor.isActive('heading', { level:2 }), () => editor.chain().focus().toggleHeading({ level:2 }).run(), <Heading2 size={16}/>)}
      {button('Bold', editor.isActive('bold'), () => editor.chain().focus().toggleBold().run(), <Bold size={16}/>)}
      {button('Italic', editor.isActive('italic'), () => editor.chain().focus().toggleItalic().run(), <Italic size={16}/>)}
      {button('Underline', editor.isActive('underline'), () => editor.chain().focus().toggleUnderline().run(), <Underline size={16}/>)}
      {button('Strikethrough', editor.isActive('strike'), () => editor.chain().focus().toggleStrike().run(), <Strikethrough size={16}/>)}
      <span className="toolbar-divider"/>
      {button('Bullet list', editor.isActive('bulletList'), () => editor.chain().focus().toggleBulletList().run(), <List size={16}/>)}
      {button('Numbered list', editor.isActive('orderedList'), () => editor.chain().focus().toggleOrderedList().run(), <ListOrdered size={16}/>)}
      {button('Checklist', editor.isActive('taskList'), () => editor.chain().focus().toggleTaskList().run(), <ListChecks size={16}/>)}
      {button('Blockquote', editor.isActive('blockquote'), () => editor.chain().focus().toggleBlockquote().run(), <Quote size={16}/>)}
      {button('Link', editor.isActive('link'), setLink, <Link size={16}/>)}
      {button('Horizontal rule', false, () => editor.chain().focus().setHorizontalRule().run(), <Minus size={16}/>)}
      <span className="toolbar-divider"/>
      {button('Undo', false, () => editor.chain().focus().undo().run(), <Undo2 size={16}/>)}
      {button('Redo', false, () => editor.chain().focus().redo().run(), <Redo2 size={16}/>)}
      <span className="toolbar-hint"><ImageIcon size={14}/>Paste or drop images</span>
    </div>
    <EditorContent editor={editor}/>
  </div>
}

function parseDocument(contentJSON: string) {
  try {
    const parsed = JSON.parse(contentJSON) as { type?: string }
    if (parsed.type === 'doc') return parsed
  } catch { /* fall through to an empty document */ }
  return { type:'doc', content:[{ type:'paragraph' }] }
}
