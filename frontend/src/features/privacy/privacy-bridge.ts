import * as NativePrivacy from '../../../bindings/solace/internal/privacy/service'
import type { Configuration, Status } from '../../../bindings/solace/internal/privacy/models'
import { isWailsHost } from '../../lib/bridge'

const disabled: Status = { enabled:false, locked:false, lockOnStart:false, inactivityMinutes:0 }

export const privacyBridge = {
  isNative:isWailsHost,
  async status(): Promise<Status> { return isWailsHost() ? NativePrivacy.GetStatus() : disabled },
  async configure(value: Configuration): Promise<Status> {
    if (!isWailsHost()) throw new Error('Privacy lock is available in the native Solace desktop app.')
    return NativePrivacy.Configure(value)
  },
  async disable(password:string): Promise<Status> {
    if (!isWailsHost()) throw new Error('Privacy lock is available in the native Solace desktop app.')
    return NativePrivacy.Disable(password)
  },
  async lock(): Promise<Status> { return isWailsHost() ? NativePrivacy.Lock() : disabled },
  async unlock(password:string): Promise<Status> {
    if (!isWailsHost()) return disabled
    return NativePrivacy.Unlock(password)
  },
}

export const privacyChangedEvent = 'solace:privacy-changed'
export function announcePrivacyChange(status:Status) {
  window.dispatchEvent(new CustomEvent<Status>(privacyChangedEvent, { detail:status }))
}
