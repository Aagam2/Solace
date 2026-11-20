import { useEffect, useState } from 'react'
import { Check, HeartPulse, KeyRound, LockKeyhole, ShieldAlert, ShieldCheck } from 'lucide-react'
import type { DataHealth } from '../../../bindings/solace/internal/app/models'
import type { Status } from '../../../bindings/solace/internal/privacy/models'
import { appBridge } from '../../lib/bridge'
import { announcePrivacyChange, privacyBridge } from './privacy-bridge'
import { friendlyError } from '../../lib/user-errors'
import { useConfirmation } from '../../components/ConfirmationDialog'

export function PrivacyPanel({ hidden }: { hidden:boolean }) {
  const confirmAction = useConfirmation()
  const [status, setStatus] = useState<Status>({ enabled:false, locked:false, lockOnStart:false, inactivityMinutes:0 })
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [lockOnStart, setLockOnStart] = useState(false)
  const [inactivity, setInactivity] = useState(0)
  const [health, setHealth] = useState<DataHealth | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  useEffect(() => { if (!hidden) void privacyBridge.status().then((next) => { setStatus(next); setLockOnStart(next.lockOnStart); setInactivity(next.inactivityMinutes) }).catch((reason) => setError(friendlyError(reason, 'Privacy settings couldn’t be loaded. Please try again.', 'load privacy settings'))) }, [hidden])
  const configure = async () => {
    setError(''); setMessage('')
    if ((!status.enabled || newPassword) && newPassword !== confirmPassword) return setError('The new passwords do not match.')
    setBusy(true)
    try {
      const next = await privacyBridge.configure({ currentPassword, newPassword, lockOnStart, inactivityMinutes:inactivity })
      setStatus(next); setCurrentPassword(''); setNewPassword(''); setConfirmPassword(''); setMessage(status.enabled ? 'Privacy lock settings updated.' : 'Privacy lock enabled.'); announcePrivacyChange(next)
    } catch (reason) { setError(friendlyError(reason, 'Privacy settings couldn’t be updated. Please try again.', 'privacy settings')) }
    finally { setBusy(false) }
  }
  const disable = async () => {
    if (!await confirmAction({ title:'Disable the privacy lock?', message:'Solace will open without asking for a password. Your journal data will not be deleted.', confirmLabel:'Disable lock', tone:'danger' })) return
    setBusy(true); setError(''); setMessage('')
    try { const next = await privacyBridge.disable(currentPassword); setStatus(next); setCurrentPassword(''); setMessage('Privacy lock disabled.'); announcePrivacyChange(next) }
    catch (reason) { setError(friendlyError(reason, 'The privacy lock couldn’t be turned off. Please try again.', 'disable privacy lock')) }
    finally { setBusy(false) }
  }
  const lock = async () => { try { const next = await privacyBridge.lock(); announcePrivacyChange(next) } catch (reason) { setError(friendlyError(reason, 'Solace couldn’t be locked. Please try again.', 'lock journal')) } }
  const checkHealth = async () => { try { setHealth(await appBridge.checkDataHealth()) } catch (reason) { setError(friendlyError(reason, 'The safety check couldn’t be completed. Please try again.', 'journal health check')) } }
  return <div id="data-panel-privacy" role="tabpanel" hidden={hidden} className="data-panel privacy-panel">
    <div className="data-panel-heading"><div><h3>Privacy &amp; data safety</h3><p>Add a local privacy screen and verify the health of Solace’s database.</p></div><ShieldCheck size={23}/></div>
    <div className="privacy-warning"><ShieldAlert size={18}/><p><strong>A lock is not encryption.</strong> It prevents casual access while Solace is open, but the SQLite database and media files remain readable to someone with access to your operating-system account.</p></div>
    <fieldset disabled={busy || !privacyBridge.isNative()} className="privacy-form">
      {status.enabled && <label>Current password<input type="password" autoComplete="current-password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)}/></label>}
      <label>{status.enabled ? 'New password (optional)' : 'Password'}<input type="password" autoComplete="new-password" minLength={8} value={newPassword} onChange={(event) => setNewPassword(event.target.value)}/><small>At least 8 characters. Solace stores a salted derived hash, never the password.</small></label>
      <label>Confirm new password<input type="password" autoComplete="new-password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)}/></label>
      <label>Lock after inactivity<select value={inactivity} onChange={(event) => setInactivity(Number(event.target.value))}><option value="0">Never</option><option value="1">1 minute</option><option value="5">5 minutes</option><option value="15">15 minutes</option><option value="30">30 minutes</option><option value="60">1 hour</option></select></label>
      <label className="data-check"><input type="checkbox" checked={lockOnStart} onChange={(event) => setLockOnStart(event.target.checked)}/><span><strong>Lock when Solace starts</strong><small>Require the password each time the application opens.</small></span></label>
    </fieldset>
    <div className="privacy-actions">{status.enabled && <button className="secondary-action danger" onClick={() => void disable()} disabled={busy || !privacyBridge.isNative()}>Disable lock</button>}<span/>{status.enabled && <button className="secondary-action" onClick={() => void lock()} disabled={busy || !privacyBridge.isNative()}><LockKeyhole size={15}/>Lock now</button>}<button className="primary-action" onClick={() => void configure()} disabled={busy || !privacyBridge.isNative()}><KeyRound size={15}/>{status.enabled ? 'Save lock settings' : 'Enable privacy lock'}</button></div>
    {error && <p className="operation-error" role="alert">{error}</p>}{message && <p className="operation-success" role="status"><Check size={15}/>{message}</p>}
    {!privacyBridge.isNative() && <p className="preview-notice">Privacy lock configuration is available in the native Solace desktop window.</p>}
    <div className="health-row"><div><HeartPulse size={18}/><span><strong>Database health</strong><small>Checks SQLite integrity and foreign-key relationships without changing your data.</small></span></div><button className="secondary-action" onClick={() => void checkHealth()}><HeartPulse size={15}/>Run check</button></div>
    {health && <div className={health.healthy ? 'health-result healthy' : 'health-result unhealthy'} role="status">{health.healthy ? <ShieldCheck size={17}/> : <ShieldAlert size={17}/>}<span><strong>{health.healthy ? 'Everything looks healthy' : 'Solace found a problem'}</strong><small>{health.message} · {formatTimestamp(health.checkedAt)}</small></span></div>}
  </div>
}

function formatTimestamp(value:string) { return new Intl.DateTimeFormat(undefined,{ dateStyle:'medium', timeStyle:'short' }).format(new Date(value)) }
