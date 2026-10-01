// 文件作用：封面缺失或不可显示时的确定性字卡兜底。
// 用本地 SVG data URI（品牌色渐变 + 标题首字），无外链依赖，弱网/国内外网络均可即时渲染；
// 同一内容按 id 稳定取色，刷新后不变。PromptCard 与首页大屏共用。

const COVER_PALETTES: Array<[string, string]> = [
  ['#1d4ed8', '#38bdf8'],
  ['#0f766e', '#34d399'],
  ['#7c3aed', '#c4b5fd'],
  ['#b45309', '#fcd34d'],
  ['#be185d', '#f9a8d4'],
  ['#334155', '#94a3b8']
]

const escapeXml = (value: string) => value
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')
  .replace(/"/g, '&quot;')
  .replace(/'/g, '&apos;')

// 字卡主字取标题前两个字符；空标题回退为产品首字母。
export function fallbackCoverGlyph(title: string | null | undefined): string {
  const trimmed = (title ?? '').trim()
  if (!trimmed) {
    return 'P'
  }
  return Array.from(trimmed).slice(0, 2).join('')
}

export function fallbackCoverUrl(seed: number, title?: string | null): string {
  const index = Number.isFinite(seed) ? Math.abs(Math.floor(seed)) : 0
  const [from, to] = COVER_PALETTES[index % COVER_PALETTES.length]
  const glyph = escapeXml(fallbackCoverGlyph(title))
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="900" viewBox="0 0 1200 900">`
    + `<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">`
    + `<stop offset="0" stop-color="${from}"/><stop offset="1" stop-color="${to}"/>`
    + `</linearGradient></defs>`
    + `<rect width="1200" height="900" fill="url(#g)"/>`
    + `<text x="600" y="520" font-family="'PingFang SC','Hiragino Sans GB','Microsoft YaHei',sans-serif" font-size="340" font-weight="700" fill="rgba(255,255,255,0.85)" text-anchor="middle">${glyph}</text>`
    + `</svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}
