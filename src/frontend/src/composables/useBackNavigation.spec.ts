import { describe, expect, it } from 'vitest'
import { isSafeInternalPath } from './useBackNavigation'

describe('isSafeInternalPath', () => {
  it('accepts ordinary internal paths and query strings', () => {
    expect(isSafeInternalPath('/prompts/42?tab=comments')).toBe('/prompts/42?tab=comments')
  })

  it('rejects protocol-relative, encoded protocol-relative and backslash redirects', () => {
    expect(isSafeInternalPath('//evil.example')).toBe('/')
    expect(isSafeInternalPath('/%2F%2Fevil.example')).toBe('/')
    expect(isSafeInternalPath('/\\\\evil.example')).toBe('/')
    expect(isSafeInternalPath('/%5C%5Cevil.example')).toBe('/')
  })

  it('rejects control characters and auth loop redirects', () => {
    expect(isSafeInternalPath('/prompts/%00')).toBe('/')
    expect(isSafeInternalPath('/login?redirect=/prompts/42')).toBe('/')
  })
})
