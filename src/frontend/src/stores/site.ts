// 文件作用：站点公开信息（名称/描述/注册开关）的轻量缓存 store。
// 只在首次访问时请求 /site/info，失败时回退默认值，不阻塞任何页面。
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { setupApi, type SiteInfo } from '@/api/setupApi'

const fallbackSiteInfo: SiteInfo = {
  name: 'PromptOS',
  description: '',
  registrationOpen: true,
  emailAuthEnabled: true,
  githubOAuthEnabled: false
}

export const useSiteStore = defineStore('site', () => {
  const siteInfo = ref<SiteInfo>(fallbackSiteInfo)
  const loaded = ref(false)
  let loadPromise: Promise<void> | null = null

  const siteName = computed(() => siteInfo.value.name || fallbackSiteInfo.name)

  const ensureLoaded = () => {
    if (loaded.value || loadPromise) {
      return loadPromise ?? Promise.resolve()
    }
    loadPromise = setupApi
      .getSiteInfo()
      .then((response) => {
        siteInfo.value = { ...fallbackSiteInfo, ...response.data }
        loaded.value = true
      })
      .catch(() => {
        // 站点信息属于增强数据：后端不可用时使用默认值，不弹错误。
        loaded.value = true
      })
    return loadPromise
  }

  return { siteInfo, siteName, loaded, ensureLoaded }
})
