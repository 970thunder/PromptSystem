function escapeHtml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
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
    '<a href="$2" class="underline" target="_blank" rel="noopener noreferrer">$1</a>'
  )
  const withBreaks = withLinks.replace(/\n/g, '<br>')

  return withBreaks
}
