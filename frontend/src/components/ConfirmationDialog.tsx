import { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react'
import type { KeyboardEvent, ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'

export type ConfirmationOptions = {
  title:string
  message:string
  confirmLabel:string
  cancelLabel?:string
  tone?:'default' | 'danger'
}

const ConfirmationContext = createContext<((options:ConfirmationOptions)=>Promise<boolean>) | null>(null)

export function ConfirmationProvider({ children }: { children:ReactNode }) {
  const [pending, setPending] = useState<ConfirmationOptions | null>(null)
  const resolver = useRef<((confirmed:boolean)=>void) | null>(null)
  const cancelButton = useRef<HTMLButtonElement>(null)
  const confirmButton = useRef<HTMLButtonElement>(null)
  const returnFocus = useRef<HTMLElement | null>(null)

  const settle = useCallback((confirmed:boolean) => {
    const resolve = resolver.current
    resolver.current = null
    setPending(null)
    resolve?.(confirmed)
    window.requestAnimationFrame(() => returnFocus.current?.focus())
  }, [])

  const requestConfirmation = useCallback((options:ConfirmationOptions) => new Promise<boolean>((resolve) => {
    resolver.current?.(false)
    resolver.current = resolve
    returnFocus.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    setPending(options)
  }), [])

  useEffect(() => () => resolver.current?.(false), [])

  useEffect(() => {
    if (!pending) return
    cancelButton.current?.focus()
    const escape = (event:globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') { event.preventDefault(); settle(false) }
    }
    window.addEventListener('keydown',escape)
    return () => window.removeEventListener('keydown',escape)
  }, [pending, settle])

  const keepFocusInside = (event:KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'Tab') return
    if (event.shiftKey && document.activeElement === cancelButton.current) { event.preventDefault(); confirmButton.current?.focus() }
    else if (!event.shiftKey && document.activeElement === confirmButton.current) { event.preventDefault(); cancelButton.current?.focus() }
  }

  return <ConfirmationContext.Provider value={requestConfirmation}>
    {children}
    {pending && <div className="confirmation-backdrop" onMouseDown={(event) => { if (event.target === event.currentTarget) settle(false) }}>
      <div className="confirmation-dialog" role="alertdialog" aria-modal="true" aria-labelledby="confirmation-title" aria-describedby="confirmation-message" onKeyDown={keepFocusInside}>
        <div className={pending.tone === 'danger' ? 'confirmation-icon danger' : 'confirmation-icon'}><AlertTriangle size={20} aria-hidden="true"/></div>
        <div className="confirmation-copy"><h2 id="confirmation-title">{pending.title}</h2><p id="confirmation-message">{pending.message}</p></div>
        <div className="confirmation-actions">
          <button ref={cancelButton} type="button" className="confirmation-cancel" onClick={() => settle(false)}>{pending.cancelLabel ?? 'Cancel'}</button>
          <button ref={confirmButton} type="button" className={pending.tone === 'danger' ? 'confirmation-submit danger' : 'confirmation-submit'} onClick={() => settle(true)}>{pending.confirmLabel}</button>
        </div>
      </div>
    </div>}
  </ConfirmationContext.Provider>
}

export function useConfirmation() {
  const request = useContext(ConfirmationContext)
  if (!request) throw new Error('useConfirmation must be used inside ConfirmationProvider')
  return request
}
