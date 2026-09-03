import { describe, expect, it } from 'vitest'
import { siteCapabilities } from '@/config/capabilities'

describe('site capabilities', () => {
  it('keeps unfinished modules disabled by default', () => {
    expect(siteCapabilities.emailAuth).toBe(true)
    expect(siteCapabilities.githubOAuth).toBe(false)
    expect(siteCapabilities.skillRunner).toBe(false)
    expect(siteCapabilities.playground).toBe(false)
    expect(siteCapabilities.creatorAcademy).toBe(false)
    expect(siteCapabilities.marketplace).toBe(false)
  })
})
