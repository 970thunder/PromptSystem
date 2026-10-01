import { describe, expect, it } from 'vitest'
import { fallbackCoverGlyph, fallbackCoverUrl } from '../coverFallback'

describe('coverFallback', () => {
  it('generates a local data-URI cover without external hosts', () => {
    const url = fallbackCoverUrl(7, '客户支持工作流')

    expect(url.startsWith('data:image/svg+xml;charset=utf-8,')).toBe(true)
    expect(url.startsWith('http')).toBe(false)
    expect(url).not.toContain('unsplash')
  })

  it('is deterministic for the same seed', () => {
    expect(fallbackCoverUrl(3, '代码审查助手')).toBe(fallbackCoverUrl(3, '代码审查助手'))
    expect(fallbackCoverUrl(3, '代码审查助手')).not.toBe(fallbackCoverUrl(4, '代码审查助手'))
  })

  it('uses the first two characters of the title as the glyph', () => {
    expect(fallbackCoverGlyph('客户支持工作流')).toBe('客户')
    expect(fallbackCoverGlyph('Code Review')).toBe('Co')
    expect(fallbackCoverGlyph('')).toBe('P')
    expect(fallbackCoverGlyph(null)).toBe('P')
    // 字形只取前两个字符，且经过 XML 转义，防止标题内容破坏 SVG 结构
    expect(fallbackCoverUrl(1, '<x>&"')).toContain(encodeURIComponent('&lt;x'))
  })
})
