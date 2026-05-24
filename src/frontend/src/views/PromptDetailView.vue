<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { promptApi } from '@/api/promptApi'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const promptStore = usePromptStore()
const userStore = useUserStore()
const liking = ref(false)
const favoriting = ref(false)

const promptId = computed(() => Number(route.params.id))
const prompt = computed(() => promptStore.currentPrompt)

const relatedPrompts = computed(() => {
  if (!prompt.value) {
    return []
  }

  return promptStore.getRelatedPrompts(prompt.value.id, prompt.value.categoryId)
})

const promptMeta = computed(() => {
  if (!prompt.value) {
    return []
  }

  return [
    { label: '模型', value: prompt.value.model },
    { label: '分类', value: prompt.value.categoryName },
    { label: '发布时间', value: prompt.value.createdAt },
    { label: '更新时间', value: prompt.value.updatedAt }
  ]
})

const coverImage = computed(() => {
  if (!prompt.value) {
    return ''
  }

  return resolveMediaUrl(prompt.value.cover)
})

const showCoverImage = computed(() => isDisplayableCover(prompt.value?.cover))

const copyText = async (label: string, text: string) => {
  if (!text.trim()) {
    message.warning(`没有可复制的${label}`)
    return
  }

  try {
    await navigator.clipboard.writeText(text)
    message.success(`已复制${label}`)
  } catch {
    message.error('复制失败，请检查浏览器权限')
  }
}

const statCards = computed(() => {
  if (!prompt.value) {
    return []
  }

  return [
    { label: '浏览', value: prompt.value.views.toLocaleString() },
    { label: '点赞', value: prompt.value.likes.toLocaleString() },
    { label: '收藏', value: prompt.value.favorites.toLocaleString() }
  ]
})

const ensureAuthenticated = async () => {
  if (userStore.isLoggedIn) {
    return true
  }

  await router.push(`/login?redirect=${encodeURIComponent(route.fullPath)}`)
  return false
}

const handleLike = async () => {
  if (!prompt.value || liking.value) {
    return
  }

  if (!(await ensureAuthenticated())) {
    return
  }

  liking.value = true
  try {
    const response = await promptApi.likePrompt(prompt.value.id)
    promptStore.mergePrompt(response.data.prompt)
    message.success(response.data.applied ? '已点赞' : '你已经点过赞了')
  } finally {
    liking.value = false
  }
}

const handleFavorite = async () => {
  if (!prompt.value || favoriting.value) {
    return
  }

  if (!(await ensureAuthenticated())) {
    return
  }

  favoriting.value = true
  try {
    const response = await promptApi.favoritePrompt(prompt.value.id)
    promptStore.mergePrompt(response.data.prompt)
    message.success(response.data.applied ? '已收藏' : '你已经收藏过了')
  } finally {
    favoriting.value = false
  }
}

const loadDetail = async () => {
  await promptStore.ensurePromptSeed()

  if (Number.isNaN(promptId.value)) {
    promptStore.setCurrentPrompt(null)
    return
  }

  await promptStore.loadPromptDetail(promptId.value)
}

onMounted(loadDetail)
watch(() => route.params.id, loadDetail)
</script>

<template>
  <div class="detail-page">
    <div class="detail-container">
      <header class="detail-breadcrumb">
        <RouterLink
          to="/"
          class="btn-pill-secondary bg-white"
        >
          返回首页
        </RouterLink>
        <span>/</span>
        <span v-if="prompt">{{ prompt.categoryName }}</span>
        <span v-else>提示词详情</span>
        <span
          v-if="promptStore.usingMockData"
          class="detail-badge"
        >
          演示详情
        </span>
      </header>

      <section
        v-if="promptStore.detailLoading"
        class="detail-loading"
      >
        <div class="detail-loading__block" />
        <div class="detail-loading__block" />
      </section>

      <section
        v-else-if="prompt"
        class="detail-layout"
      >
        <div class="detail-main">
          <section class="panel-card detail-hero">
            <div
              v-if="showCoverImage"
              class="detail-cover-wrap"
            >
              <div class="detail-cover-inner">
                <img
                  :src="coverImage"
                  :alt="prompt.title"
                  class="detail-cover-image"
                >
              </div>
            </div>

            <div class="detail-hero__grid">
              <div class="detail-preview">
                <div class="detail-preview__badge">
                  AI 效果预览
                </div>
                <h1 class="detail-preview__title">
                  {{ prompt.title }}
                </h1>
                <p class="detail-preview__desc">
                  {{ prompt.description }}
                </p>

                <div class="detail-preview__tags">
                  <span
                    v-for="tag in prompt.tags"
                    :key="tag"
                    class="detail-tag"
                  >
                    {{ tag }}
                  </span>
                </div>
              </div>

              <div class="detail-sidebar-panel">
                <div>
                  <div class="detail-eyebrow">
                    创作者
                  </div>
                  <div class="detail-creator-name">
                    {{ prompt.user.username }}
                  </div>
                  <p class="detail-creator-bio">
                    {{ prompt.user.bio }}
                  </p>
                </div>

                <div class="detail-actions">
                  <button
                    class="detail-btn-like"
                    :disabled="liking"
                    @click="handleLike"
                  >
                    {{ liking ? '点赞中...' : `点赞 · ${prompt.likes.toLocaleString()}` }}
                  </button>
                  <button
                    class="detail-btn-favorite"
                    :disabled="favoriting"
                    @click="handleFavorite"
                  >
                    {{ favoriting ? '收藏中...' : `收藏 · ${prompt.favorites.toLocaleString()}` }}
                  </button>
                </div>

                <div class="detail-stat-grid">
                  <div
                    v-for="stat in statCards"
                    :key="stat.label"
                    class="detail-stat"
                  >
                    <div class="detail-stat__value">
                      {{ stat.value }}
                    </div>
                    <div class="detail-stat__label">
                      {{ stat.label }}
                    </div>
                  </div>
                </div>

                <div class="detail-output">
                  <div class="detail-output__label">
                    预期输出
                  </div>
                  <p class="detail-output__text">
                    适合作为可直接落地的起点。上线前请替换为你的业务场景、品牌信息与约束条件。
                  </p>
                </div>
              </div>
            </div>
          </section>

          <section class="detail-content-grid">
            <article class="detail-content-card">
              <div class="detail-content-head">
                <div class="detail-eyebrow">
                  提示词正文
                </div>
                <button
                  class="detail-copy-btn"
                  @click="copyText('提示词', prompt.content)"
                >
                  复制提示词
                </button>
              </div>
              <pre class="detail-pre">{{ prompt.content }}</pre>
            </article>

            <article class="detail-content-card">
              <div class="detail-content-head">
                <div class="detail-eyebrow">
                  系统提示词
                </div>
                <button
                  class="detail-copy-btn"
                  @click="copyText('系统提示词', prompt.systemPrompt)"
                >
                  复制系统提示词
                </button>
              </div>
              <pre class="detail-pre">{{ prompt.systemPrompt }}</pre>
            </article>
          </section>

          <section
            v-if="relatedPrompts.length > 0"
            class="detail-content-card"
          >
            <div>
              <div class="detail-eyebrow">
                相关推荐
              </div>
              <h2 class="detail-section-title">
                同分类更多内容
              </h2>
            </div>

            <div class="detail-related-grid">
              <RouterLink
                v-for="item in relatedPrompts"
                :key="item.id"
                :to="`/prompt/${item.id}`"
                class="detail-related-card"
              >
                <div class="detail-related-model">
                  {{ item.model }}
                </div>
                <div class="detail-related-title">
                  {{ item.title }}
                </div>
                <p class="detail-related-desc">
                  {{ item.description }}
                </p>
              </RouterLink>
            </div>
          </section>
        </div>

        <aside class="detail-aside">
          <section class="detail-content-card">
            <div class="detail-eyebrow">
              提示词信息
            </div>
            <div class="detail-meta-list">
              <div
                v-for="item in promptMeta"
                :key="item.label"
                class="detail-meta-row"
              >
                <div class="detail-meta-label">
                  {{ item.label }}
                </div>
                <div class="detail-meta-value">
                  {{ item.value }}
                </div>
              </div>
            </div>
          </section>

          <section class="detail-content-card">
            <div class="detail-eyebrow">
              参数
            </div>
            <div class="detail-params-grid">
              <div class="detail-stat">
                <div class="detail-stat__value">
                  {{ prompt.params.temperature ?? '-' }}
                </div>
                <div class="detail-stat__label">
                  温度
                </div>
              </div>
              <div class="detail-stat">
                <div class="detail-stat__value">
                  {{ prompt.params.topP ?? '-' }}
                </div>
                <div class="detail-stat__label">
                  Top P
                </div>
              </div>
              <div class="detail-stat">
                <div class="detail-stat__value">
                  {{ prompt.params.maxTokens ?? '-' }}
                </div>
                <div class="detail-stat__label">
                  最大 Token
                </div>
              </div>
            </div>
          </section>

          <section class="detail-content-card">
            <div class="detail-eyebrow">
              使用说明
            </div>
            <ul class="detail-usage-list">
              <li>保留整体结构，再替换为你的业务场景、目标用户与约束条件。</li>
              <li>若输出过于发散，可先降低温度，再补充更强示例。</li>
              <li>上线前请用真实生产输入至少跑一遍工作流回归验证。</li>
            </ul>
          </section>
        </aside>
      </section>

      <section
        v-else
        class="empty-state"
      >
        <h1 class="empty-state__title text-2xl">
          未找到该提示词
        </h1>
        <p class="empty-state__desc">
          内容可能已被删除，或链接已失效。
        </p>
        <RouterLink
          to="/"
          class="btn-pill-secondary detail-back"
        >
          返回首页
        </RouterLink>
      </section>
    </div>
  </div>
</template>

<style scoped>
.detail-page {
  @apply view-page;
}

.detail-container {
  @apply view-container;
}

.detail-breadcrumb {
  @apply mb-8 flex flex-wrap items-center gap-3 text-sm text-[#777777];
}

.detail-badge {
  @apply rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-xs text-amber-800;
}

.detail-loading {
  @apply grid gap-6 lg:grid-cols-[1.35fr_0.65fr];
}

.detail-loading__block {
  @apply h-[520px] animate-pulse rounded-[28px] bg-black/5;
}

.detail-layout {
  @apply grid gap-6 lg:grid-cols-[1.35fr_0.65fr];
}

.detail-back {
  @apply mt-6 inline-flex px-5 py-3;
}

.detail-main {
  @apply space-y-6;
}

.detail-hero {
  @apply overflow-hidden;
}

.detail-cover-wrap {
  @apply max-h-[420px] overflow-hidden border-b border-black/10 bg-[#f6f4ef];
}

.detail-cover-inner {
  @apply flex max-h-[420px] items-center justify-center overflow-hidden;
}

.detail-cover-image {
  @apply max-h-[420px] max-w-full h-auto w-auto object-contain;
}

.detail-hero__grid {
  @apply grid gap-0 lg:grid-cols-[1.1fr_0.9fr];
}

.detail-preview {
  @apply min-h-[340px] bg-[#faf8f4] p-8;
}

.detail-preview__badge {
  @apply inline-flex rounded-full border border-black/10 bg-white px-3 py-1 text-xs text-[#444444];
}

.detail-preview__title {
  @apply mt-5 max-w-2xl text-3xl font-semibold leading-tight text-black sm:text-4xl;
}

.detail-preview__desc {
  @apply mt-4 max-w-xl text-base leading-7 text-[#555555];
}

.detail-preview__tags {
  @apply mt-6 flex flex-wrap gap-2;
}

.detail-tag {
  @apply rounded-full border border-black/10 bg-white px-3 py-1 text-xs text-[#444444];
}

.detail-sidebar-panel {
  @apply flex flex-col justify-between gap-5 border-t border-black/10 bg-white p-8 lg:border-l lg:border-t-0;
}

.detail-eyebrow {
  @apply text-xs uppercase tracking-[0.2em] text-[#777777];
}

.detail-creator-name {
  @apply mt-3 text-2xl font-semibold text-black;
}

.detail-creator-bio {
  @apply mt-3 text-sm leading-6 text-[#555555];
}

.detail-actions {
  @apply flex flex-wrap gap-3;
}

.detail-btn-like {
  @apply rounded-full bg-black px-4 py-2 text-sm font-medium text-white transition hover:bg-black/80 disabled:cursor-not-allowed disabled:opacity-70;
}

.detail-btn-favorite {
  @apply rounded-full border border-black/10 bg-[#f6f4ef] px-4 py-2 text-sm font-medium text-[#333333] transition hover:border-black/20 hover:text-black disabled:cursor-not-allowed disabled:opacity-70;
}

.detail-stat-grid {
  @apply grid grid-cols-3 gap-3;
}

.detail-stat {
  @apply rounded-[18px] border border-black/10 bg-[#faf8f4] p-4 text-center;
}

.detail-stat__value {
  @apply text-lg font-semibold text-black;
}

.detail-stat__label {
  @apply mt-1 text-xs text-[#777777];
}

.detail-output {
  @apply rounded-[18px] border border-black/10 bg-[#111111] p-4 text-white;
}

.detail-output__label {
  @apply text-xs uppercase tracking-[0.2em] text-white/60;
}

.detail-output__text {
  @apply mt-3 text-sm leading-6 text-white/70;
}

.detail-content-grid {
  @apply grid gap-6 xl:grid-cols-[1fr_1fr];
}

.detail-content-card {
  @apply panel-card p-6;
}

.detail-content-head {
  @apply flex items-center justify-between gap-3;
}

.detail-copy-btn {
  @apply rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1.5 text-xs text-[#333333] transition hover:border-black/20 hover:text-black;
}

.detail-pre {
  @apply mt-4 whitespace-pre-wrap text-sm leading-7 text-[#444444];
}

.detail-related-grid {
  @apply mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3;
}

.detail-related-card {
  @apply rounded-[20px] border border-black/10 bg-[#faf8f4] p-5 transition hover:-translate-y-1 hover:border-black/20 hover:bg-white;
}

.detail-related-model {
  @apply text-xs uppercase tracking-[0.18em] text-[#7c7c7c];
}

.detail-related-title {
  @apply mt-3 text-lg font-semibold text-black;
}

.detail-related-desc {
  @apply mt-3 text-sm leading-6 text-[#555555];
}

.detail-aside {
  @apply space-y-6;
}

.detail-meta-list {
  @apply mt-5 space-y-4;
}

.detail-meta-row {
  @apply flex items-start justify-between gap-4 border-b border-black/10 pb-4 last:border-b-0 last:pb-0;
}

.detail-meta-label {
  @apply text-sm text-[#777777];
}

.detail-meta-value {
  @apply text-right text-sm text-black;
}

.detail-params-grid {
  @apply mt-5 grid grid-cols-3 gap-3;
}

.detail-usage-list {
  @apply mt-5 space-y-3 text-sm leading-6 text-[#555555];
}

.detail-section-title {
  @apply mt-2 text-2xl font-semibold text-black;
}
</style>
