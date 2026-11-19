import * as NativeService from '../../bindings/solace/internal/app/service'
import { isWailsHost } from './bridge'

const knownMessages: Array<[RegExp, string]> = [
  [/wrong password|invalid password|password does not match/i, 'That password didn’t match. Please try again.'],
  [/not found|no rows/i, 'We couldn’t find that item. It may have been moved or removed.'],
  [/stale|version conflict|changed by another operation/i, 'This entry changed elsewhere. Reopen it, then try again.'],
  [/permission denied|access denied|operation not permitted/i, 'Solace couldn’t access that location. Check its permissions, then try again.'],
  [/no space|disk full/i, 'There isn’t enough free space to finish this. Free some space, then try again.'],
  [/integrity|corrupt|malformed database/i, 'Solace found a problem with your journal data. Nothing was changed. Check Data & safety in Settings.'],
  [/cancelled|canceled/i, 'That action was cancelled. Your journal was not changed.'],
]

export function friendlyError(reason: unknown, fallback: string, area: string) {
  const detail = technicalMessage(reason)
  reportClientError(area, detail)
  return knownMessages.find(([pattern]) => pattern.test(detail))?.[1] ?? fallback
}

export function reportClientError(area: string, reason: unknown) {
  const detail = typeof reason === 'string' ? reason : technicalMessage(reason)
  if (isWailsHost()) {
    void NativeService.ReportClientError(area, detail).catch(() => undefined)
  } else {
    console.error(`[Solace: ${area}]`, detail)
  }
}

function technicalMessage(reason: unknown) {
  if (reason instanceof Error) return reason.message || reason.name
  if (typeof reason === 'string') return reason
  try { return JSON.stringify(reason) } catch { return 'Unknown interface error' }
}
