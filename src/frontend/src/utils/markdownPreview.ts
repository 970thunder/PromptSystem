// 文件作用：将发布页输入的 Markdown 安全转换为受控预览 HTML。
function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

const allowedLinkProtocols = ['http:', 'https:', 'mailto:']

export interface MarkdownPreviewBlock {
  type: 'paragraph' | 'heading' | 'code'
  text: string
  level?: 1 | 2 | 3
}

function sanitizeLinkUrl(rawUrl: string): string {
  const trimmed = rawUrl.trim()
  if (trimmed.startsWith('/') && !trimmed.startsWith('//')) {
    return trimmed
  }

  try {
    const url = new URL(trimmed)
    return allowedLinkProtocols.includes(url.protocol) ? url.href : '#'
  } catch {
    return '#'
  }
}

export function renderMarkdownPreviewBlocks(source: string): MarkdownPreviewBlock[] {
  const blocks: MarkdownPreviewBlock[] = []
  const lines = source.split(/\r?\n/)
  let inCodeBlock = false
  let codeLines: string[] = []

  for (const line of lines) {
    if (line.trim().startsWith('```')) {
      if (inCodeBlock) {
        blocks.push({ type: 'code', text: codeLines.join('\n').trim() })
        codeLines = []
        inCodeBlock = false
      } else {
        inCodeBlock = true
      }
      continue
    }

    if (inCodeBlock) {
      codeLines.push(line)
      continue
    }

    const heading = parseHeading(line)
    if (heading) {
      blocks.push(heading)
      continue
    }

    const text = renderInlinePreviewText(line).trim()
    if (text) {
      blocks.push({ type: 'paragraph', text })
    }
  }

  if (codeLines.length > 0) {
    blocks.push({ type: 'code', text: codeLines.join('\n').trim() })
  }

  return blocks
}

function parseHeading(line: string): MarkdownPreviewBlock | null {
  const matched = line.match(/^(#{1,3})\s+(.+)$/)
  if (!matched) {
    return null
  }

  return {
    type: 'heading',
    level: matched[1].length as 1 | 2 | 3,
    text: renderInlinePreviewText(matched[2]).trim()
  }
}

function renderInlinePreviewText(value: string): string {
  return value
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_match, label: string, rawUrl: string) => {
      const safeUrl = sanitizeLinkUrl(rawUrl)
      return safeUrl === '#' ? label : `${label} (${safeUrl})`
    })
    .replace(/\*\*(.+?)\*\*/g, '$1')
    .replace(/\*(.+?)\*/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
}

function renderSafeLink(label: string, rawUrl: string): string {
  const safeLabel = escapeHtml(label)
  const safeUrl = escapeHtml(sanitizeLinkUrl(rawUrl))
  return `<a href="${safeUrl}" class="underline" target="_blank" rel="noopener noreferrer">${safeLabel}</a>`
}

/** Lightweight Markdown preview for publish step (not a full spec implementation). */
export function renderMarkdownPreview(source: string): string {
  const escaped = escapeHtml(source)

  const withCode = escaped.replace(/```([\s\S]*?)```/g, (_match, code: string) => {
    return `<pre class="rounded-xl bg-black/5 p-3 text-xs overflow-x-auto"><code>${code.trim()}</code></pre>`
  })

  const withInlineCode = withCode.replace(/`([^`]+)`/g, '<code class="rounded bg-black/5 px-1 text-xs">$1</code>')
  const withHeadings = withInlineCode
    .replace(/^### (.+)$/gm, '<h3 class="mt-3 text-base font-semibold">$1</h3>')
    .replace(/^## (.+)$/gm, '<h2 class="mt-4 text-lg font-semibold">$1</h2>')
    .replace(/^# (.+)$/gm, '<h1 class="mt-4 text-xl font-semibold">$1</h1>')
  const withBold = withHeadings.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  const withItalic = withBold.replace(/\*(.+?)\*/g, '<em>$1</em>')
  const withLinks = withItalic.replace(
    /\[([^\]]+)\]\(([^)]+)\)/g,
    (_match, label: string, rawUrl: string) => renderSafeLink(label, rawUrl)
  )
  const withBreaks = withLinks.replace(/\n/g, '<br>')

  return withBreaks
}
