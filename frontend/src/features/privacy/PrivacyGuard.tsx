import { useCallback, useEffect, useRef, useState } from 'react'
import type { KeyboardEvent as ReactKeyboardEvent } from 'react'
import { LockKeyhole, ShieldCheck } from 'lucide-react'
import type { Status } from '../../../bindings/solace/internal/privacy/models'
import { announcePrivacyChange, privacyBridge, privacyChangedEvent } from './privacy-bridge'
import { friendlyError, reportClientError } from '../../lib/user-errors'

const initial: Status = { enabled:false, locked:false, lockOnStart:false, inactivityMinutes:0 }

export function usePrivacyGuard() {
  const [status, setStatus] = useState<Status>(initial)
  const [ready, setReady] = useState(false)
  const timer = useRef<number | null>(null)
  const apply = useCallback((next:Status) => { setStatus(next); setReady(true) }, [])
  const lock = useCallback(async () => { const next = await privacyBridge.lock(); apply(next); announcePrivacyChange(next) }, [apply])

  useEffect(() => { void privacyBridge.status().then(apply).catch((reason) => { reportClientError('privacy status', reason); setReady(true) }) }, [apply])
  useEffect(() => {
    const changed = (event:Event) => apply((event as CustomEvent<Status>).detail)
    window.addEventListener(privacyChangedEvent, changed)
    return () => window.removeEventListener(privacyChangedEvent, changed)
  }, [apply])
  useEffect(() => {
    if (timer.current !== null) window.clearTimeout(timer.current)
    if (!status.enabled || status.locked || status.inactivityMinutes === 0) return
    const schedule = () => {
      if (timer.current !== null) window.clearTimeout(timer.current)
      timer.current = window.setTimeout(() => void lock(), status.inactivityMinutes * 60_000)
    }
    const events = ['pointerdown','keydown','focus'] as const
    events.forEach((name) => window.addEventListener(name, schedule))
    schedule()
    return () => { events.forEach((name) => window.removeEventListener(name, schedule)); if (timer.current !== null) window.clearTimeout(timer.current) }
  }, [lock, status.enabled, status.inactivityMinutes, status.locked])
  useEffect(() => {
    const shortcut = (event:KeyboardEvent) => {
      if (status.enabled && !status.locked && event.ctrlKey && event.shiftKey && event.key.toLowerCase() === 'l') { event.preventDefault(); void lock() }
    }
    window.addEventListener('keydown', shortcut)
    return () => window.removeEventListener('keydown', shortcut)
  }, [lock, status.enabled, status.locked])

  return { status, ready, lock, apply }
}

export function LockScreen({ status, ready, onUnlocked }: { status:Status; ready:boolean; onUnlocked:(status:Status)=>void }) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const input = useRef<HTMLInputElement>(null)
  const panel = useRef<HTMLDivElement>(null)
  useEffect(() => { if (status.locked) window.setTimeout(() => input.current?.focus(), 0) }, [status.locked])
  if (ready && !status.locked) return null
  const unlock = async () => {
    setBusy(true); setError('')
    try { const next = await privacyBridge.unlock(password); setPassword(''); onUnlocked(next); announcePrivacyChange(next) }
    catch (reason) { setError(friendlyError(reason, 'Solace couldn’t be unlocked. Please try again.', 'unlock journal')); input.current?.select() }
    finally { setBusy(false) }
  }
  const trapFocus = (event:ReactKeyboardEvent) => {
    if (event.key !== 'Tab') return
    const controls = panel.current?.querySelectorAll<HTMLElement>('input,button:not(:disabled)')
    if (!controls?.length) return
    const first = controls[0], last = controls[controls.length - 1]
    if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
  }
  return <div className="lock-screen" role="dialog" aria-modal="true" aria-labelledby="lock-title" onKeyDown={trapFocus}>
    <div className="lock-panel" ref={panel}>
      <div className="lock-mark"><LockKeyhole size={25}/></div>
      <span className="eyebrow">Private writing room</span><h2 id="lock-title">Solace is locked</h2>
      <p>Your journal remains stored locally. This privacy screen prevents casual access; it does not encrypt the database or media files.</p>
      {!ready ? <div className="lock-loading" role="status">Preparing your journal…</div> : <form onSubmit={(event) => { event.preventDefault(); void unlock() }}>
        <label>Password<input ref={input} type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)}/></label>
        {error && <p className="operation-error" role="alert">{error}</p>}
        <button type="submit" disabled={busy || !password}><ShieldCheck size={16}/>{busy ? 'Unlocking…' : 'Unlock Solace'}</button>
      </form>}
    </div>
  </div>
}

