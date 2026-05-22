const uploadPathPattern = /^\/uploads\//

export function resolveMediaUrl(url: string | undefined | null): string {
  if (!url?.trim()) {
    return ''
  }

  const trimmed = url.trim()

  if (uploadPathPattern.test(trimmed) || trimmed.startsWith('data:')) {
    return trimmed
  }

  if (/^https?:\/\//i.test(trimmed)) {
    try {
      const parsed = new URL(trimmed)
      if (uploadPathPattern.test(parsed.pathname)) {
        return `${parsed.pathname}${parsed.search}`
      }
    } catch {
      return trimmed
    }

    return trimmed
  }

  return trimmed
}

export function isDisplayableCover(url: string | undefined | null): boolean {
  const resolved = resolveMediaUrl(url)
  return /^https?:\/\//i.test(resolved) || uploadPathPattern.test(resolved) || resolved.startsWith('data:image')
}
