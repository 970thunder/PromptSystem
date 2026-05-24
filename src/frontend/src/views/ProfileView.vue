<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useDialog, useMessage } from 'naive-ui'
import { promptApi } from '@/api/promptApi'
import { mockPrompts } from '@/mock/prompts'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import type { Prompt, User } from '@/types'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'

const route = useRoute()
const router = useRouter()
const dialog = useDialog()
const message = useMessage()
const promptStore = usePromptStore()
const userStore = useUserStore()

const loading = ref(false)
const savingProfile = ref(false)
const prompts = ref<Prompt[]>([])
const profileUser = ref<User | null>(null)
const profileForm = reactive({
  username: '',
  bio: ''
})

const viewedUserId = computed(() => Number(route.params.userId) || userStore.userInfo?.id || 0)
const isOwnerView = computed(() => Boolean(userStore.userInfo && viewedUserId.value === userStore.userInfo.id))

const fallbackCoverMap: Record<number, string> = {
  101: 'https://images.unsplash.com/photo-1516321497487-e288fb19713f?auto=format&fit=crop&w=1200&q=80',
  102: 'https://images.unsplash.com/photo-1498050108023-c5249f4df085?auto=format&fit=crop&w=1200&q=80',
  103: 'https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1200&q=80',
  104: 'https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=1200&q=80',
  105: 'https://images.unsplash.com/photo-1526379095098-d400fd0bf935?auto=format&fit=crop&w=1200&q=80',
  106: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80'
}

const stats = computed(() => {
  const published = prompts.value.length
  const totalLikes = prompts.value.reduce((sum, prompt) => sum + prompt.likes, 0)
  const totalFavorites = prompts.value.reduce((sum, prompt) => sum + prompt.favorites, 0)
  const totalViews = prompts.value.reduce((sum, prompt) => sum + prompt.views, 0)

  return [
    { label: '已发布', value: published },
    { label: '获赞', value: totalLikes },
    { label: '收藏', value: totalFavorites },
    { label: '浏览', value: totalViews }
  ]
})

const favoriteModels = computed(() => {
  const counts = new Map<string, number>()
  prompts.value.forEach((prompt) => {
    counts.set(prompt.model, (counts.get(prompt.model) ?? 0) + 1)
  })

  return Array.from(counts.entries())
    .sort((left, right) => right[1] - left[1])
    .slice(0, 3)
})

const resolveCover = (prompt: Prompt, index: number) => {
  if (isDisplayableCover(prompt.cover)) {
    return resolveMediaUrl(prompt.cover)
  }

  return fallbackCoverMap[prompt.id] ?? fallbackCoverMap[101 + (index % Object.keys(fallbackCoverMap).length)]
}

const syncProfileForm = () => {
  if (!profileUser.value) {
    return
  }

  profileForm.username = profileUser.value.username
  profileForm.bio = profileUser.value.bio
}

const handleSaveProfile = async () => {
  if (!isOwnerView.value) {
    return
  }

  savingProfile.value = true
  try {
    const updated = await userStore.updateProfile({
      username: profileForm.username.trim(),
      bio: profileForm.bio.trim()
    })
    profileUser.value = updated
    message.success('资料已更新')
  } finally {
    savingProfile.value = false
  }
}

const formatCount = (value: number) => {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}k`
  }

  return `${value}`
}

const loadProfile = async () => {
  if (promptStore.categories.length === 0) {
    await promptStore.loadHomeFeed()
  }

  loading.value = true
  try {
    const response = await promptApi.getPromptList({
      page: 1,
      pageSize: 24,
      userId: viewedUserId.value,
      sort: 'latest'
    })
    prompts.value = response.data.list
  } catch {
    prompts.value = mockPrompts.filter((prompt) => prompt.userId === viewedUserId.value)
  } finally {
    loading.value = false
  }

  if (userStore.userInfo && viewedUserId.value === userStore.userInfo.id) {
    await userStore.fetchUserInfo()
    profileUser.value = userStore.userInfo
    syncProfileForm()
    return
  }

  profileUser.value = prompts.value[0]?.user ?? null
}

const handleEditPrompt = async (promptId: number) => {
  await router.push(`/publish?edit=${promptId}`)
}

const handleDeletePrompt = (promptId: number) => {
  dialog.warning({
    title: '删除提示词',
    content: '该提示词将从你的已发布列表中移除。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await promptApi.deletePrompt(promptId)
      promptStore.removePrompt(promptId)
      prompts.value = prompts.value.filter((item) => item.id !== promptId)
      message.success('提示词已删除')
    }
  })
}

onMounted(loadProfile)
watch(() => route.params.userId, loadProfile)
</script>

<template>
  <div class="profile-page">
    <div class="profile-container">
      <header class="profile-header">
        <div>
          <div class="section-eyebrow">
            个人主页
          </div>
          <h1 class="profile-header__title">
            创作者工作台
          </h1>
        </div>
        <div class="profile-header__actions">
          <RouterLink
            to="/"
            class="btn-pill-secondary bg-white"
          >
            返回首页
          </RouterLink>
          <RouterLink
            to="/publish"
            class="btn-pill-primary"
          >
            发布提示词
          </RouterLink>
        </div>
      </header>

      <section class="profile-layout">
        <aside class="profile-sidebar">
          <section class="profile-card">
            <div class="profile-card__user">
              <div class="profile-avatar">
                {{ profileUser?.username?.slice(0, 1) || '?' }}
              </div>
              <div class="profile-card__info">
                <div class="profile-card__name">
                  {{ profileUser?.username || '创作者' }}
                </div>
                <div class="profile-card__email">
                  {{ profileUser?.email || '暂无邮箱' }}
                </div>
                <p class="profile-card__bio">
                  {{ profileUser?.bio || '还没有简介。发布几条提示词，让这个页面更像真正的创作者主页。' }}
                </p>
              </div>
            </div>

            <div class="profile-stats">
              <div
                v-for="stat in stats"
                :key="stat.label"
                class="profile-stat"
              >
                <div class="profile-stat__label">
                  {{ stat.label }}
                </div>
                <div class="profile-stat__value">
                  {{ formatCount(stat.value) }}
                </div>
              </div>
            </div>
          </section>

          <section
            v-if="isOwnerView"
            class="profile-card"
          >
            <div class="profile-card__title">
              编辑资料
            </div>
            <div class="profile-form">
              <label class="profile-field">
                展示名称
                <input
                  v-model="profileForm.username"
                  type="text"
                  maxlength="20"
                  class="profile-input"
                >
              </label>
              <label class="profile-field">
                简介
                <textarea
                  v-model="profileForm.bio"
                  rows="3"
                  maxlength="500"
                  class="profile-input"
                />
              </label>
              <button
                class="btn-pill-primary profile-save"
                :disabled="savingProfile"
                @click="handleSaveProfile"
              >
                {{ savingProfile ? '保存中...' : '保存资料' }}
              </button>
            </div>
          </section>

          <section class="profile-card">
            <div class="profile-card__title">
              发布概况
            </div>
            <div class="profile-summary">
              <div>加入时间：{{ profileUser?.createdAt || '-' }}</div>
              <div>等级：{{ profileUser?.level ?? '-' }}</div>
              <div>经验值：{{ profileUser?.experience ?? '-' }}</div>
              <div>常用模型：{{ favoriteModels.map(([name]) => name).join('、') || '暂无使用记录' }}</div>
            </div>
          </section>
        </aside>

        <section class="profile-card profile-prompts">
          <div class="profile-prompts__head">
            <div>
              <div class="text-muted-sm">
                已发布提示词
              </div>
              <div class="profile-prompts__count">
                {{ prompts.length }} 条
              </div>
            </div>
            <div class="profile-prompts__sort">
              按最新排序
            </div>
          </div>

          <div
            v-if="loading"
            class="profile-prompts__grid"
          >
            <div
              v-for="index in 4"
              :key="index"
              class="profile-skeleton"
            />
          </div>

          <div
            v-else-if="prompts.length > 0"
            class="profile-prompts__grid"
          >
            <article
              v-for="(prompt, index) in prompts"
              :key="prompt.id"
              class="profile-prompt-card"
            >
              <RouterLink
                :to="`/prompt/${prompt.id}`"
                class="profile-prompt-card__cover"
              >
                <img
                  :src="resolveCover(prompt, index)"
                  :alt="prompt.title"
                  class="profile-prompt-card__image"
                >
              </RouterLink>
              <div class="profile-prompt-card__body">
                <div class="profile-prompt-card__head">
                  <div class="profile-prompt-card__meta">
                    <span>{{ prompt.categoryName }}</span>
                    <span>{{ prompt.model }}</span>
                  </div>
                  <div
                    v-if="isOwnerView"
                    class="profile-prompt-card__actions"
                  >
                    <button
                      class="profile-action-btn"
                      @click="handleEditPrompt(prompt.id)"
                    >
                      编辑
                    </button>
                    <button
                      class="profile-action-btn profile-action-btn--danger"
                      @click="handleDeletePrompt(prompt.id)"
                    >
                      删除
                    </button>
                  </div>
                </div>
                <RouterLink :to="`/prompt/${prompt.id}`">
                  <h2 class="profile-prompt-card__title">
                    {{ prompt.title }}
                  </h2>
                </RouterLink>
                <p class="profile-prompt-card__desc">
                  {{ prompt.description }}
                </p>
                <div class="profile-prompt-card__footer">
                  <span>{{ prompt.createdAt }}</span>
                  <div class="profile-prompt-card__stats">
                    <span>{{ formatCount(prompt.likes) }} 赞</span>
                    <span>{{ formatCount(prompt.views) }} 浏览</span>
                  </div>
                </div>
              </div>
            </article>
          </div>

          <div
            v-else
            class="profile-empty"
          >
            <div class="profile-empty__title">
              还没有已发布的提示词
            </div>
            <p class="profile-empty__desc">
              个人主页已就绪，发布前几条内容就能让它更有分量。
            </p>
            <RouterLink
              to="/publish"
              class="btn-pill-primary profile-empty__cta"
            >
              发布第一条
            </RouterLink>
          </div>
        </section>
      </section>
    </div>
  </div>
</template>

<style scoped>
.profile-page {
  @apply min-h-screen bg-[#f5f3ee] text-[#111111];
}

.profile-container {
  @apply mx-auto max-w-[1160px] px-4 pb-16 pt-6 sm:px-6 lg:px-8;
}

.profile-header {
  @apply flex flex-wrap items-center justify-between gap-3;
}

.profile-header__title {
  @apply mt-2 text-3xl font-semibold;
}

.profile-header__actions {
  @apply flex flex-wrap items-center gap-2;
}

.profile-layout {
  @apply mt-8 grid gap-6 lg:grid-cols-[0.95fr_1.05fr];
}

.profile-sidebar {
  @apply grid gap-6 self-start;
}

.profile-card {
  @apply rounded-[28px] border border-black/10 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)];
}

.profile-card__user {
  @apply flex items-start gap-4;
}

.profile-avatar {
  @apply flex h-16 w-16 items-center justify-center rounded-full bg-black text-xl font-semibold text-white;
}

.profile-card__info {
  @apply min-w-0;
}

.profile-card__name {
  @apply text-2xl font-semibold text-black;
}

.profile-card__email {
  @apply mt-1 text-sm text-[#666666];
}

.profile-card__bio {
  @apply mt-3 text-sm leading-6 text-[#5f5f5f];
}

.profile-stats {
  @apply mt-6 grid grid-cols-2 gap-3;
}

.profile-stat {
  @apply rounded-[18px] border border-black/10 bg-[#faf8f4] p-4;
}

.profile-stat__label {
  @apply text-sm text-[#777777];
}

.profile-stat__value {
  @apply mt-2 text-2xl font-semibold text-black;
}

.profile-card__title {
  @apply text-lg font-semibold text-black;
}

.profile-form {
  @apply mt-4 grid gap-3;
}

.profile-field {
  @apply grid gap-1 text-sm text-[#555555];
}

.profile-input {
  @apply rounded-[14px] border border-black/10 bg-[#faf8f4] px-3 py-2 text-sm text-black outline-none focus:border-black/30;
}

.profile-save {
  @apply disabled:opacity-60;
}

.profile-summary {
  @apply mt-4 space-y-3 text-sm text-[#4f4f4f];
}

.profile-prompts {
  @apply min-w-0;
}

.profile-prompts__head {
  @apply flex flex-wrap items-end justify-between gap-3;
}

.profile-prompts__count {
  @apply mt-1 text-2xl font-semibold text-black;
}

.profile-prompts__sort {
  @apply text-sm text-[#666666];
}

.profile-prompts__grid {
  @apply mt-6 grid gap-4 md:grid-cols-2;
}

.profile-skeleton {
  @apply h-[280px] animate-pulse rounded-[24px] bg-black/5;
}

.profile-prompt-card {
  @apply overflow-hidden rounded-[24px] border border-black/10 bg-[#faf8f4] transition hover:-translate-y-1;
}

.profile-prompt-card__cover {
  @apply block h-[180px] overflow-hidden bg-[#ebe8e1];
}

.profile-prompt-card__image {
  @apply h-full w-full max-h-full max-w-full object-cover;
}

.profile-prompt-card__body {
  @apply p-5;
}

.profile-prompt-card__head {
  @apply flex items-start justify-between gap-3;
}

.profile-prompt-card__meta {
  @apply flex items-center gap-3 text-xs uppercase tracking-[0.14em] text-[#7c7c7c];
}

.profile-prompt-card__actions {
  @apply flex items-center gap-2;
}

.profile-action-btn {
  @apply rounded-full border border-black/10 bg-white px-3 py-1.5 text-xs text-[#555555] transition hover:border-black/20 hover:text-black;
}

.profile-action-btn--danger {
  @apply border-red-200 text-red-600 hover:border-red-300 hover:bg-red-50;
}

.profile-prompt-card__title {
  @apply mt-3 line-clamp-2 text-xl font-semibold text-black;
}

.profile-prompt-card__desc {
  @apply mt-2 line-clamp-2 text-sm leading-6 text-[#5f5f5f];
}

.profile-prompt-card__footer {
  @apply mt-4 flex items-center justify-between gap-3 text-sm text-[#666666];
}

.profile-prompt-card__stats {
  @apply flex items-center gap-3;
}

.profile-empty {
  @apply mt-6 rounded-[24px] border border-dashed border-black/10 bg-[#faf8f4] px-6 py-16 text-center;
}

.profile-empty__title {
  @apply text-xl font-semibold text-black;
}

.profile-empty__desc {
  @apply mt-3 text-sm text-[#777777];
}

.profile-empty__cta {
  @apply mt-6 inline-flex px-5 py-3;
}
</style>
