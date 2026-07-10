<!-- 文件作用：展示 Prompt 详情、互动操作、结构化示例和社区评论。 -->
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useDialog, useMessage } from 'naive-ui'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { promptApi } from '@/api/promptApi'
import { userApi } from '@/api/userApi'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import { extractPromptExamples, extractPromptWorkflow } from '@/utils/promptStructure'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'
import type { Comment, FollowStatus } from '@/types'

const route = useRoute()
const router = useRouter()
const dialog = useDialog()
const message = useMessage()
const promptStore = usePromptStore()
const userStore = useUserStore()
const liking = ref(false)
const favoriting = ref(false)
const followingCreator = ref(false)
const commentSubmitting = ref(false)
const commentReporting = ref(false)
const promptReporting = ref(false)
const activeReplyId = ref<number | null>(null)
const commentDraft = ref('')
const replyDrafts = ref<Record<number, string>>({})
const commentSort = ref<'latest' | 'popular' | 'oldest'>('latest')
const activeImageIndex = ref(0)

const promptId = computed(() => Number(route.params.id))
const prompt = computed(() => promptStore.currentPrompt)
const comments = computed(() => promptStore.comments)
const followStatus = ref<FollowStatus | null>(null)
const commentSortOptions = [
  { label: '最新', value: 'latest' },
  { label: '热门', value: 'popular' },
  { label: '最早', value: 'oldest' }
] as const

const relatedPrompts = computed(() => {
  if (!prompt.value) {
    return []
  }

  return promptStore.getRelatedPrompts(prompt.value.id, prompt.value.categoryId)
})

const promptExamples = computed(() => prompt.value ? extractPromptExamples(prompt.value.content) : [])
const promptWorkflow = computed(() => prompt.value ? extractPromptWorkflow(prompt.value.content) : [])

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

const galleryImages = computed(() => {
  if (!prompt.value) {
    return []
  }

  const images = [prompt.value.cover, ...(prompt.value.images ?? [])]
    .filter((image) => isDisplayableCover(image))
    .map((image) => resolveMediaUrl(image))

  return Array.from(new Set(images))
})

const activeGalleryImage = computed(() => galleryImages.value[activeImageIndex.value] ?? '')
const showCoverImage = computed(() => galleryImages.value.length > 0)
const shareUrl = computed(() => {
  if (!prompt.value) {
    return window.location.href
  }

  return `${window.location.origin}/prompt/${prompt.value.id}`
})
const shareText = computed(() => {
  if (!prompt.value) {
    return ''
  }

  return `${prompt.value.title} - ${prompt.value.description}`
})

const setActiveImage = (index: number) => {
  if (galleryImages.value.length === 0) {
    activeImageIndex.value = 0
    return
  }

  const next = Math.max(0, Math.min(index, galleryImages.value.length - 1))
  activeImageIndex.value = next
}

const goImage = (direction: -1 | 1) => {
  if (galleryImages.value.length <= 1) {
    return
  }

  const next = (activeImageIndex.value + direction + galleryImages.value.length) % galleryImages.value.length
  activeImageIndex.value = next
}

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

const handleShare = async () => {
  if (!prompt.value) {
    return
  }

  const payload = {
    title: prompt.value.title,
    text: shareText.value,
    url: shareUrl.value
  }

  if (navigator.share) {
    try {
      await navigator.share(payload)
      message.success('分享面板已打开')
      return
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        return
      }
    }
  }

  await copyText('分享链接', shareUrl.value)
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

const canFollowCreator = computed(() => {
  return Boolean(prompt.value && userStore.userInfo?.id !== prompt.value.userId)
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

const loadFollowStatus = async () => {
  if (!prompt.value || !userStore.isLoggedIn || !canFollowCreator.value) {
    followStatus.value = null
    return
  }

  try {
    const response = await userApi.getFollowStatus(prompt.value.userId)
    followStatus.value = response.data
  } catch {
    followStatus.value = null
  }
}

const handleFollowCreator = async () => {
  if (!prompt.value || followingCreator.value || !canFollowCreator.value) {
    return
  }

  if (!(await ensureAuthenticated())) {
    return
  }

  followingCreator.value = true
  try {
    const response = followStatus.value?.following
      ? await userApi.unfollowUser(prompt.value.userId)
      : await userApi.followUser(prompt.value.userId)
    followStatus.value = response.data.status
    message.success(response.data.status.following ? '已关注创作者' : '已取消关注')
  } finally {
    followingCreator.value = false
  }
}

const ensureCommentInput = (value: string) => {
  const trimmed = value.trim()
  if (!trimmed) {
    message.warning('请先输入评论内容')
    return ''
  }

  return trimmed
}

const submitComment = async (parentId?: number) => {
  if (!prompt.value || commentSubmitting.value) {
    return
  }

  if (!(await ensureAuthenticated())) {
    return
  }

  const source = parentId ? replyDrafts.value[parentId] ?? '' : commentDraft.value
  const content = ensureCommentInput(source)
  if (!content) {
    return
  }

  commentSubmitting.value = true
  try {
    await promptStore.createPromptComment(prompt.value.id, {
      content,
      parentId: parentId ?? null
    }, commentSort.value)

    if (parentId) {
      replyDrafts.value = {
        ...replyDrafts.value,
        [parentId]: ''
      }
      activeReplyId.value = null
    } else {
      commentDraft.value = ''
    }

    message.success(parentId ? '回复已发布' : '评论已发布')
  } finally {
    commentSubmitting.value = false
  }
}

const handleCommentLike = async (comment: Comment) => {
  if (!prompt.value) {
    return
  }

  if (!(await ensureAuthenticated())) {
    return
  }

  await promptStore.likeComment(prompt.value.id, comment.id, commentSort.value)
  message.success('评论已点赞')
}

const toggleReply = (commentId: number) => {
  activeReplyId.value = activeReplyId.value === commentId ? null : commentId
}

const handleCommentReport = async (comment: Comment) => {
  if (commentReporting.value) {
    return
  }

  if (!(await ensureAuthenticated())) {
    return
  }

  dialog.create({
    title: '举报评论',
    content: '确认举报这条评论？系统会记录为待处理状态，后续可在管理端集中审核。',
    positiveText: '确认举报',
    negativeText: '取消',
    onPositiveClick: async () => {
      commentReporting.value = true
      try {
        const response = await promptStore.reportComment(comment.id, {
          reason: '不当评论',
          detail: comment.content.slice(0, 200)
        })
        message.success(response.applied ? '已提交举报' : '你已经举报过这条评论')
      } finally {
        commentReporting.value = false
      }
    }
  })
}

const handlePromptReport = async () => {
  const current = prompt.value
  if (!current || promptReporting.value) {
    return
  }

  if (!(await ensureAuthenticated())) {
    return
  }

  dialog.create({
    title: '举报提示词',
    content: '确认举报这条提示词？系统会记录为待处理状态，后续可集中审核。',
    positiveText: '确认举报',
    negativeText: '取消',
    onPositiveClick: async () => {
      promptReporting.value = true
      try {
        const response = await promptStore.reportPrompt(current.id, {
          reason: '不当提示词',
          detail: current.title.slice(0, 200)
        })
        message.success(response.applied ? '已提交举报' : '你已经举报过这条提示词')
      } finally {
        promptReporting.value = false
      }
    }
  })
}

const formatCommentTime = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return date.toLocaleString()
}

const loadDetail = async () => {
  await promptStore.ensurePromptSeed()

  if (Number.isNaN(promptId.value)) {
    promptStore.setCurrentPrompt(null)
    return
  }

  await promptStore.loadPromptDetail(promptId.value)
  if (prompt.value && userStore.isLoggedIn) {
    try {
      const response = await promptApi.recordPromptView(prompt.value.id)
      promptStore.mergePrompt(response.data.prompt)
    } catch {
      // Browsing should never block reading a prompt.
    }
  }
  await promptStore.loadPromptComments(promptId.value, commentSort.value)
  await loadFollowStatus()
  setActiveImage(0)
}

watch(galleryImages, () => setActiveImage(0))

onMounted(loadDetail)
watch(() => route.params.id, loadDetail)
watch(commentSort, async () => {
  if (!Number.isNaN(promptId.value)) {
    await promptStore.loadPromptComments(promptId.value, commentSort.value)
  }
})
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
                  :src="activeGalleryImage"
                  :alt="prompt.title"
                  class="detail-cover-image"
                >
                <button
                  v-if="galleryImages.length > 1"
                  class="detail-gallery-nav detail-gallery-nav--prev"
                  type="button"
                  @click="goImage(-1)"
                >
                  上一张
                </button>
                <button
                  v-if="galleryImages.length > 1"
                  class="detail-gallery-nav detail-gallery-nav--next"
                  type="button"
                  @click="goImage(1)"
                >
                  下一张
                </button>
                <div
                  v-if="galleryImages.length > 1"
                  class="detail-gallery-counter"
                >
                  {{ activeImageIndex + 1 }} / {{ galleryImages.length }}
                </div>
              </div>

              <div
                v-if="galleryImages.length > 1"
                class="detail-gallery-thumbs"
              >
                <button
                  v-for="(image, index) in galleryImages"
                  :key="image"
                  class="detail-gallery-thumb"
                  :class="{ 'detail-gallery-thumb--active': activeImageIndex === index }"
                  type="button"
                  @click="setActiveImage(index)"
                >
                  <img
                    :src="image"
                    :alt="`${prompt.title} ${index + 1}`"
                    class="detail-gallery-thumb__image"
                  >
                </button>
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
                  <div class="detail-creator-follow">
                    <RouterLink
                      :to="`/profile/${prompt.userId}`"
                      class="detail-creator-link"
                    >
                      查看主页
                    </RouterLink>
                    <button
                      v-if="canFollowCreator"
                      class="detail-btn-follow"
                      :class="{ 'detail-btn-follow--active': followStatus?.following }"
                      :disabled="followingCreator"
                      @click="handleFollowCreator"
                    >
                      {{
                        followingCreator
                          ? '处理中...'
                          : followStatus?.following
                            ? '已关注'
                            : '关注创作者'
                      }}
                    </button>
                  </div>
                  <div
                    v-if="followStatus"
                    class="detail-creator-stats"
                  >
                    <span>{{ followStatus.followerCount }} 粉丝</span>
                    <span>{{ followStatus.followingCount }} 关注</span>
                  </div>
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
                  <button
                    class="detail-btn-share"
                    @click="handleShare"
                  >
                    分享
                  </button>
                  <button
                    class="detail-btn-share"
                    :disabled="promptReporting"
                    @click="handlePromptReport"
                  >
                    {{ promptReporting ? '提交中...' : '举报' }}
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

          <section class="detail-content-grid">
            <article class="detail-content-card">
              <div>
                <div class="detail-eyebrow">
                  Few-shot
                </div>
                <h2 class="detail-section-title">
                  示例输入与输出
                </h2>
              </div>

              <div
                v-if="promptExamples.length > 0"
                class="detail-structure-list"
              >
                <article
                  v-for="example in promptExamples"
                  :key="example.title"
                  class="detail-structure-card"
                >
                  <div class="detail-structure-card__title">
                    {{ example.title }}
                  </div>
                  <div class="detail-structure-columns">
                    <div>
                      <div class="detail-structure-label">
                        输入
                      </div>
                      <p class="detail-structure-text">
                        {{ example.input || '未提供输入内容' }}
                      </p>
                    </div>
                    <div>
                      <div class="detail-structure-label">
                        输出
                      </div>
                      <p class="detail-structure-text">
                        {{ example.output || '未提供输出内容' }}
                      </p>
                    </div>
                  </div>
                </article>
              </div>
              <p
                v-else
                class="detail-structure-empty"
              >
                当前提示词未提供结构化示例。
              </p>
            </article>

            <article class="detail-content-card">
              <div>
                <div class="detail-eyebrow">
                  Workflow
                </div>
                <h2 class="detail-section-title">
                  只读流程说明
                </h2>
              </div>

              <div
                v-if="promptWorkflow.length > 0"
                class="detail-workflow-list"
              >
                <div
                  v-for="step in promptWorkflow"
                  :key="`${step.title}-${step.detail}`"
                  class="detail-workflow-step"
                >
                  <div class="detail-workflow-step__index">
                    {{ step.title }}
                  </div>
                  <p class="detail-workflow-step__detail">
                    {{ step.detail }}
                  </p>
                </div>
              </div>
              <p
                v-else
                class="detail-structure-empty"
              >
                当前提示词未提供结构化流程说明。
              </p>
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

          <section class="detail-content-card">
            <div class="detail-comments-head">
              <div>
                <div class="detail-eyebrow">
                  评论
                </div>
                <h2 class="detail-section-title">
                  社区反馈
                </h2>
              </div>
              <div class="detail-comments-tools">
                <div class="detail-comment-sort">
                  <button
                    v-for="option in commentSortOptions"
                    :key="option.value"
                    class="detail-comment-sort__btn"
                    :class="{ 'detail-comment-sort__btn--active': commentSort === option.value }"
                    :disabled="promptStore.commentsLoading"
                    @click="commentSort = option.value"
                  >
                    {{ option.label }}
                  </button>
                </div>
                <div class="detail-comments-count">
                  {{ comments.length }}
                </div>
              </div>
            </div>

            <div class="detail-comment-editor">
              <textarea
                v-model="commentDraft"
                class="detail-comment-textarea"
                rows="4"
                placeholder="写下使用反馈、优化建议或实际效果"
              />
              <div class="detail-comment-actions">
                <span class="detail-comment-tip">
                  评论最多 1000 字。
                </span>
                <button
                  class="detail-btn-like"
                  :disabled="commentSubmitting"
                  @click="submitComment()"
                >
                  {{ commentSubmitting ? '发布中...' : '发布评论' }}
                </button>
              </div>
            </div>

            <div
              v-if="promptStore.commentsLoading"
              class="detail-comments-loading"
            >
              评论加载中...
            </div>

            <div
              v-else-if="comments.length === 0"
              class="detail-comments-empty"
            >
              还没有评论。
            </div>

            <div
              v-else
              class="detail-comments-list"
            >
              <article
                v-for="comment in comments"
                :key="comment.id"
                class="detail-comment-card"
              >
                <div class="detail-comment-header">
                  <div>
                    <div class="detail-comment-author">
                      {{ comment.user.username }}
                    </div>
                    <div class="detail-comment-time">
                      {{ formatCommentTime(comment.createdAt) }}
                    </div>
                  </div>
                  <div class="detail-comment-meta">
                    Lv.{{ comment.user.level }}
                  </div>
                </div>

                <p class="detail-comment-content">
                  {{ comment.content }}
                </p>

                <div class="detail-comment-row">
                  <button
                    class="detail-comment-link"
                    @click="handleCommentLike(comment)"
                  >
                    赞 · {{ comment.likes }}
                  </button>
                  <button
                    class="detail-comment-link"
                    @click="toggleReply(comment.id)"
                  >
                    回复
                  </button>
                  <button
                    class="detail-comment-link"
                    :disabled="commentReporting"
                    @click="handleCommentReport(comment)"
                  >
                    举报
                  </button>
                </div>

                <div
                  v-if="activeReplyId === comment.id"
                  class="detail-reply-editor"
                >
                  <textarea
                    v-model="replyDrafts[comment.id]"
                    class="detail-comment-textarea"
                    rows="3"
                    placeholder="写一条直接回复"
                  />
                  <div class="detail-comment-actions">
                    <span class="detail-comment-tip">
                      回复 {{ comment.user.username }}
                    </span>
                    <button
                      class="detail-btn-favorite"
                      :disabled="commentSubmitting"
                      @click="submitComment(comment.id)"
                    >
                      {{ commentSubmitting ? '发布中...' : '发布回复' }}
                    </button>
                  </div>
                </div>

                <div
                  v-if="comment.replies.length > 0"
                  class="detail-replies"
                >
                  <article
                    v-for="reply in comment.replies"
                    :key="reply.id"
                    class="detail-reply-card"
                  >
                    <div class="detail-comment-header">
                      <div>
                        <div class="detail-comment-author">
                          {{ reply.user.username }}
                        </div>
                        <div class="detail-comment-time">
                          {{ formatCommentTime(reply.createdAt) }}
                        </div>
                      </div>
                      <div class="detail-comment-meta">
                        Lv.{{ reply.user.level }}
                      </div>
                    </div>

                    <p class="detail-comment-content">
                      {{ reply.content }}
                    </p>

                    <div class="detail-comment-row">
                      <button
                        class="detail-comment-link"
                        @click="handleCommentLike(reply)"
                      >
                        赞 · {{ reply.likes }}
                      </button>
                      <button
                        class="detail-comment-link"
                        :disabled="commentReporting"
                        @click="handleCommentReport(reply)"
                      >
                        举报
                      </button>
                    </div>
                  </article>
                </div>
              </article>
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
  @apply relative flex max-h-[420px] items-center justify-center overflow-hidden;
}

.detail-cover-image {
  @apply max-h-[420px] max-w-full h-auto w-auto object-contain;
}

.detail-gallery-nav {
  @apply absolute top-1/2 -translate-y-1/2 rounded-full bg-black/70 px-3 py-2 text-xs font-medium text-white transition hover:bg-black;
}

.detail-gallery-nav--prev {
  @apply left-4;
}

.detail-gallery-nav--next {
  @apply right-4;
}

.detail-gallery-counter {
  @apply absolute bottom-4 right-4 rounded-full bg-black/70 px-3 py-1 text-xs font-medium text-white;
}

.detail-gallery-thumbs {
  @apply flex gap-2 overflow-x-auto border-t border-black/10 bg-white p-3;
  scrollbar-width: none;
}

.detail-gallery-thumbs::-webkit-scrollbar {
  display: none;
}

.detail-gallery-thumb {
  @apply h-16 w-20 shrink-0 overflow-hidden rounded-[12px] border border-black/10 bg-[#f6f4ef] opacity-70 transition hover:opacity-100;
}

.detail-gallery-thumb--active {
  @apply border-black opacity-100;
}

.detail-gallery-thumb__image {
  @apply h-full w-full object-cover;
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

.detail-creator-follow {
  @apply mt-5 flex flex-wrap items-center gap-2;
}

.detail-creator-link,
.detail-btn-follow {
  @apply rounded-full border border-black/10 bg-[#f6f4ef] px-4 py-2 text-sm font-medium text-[#333333] transition hover:border-black/20 hover:text-black disabled:cursor-not-allowed disabled:opacity-70;
}

.detail-btn-follow--active {
  @apply bg-black text-white hover:text-white;
}

.detail-creator-stats {
  @apply mt-3 flex flex-wrap gap-3 text-xs text-[#777777];
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

.detail-btn-share {
  @apply rounded-full border border-black/10 bg-white px-4 py-2 text-sm font-medium text-[#333333] transition hover:border-black/20 hover:text-black;
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

.detail-structure-list,
.detail-workflow-list {
  @apply mt-5 space-y-4;
}

.detail-structure-card,
.detail-workflow-step {
  @apply rounded-[18px] border border-black/10 bg-[#faf8f4] p-4;
}

.detail-structure-card__title,
.detail-workflow-step__index {
  @apply text-sm font-semibold text-black;
}

.detail-structure-columns {
  @apply mt-3 grid gap-3 md:grid-cols-2;
}

.detail-structure-label {
  @apply text-xs uppercase tracking-[0.16em] text-[#777777];
}

.detail-structure-text,
.detail-workflow-step__detail,
.detail-structure-empty {
  @apply mt-2 whitespace-pre-wrap text-sm leading-6 text-[#555555];
}

.detail-structure-empty {
  @apply rounded-[18px] border border-dashed border-black/10 bg-[#faf8f4] px-4 py-5;
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

.detail-comments-head {
  @apply flex items-center justify-between gap-4;
}

.detail-comments-tools {
  @apply flex flex-wrap items-center justify-end gap-2;
}

.detail-comment-sort {
  @apply flex rounded-full border border-black/10 bg-[#f6f4ef] p-1;
}

.detail-comment-sort__btn {
  @apply rounded-full px-3 py-1 text-xs text-[#666666] transition hover:text-black disabled:cursor-not-allowed disabled:opacity-60;
}

.detail-comment-sort__btn--active {
  @apply bg-black text-white hover:text-white;
}

.detail-comments-count {
  @apply rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1 text-sm text-[#444444];
}

.detail-comment-editor,
.detail-reply-editor {
  @apply mt-5 rounded-[20px] border border-black/10 bg-[#faf8f4] p-4;
}

.detail-comment-textarea {
  @apply min-h-[110px] w-full resize-y rounded-[16px] border border-black/10 bg-white px-4 py-3 text-sm leading-6 text-[#222222] outline-none transition focus:border-black/20;
}

.detail-comment-actions {
  @apply mt-3 flex flex-wrap items-center justify-between gap-3;
}

.detail-comment-tip {
  @apply text-xs text-[#777777];
}

.detail-comments-loading,
.detail-comments-empty {
  @apply mt-5 rounded-[18px] border border-dashed border-black/10 bg-[#faf8f4] px-4 py-5 text-sm text-[#666666];
}

.detail-comments-list {
  @apply mt-5 space-y-4;
}

.detail-comment-card,
.detail-reply-card {
  @apply rounded-[20px] border border-black/10 bg-[#faf8f4] p-4;
}

.detail-replies {
  @apply mt-4 space-y-3 border-l border-black/10 pl-4;
}

.detail-comment-header {
  @apply flex items-start justify-between gap-4;
}

.detail-comment-author {
  @apply text-sm font-semibold text-black;
}

.detail-comment-time {
  @apply mt-1 text-xs text-[#777777];
}

.detail-comment-meta {
  @apply rounded-full border border-black/10 bg-white px-2.5 py-1 text-xs text-[#555555];
}

.detail-comment-content {
  @apply mt-3 whitespace-pre-wrap text-sm leading-6 text-[#333333];
}

.detail-comment-row {
  @apply mt-3 flex flex-wrap gap-3;
}

.detail-comment-link {
  @apply text-xs text-[#555555] transition hover:text-black;
}

.detail-comment-link:disabled {
  @apply cursor-not-allowed opacity-60;
}
</style>
