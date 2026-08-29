// 文件作用：封面缺失或不可显示时的确定性回退图池。按稳定种子（如提示词 id）
// 取图，保证同一条内容刷新后仍显示同一张占位图；PromptCard 与首页大屏共用。
export const FALLBACK_COVER_URLS = [
  'https://images.unsplash.com/photo-1516321497487-e288fb19713f?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1498050108023-c5249f4df085?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1526379095098-d400fd0bf935?auto=format&fit=crop&w=1200&q=80',
  'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80'
]

export function fallbackCoverUrl(seed: number): string {
  if (!Number.isFinite(seed) || seed < 0) {
    return FALLBACK_COVER_URLS[0]
  }
  return FALLBACK_COVER_URLS[seed % FALLBACK_COVER_URLS.length]
}
