import { afterEach, describe, expect, it, vi } from 'vitest'
import { friendlyError } from './user-errors'

describe('friendly interface errors', () => {
  afterEach(() => vi.restoreAllMocks())

  it('hides unknown technical details behind contextual guidance', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const visible = friendlyError(new Error('SQLITE_BUSY: database table is locked'), 'Your changes couldn’t be saved. Please try again.', 'save entry')
    expect(visible).toBe('Your changes couldn’t be saved. Please try again.')
    expect(visible).not.toContain('SQLITE')
  })

  it('translates a known password failure into plain language', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
    expect(friendlyError(new Error('wrong password'), 'Unlock failed.', 'unlock journal')).toBe('That password didn’t match. Please try again.')
  })
})
