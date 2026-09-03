<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useDialog, useMessage } from 'naive-ui'
import { Download, Eraser, UserRoundX } from 'lucide-vue-next'
import { promptApi } from '@/api/promptApi'
import { userApi } from '@/api/userApi'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import type { FollowStatus, Prompt, User } from '@/types'
import { githubAuthUrl, githubOAuthEnabled } from '@/utils/authUrl'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'
import BackButton from '@/components/navigation/BackButton.vue'
import AppShell from '@/components/layout/AppShell.vue'
import PageLoading from '@/components/feedback/PageLoading.vue'

type LibraryTab = 'published' | 'drafts' | 'favorites' | 'likes' | 'history' | 'following' | 'followers'

const route = useRoute()
const router = useRouter()
const dialog = useDialog()
const message = useMessage()
const promptStore = usePromptStore()
const userStore = useUserStore()

const loading = ref(false)
const savingProfile = ref(false)
const uploadingAvatar = ref(false)
const prompts = ref<Prompt[]>([])
const draftPrompts = ref<Prompt[]>([])
const favoritePrompts = ref<Prompt[]>([])
const likedPrompts = ref<Prompt[]>([])
const historyPrompts = ref<Prompt[]>([])
const historyTotal = ref(0)
const historyPage = ref(1)
const historyLoadingMore = ref(false)
const exportingData = ref(false)
const clearingHistory = ref(false)
const deletingAccount = ref(false)
const followingUsers = ref<User[]>([])
const followerUsers = ref<User[]>([])
const followStatus = ref<FollowStatus | null>(null)
const activeLibraryTab = ref<LibraryTab>('published')
const profileUser = ref<User | null>(null)
const avatarInput = ref<HTMLInputElement | null>(null)
const profileForm = reactive({
  username: '',
  bio: '',
  avatar: ''
})

const viewedUserId = computed(() => Number(route.params.userId) || userStore.userInfo?.id || 0)
const isOwnerView = computed(() => Boolean(userStore.userInfo && viewedUserId.value === userStore.userInfo.id))
const resolvedAvatar = computed(() => {
  const avatar = profileForm.avatar || profileUser.value?.avatar || ''
  return avatar ? resolveMediaUrl(avatar) : ''
})

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
    { label: '浏览', value: totalViews },
    { label: '关注', value: isOwnerView.value ? followingUsers.value.length : followStatus.value?.followingCount ?? 0 },
    { label: '粉丝', value: isOwnerView.value ? followerUsers.value.length : followStatus.value?.followerCount ?? 0 }
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

const activePromptList = computed(() => {
  if (activeLibraryTab.value === 'favorites') {
    return favoritePrompts.value
  }
  if (activeLibraryTab.value === 'drafts') {
    return draftPrompts.value
  }
  if (activeLibraryTab.value === 'likes') {
    return likedPrompts.value
  }
  if (activeLibraryTab.value === 'history') {
    return historyPrompts.value
  }

  return prompts.value
})

const activeUserList = computed(() => {
  if (activeLibraryTab.value === 'following') {
    return followingUsers.value
  }
  if (activeLibraryTab.value === 'followers') {
    return followerUsers.value
  }

  return []
})

const isUserLibraryTab = computed(() => activeLibraryTab.value === 'following' || activeLibraryTab.value === 'followers')

const libraryTabs = computed(() => [
  { key: 'published' as const, label: '已发布', count: prompts.value.length },
  { key: 'drafts' as const, label: '草稿', count: draftPrompts.value.length },
  { key: 'favorites' as const, label: '收藏', count: favoritePrompts.value.length },
  { key: 'likes' as const, label: '点赞', count: likedPrompts.value.length },
  { key: 'history' as const, label: '浏览', count: historyTotal.value },
  { key: 'following' as const, label: '关注', count: followingUsers.value.length },
  { key: 'followers' as const, label: '粉丝', count: followerUsers.value.length }
])

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
  profileForm.avatar = profileUser.value.avatar
}

const handleSaveProfile = async () => {
  if (!isOwnerView.value) {
    return
  }

  savingProfile.value = true
  try {
    const updated = await userStore.updateProfile({
      username: profileForm.username.trim(),
      bio: profileForm.bio.trim(),
      avatar: profileForm.avatar.trim()
    })
    profileUser.value = updated
    syncProfileForm()
    message.success('资料已更新')
  } catch {
    message.error('资料保存失败，请稍后重试')
  } finally {
    savingProfile.value = false
  }
}

const handleAvatarClick = () => {
  if (!isOwnerView.value || uploadingAvatar.value) {
    return
  }

  avatarInput.value?.click()
}

const handleAvatarUpload = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''

  if (!file) {
    return
  }

  if (!file.type.startsWith('image/')) {
    message.error('请选择图片文件')
    return
  }

  uploadingAvatar.value = true
  try {
    const uploadRes = await promptApi.uploadCover(file)
    const updated = await userStore.updateProfile({
      username: profileForm.username.trim(),
      bio: profileForm.bio.trim(),
      avatar: uploadRes.data.url
    })
    profileUser.value = updated
    syncProfileForm()
    message.success('头像已更新')
  } catch {
    message.error('头像上传失败，请稍后重试')
  } finally {
    uploadingAvatar.value = false
  }
}

const formatCount = (value: number) => {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}k`
  }

  return `${value}`
}

const loadSocialData = async () => {
  if (!userStore.isLoggedIn || !isOwnerView.value) {
    followingUsers.value = []
    followerUsers.value = []
    return
  }

  try {
    const [followingRes, followersRes] = await Promise.all([
      userApi.getFollowingUsers(),
      userApi.getFollowerUsers()
    ])
    followingUsers.value = followingRes.data
    followerUsers.value = followersRes.data
  } catch {
    followingUsers.value = []
    followerUsers.value = []
  }
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
    prompts.value = []
    message.error('个人内容加载失败，请稍后重试')
  } finally {
    loading.value = false
  }

  if (userStore.userInfo && viewedUserId.value === userStore.userInfo.id) {
    await userStore.fetchUserInfo()
    profileUser.value = userStore.userInfo
    try {
      const [favoritesRes, likesRes, historyRes, draftsRes] = await Promise.all([
        userApi.getFavoritePrompts(),
        userApi.getLikedPrompts(),
        userApi.getHistoryPrompts(),
        promptApi.getMyDraftPrompts()
      ])
      favoritePrompts.value = favoritesRes.data
      likedPrompts.value = likesRes.data
      historyPrompts.value = historyRes.data.list
      historyTotal.value = historyRes.data.total
      historyPage.value = historyRes.data.page
      draftPrompts.value = draftsRes.data
    } catch {
      favoritePrompts.value = []
      likedPrompts.value = []
      historyPrompts.value = []
      historyTotal.value = 0
      historyPage.value = 1
      draftPrompts.value = []
    }
    await loadSocialData()
    followStatus.value = null
    syncProfileForm()
    return
  }

  favoritePrompts.value = []
  draftPrompts.value = []
  likedPrompts.value = []
  historyPrompts.value = []
  historyTotal.value = 0
  historyPage.value = 1
  followingUsers.value = []
  followerUsers.value = []
  activeLibraryTab.value = 'published'
  profileUser.value = prompts.value[0]?.user ?? null
  if (userStore.isLoggedIn && viewedUserId.value > 0) {
    try {
      const statusRes = await userApi.getFollowStatus(viewedUserId.value)
      followStatus.value = statusRes.data
    } catch {
      followStatus.value = null
    }
  } else {
    followStatus.value = null
  }
  syncProfileForm()
}

const loadMoreHistory = async () => {
  if (historyLoadingMore.value || historyPrompts.value.length >= historyTotal.value) {
    return
  }

  historyLoadingMore.value = true
  try {
    const response = await userApi.getHistoryPrompts(historyPage.value + 1)
    const existing = new Set(historyPrompts.value.map((prompt) => prompt.id))
    historyPrompts.value = [
      ...historyPrompts.value,
      ...response.data.list.filter((prompt) => !existing.has(prompt.id))
    ]
    historyTotal.value = response.data.total
    historyPage.value = response.data.page
  } finally {
    historyLoadingMore.value = false
  }
}

const handleEditPrompt = async (promptId: number) => {
  await router.push(`/publish?edit=${promptId}`)
}

const promptDetailTarget = (prompt: Prompt) => {
  if (prompt.status === 0) {
    return `/publish?edit=${prompt.id}`
  }

  return `/prompt/${prompt.id}`
}

const handleDeletePrompt = (promptId: number) => {
  dialog.warning({
    title: '删除 Prompt',
    content: '该 Prompt 将从你的已发布列表中移除。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await promptApi.deletePrompt(promptId)
      promptStore.removePrompt(promptId)
      prompts.value = prompts.value.filter((item) => item.id !== promptId)
      draftPrompts.value = draftPrompts.value.filter((item) => item.id !== promptId)
      message.success('Prompt 已删除')
    }
  })
}

const handleBindGitHub = () => {
  window.location.href = githubAuthUrl()
}

const handleExportData = async () => {
  if (!isOwnerView.value || exportingData.value) {
    return
  }

  exportingData.value = true
  try {
    const response = await userApi.exportData()
    const blob = new Blob([JSON.stringify(response.data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `promptos-data-${userStore.userInfo?.id ?? 'account'}.json`
    anchor.click()
    URL.revokeObjectURL(url)
    message.success('数据导出已开始')
  } catch {
    message.error('数据导出失败，请稍后重试')
  } finally {
    exportingData.value = false
  }
}

const handleClearHistory = () => {
  if (!isOwnerView.value || clearingHistory.value) {
    return
  }

  dialog.warning({
    title: '清空浏览记录',
    content: '这会永久删除你的浏览历史，但不会影响 Prompt 的累计浏览数。',
    positiveText: '清空记录',
    negativeText: '取消',
    onPositiveClick: async () => {
      clearingHistory.value = true
      try {
        await userApi.clearHistory()
        historyPrompts.value = []
        historyTotal.value = 0
        historyPage.value = 1
        message.success('浏览记录已清空')
      } catch {
        message.error('浏览记录清空失败，请稍后重试')
      } finally {
        clearingHistory.value = false
      }
    }
  })
}

const handleDeleteAccount = () => {
  if (!isOwnerView.value || deletingAccount.value) {
    return
  }

  dialog.warning({
    title: '注销账号',
    content: '账号会立即失效，个人标识会被匿名化，已发布内容按保留策略留存且不再公开。此操作不可撤销。',
    positiveText: '确认注销',
    negativeText: '取消',
    onPositiveClick: async () => {
      deletingAccount.value = true
      try {
        await userApi.deleteAccount()
        userStore.logout()
        await router.replace('/')
      } catch {
        message.error('账号注销失败，请稍后重试')
      } finally {
        deletingAccount.value = false
      }
    }
  })
}

const handlePublishClick = async () => {
  if (!userStore.isLoggedIn) {
    await router.push('/login?redirect=/publish')
    return
  }
  await router.push('/publish')
}

onMounted(loadProfile)
watch(() => route.params.userId, loadProfile)
</script>

<template>
  <AppShell>
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
            <BackButton
              fallback="/"
              label="返回"
              aria-label="返回上一页或首页"
            />
            <button
              type="button"
              class="btn-pill-primary"
              @click="handlePublishClick"
            >
              发布 Prompt
            </button>
          </div>
        </header>

        <section class="profile-layout">
          <aside class="profile-sidebar">
            <section class="profile-card">
              <div class="profile-card__user">
                <button
                  class="profile-avatar"
                  :class="{ 'profile-avatar--clickable': isOwnerView }"
                  type="button"
                  aria-label="更换头像"
                  :disabled="!isOwnerView || uploadingAvatar"
                  @click="handleAvatarClick"
                >
                  <img
                    v-if="resolvedAvatar"
                    :src="resolvedAvatar"
                    :alt="profileUser?.username || 'avatar'"
                    class="profile-avatar__image"
                  >
                  <span v-else>{{ profileUser?.username?.slice(0, 1) || '?' }}</span>
                  <span
                    v-if="isOwnerView"
                    class="profile-avatar__hint"
                  >
                    {{ uploadingAvatar ? '上传中' : '更换' }}
                  </span>
                </button>
                <input
                  ref="avatarInput"
                  type="file"
                  accept="image/png,image/jpeg,image/webp,image/gif"
                  class="profile-avatar-input"
                  aria-label="选择头像图片"
                  @change="handleAvatarUpload"
                >
                <div class="profile-card__info">
                  <div class="profile-card__name">
                    {{ profileUser?.username || '创作者' }}
                  </div>
                  <div class="profile-card__email">
                    {{ profileUser?.email || '暂无邮箱' }}
                  </div>
                  <p class="profile-card__bio">
                    {{ profileUser?.bio || '还没有简介。发布几条 Prompt 后，这里会更像真正的创作者主页。' }}
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
              v-if="isOwnerView && githubOAuthEnabled"
              class="profile-card"
            >
              <div class="profile-card__title">
                GitHub 账号
              </div>
              <div class="profile-bind-status">
                <span
                  class="profile-bind-dot"
                  :class="{ 'profile-bind-dot--active': profileUser?.hasGitHubBound }"
                />
                <span>{{ profileUser?.hasGitHubBound ? '已绑定 GitHub' : '未绑定 GitHub' }}</span>
              </div>
              <p class="profile-bind-desc">
                绑定后可以使用 GitHub 一键登录；如果邮箱一致，也会自动关联当前账号。
              </p>
              <button
                v-if="!profileUser?.hasGitHubBound"
                class="btn-pill-primary profile-bind-action"
                @click="handleBindGitHub"
              >
                绑定 GitHub
              </button>
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
                <label class="profile-field">
                  头像地址
                  <input
                    v-model="profileForm.avatar"
                    type="text"
                    class="profile-input"
                    placeholder="上传头像后自动填充"
                  >
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

            <section
              v-if="isOwnerView"
              class="profile-card"
            >
              <div class="profile-card__title">
                账户数据
              </div>
              <div class="profile-account-actions">
                <button
                  type="button"
                  class="profile-account-action"
                  :disabled="exportingData"
                  @click="handleExportData"
                >
                  <Download
                    :size="16"
                    aria-hidden="true"
                  />
                  {{ exportingData ? '准备导出...' : '导出我的数据' }}
                </button>
                <button
                  type="button"
                  class="profile-account-action"
                  :disabled="clearingHistory || historyTotal === 0"
                  @click="handleClearHistory"
                >
                  <Eraser
                    :size="16"
                    aria-hidden="true"
                  />
                  {{ clearingHistory ? '清理中...' : '清空浏览记录' }}
                </button>
                <button
                  type="button"
                  class="profile-account-action profile-account-action--danger"
                  :disabled="deletingAccount"
                  @click="handleDeleteAccount"
                >
                  <UserRoundX
                    :size="16"
                    aria-hidden="true"
                  />
                  {{ deletingAccount ? '注销中...' : '注销账号' }}
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
                  内容库
                </div>
                <div class="profile-prompts__count">
                  {{ isUserLibraryTab ? activeUserList.length : activePromptList.length }} 条
                </div>
              </div>
              <div
                v-if="isOwnerView"
                class="profile-library-tabs"
              >
                <button
                  v-for="tab in libraryTabs"
                  :key="tab.key"
                  class="profile-library-tab"
                  :class="{ 'profile-library-tab--active': activeLibraryTab === tab.key }"
                  @click="activeLibraryTab = tab.key"
                >
                  {{ tab.label }} · {{ tab.count }}
                </button>
              </div>
              <div
                v-else
                class="profile-prompts__sort"
              >
                按最新排序
              </div>
            </div>

            <PageLoading
              v-if="loading"
              variant="grid"
              :rows="4"
              label="正在加载个人内容"
            />

            <div
              v-else-if="isUserLibraryTab && activeUserList.length > 0"
              class="profile-users-list"
            >
              <RouterLink
                v-for="user in activeUserList"
                :key="user.id"
                :to="`/profile/${user.id}`"
                class="profile-user-row"
              >
                <div class="profile-user-row__avatar">
                  <img
                    v-if="user.avatar"
                    :src="resolveMediaUrl(user.avatar)"
                    :alt="user.username"
                    class="profile-user-row__image"
                    width="48"
                    height="48"
                    loading="lazy"
                    decoding="async"
                  >
                  <span v-else>{{ user.username.slice(0, 1) }}</span>
                </div>
                <div class="profile-user-row__body">
                  <div class="profile-user-row__name">
                    {{ user.username }}
                  </div>
                  <p class="profile-user-row__bio">
                    {{ user.bio || '这个创作者还没有填写简介。' }}
                  </p>
                </div>
                <div class="profile-user-row__meta">
                  Lv.{{ user.level }}
                </div>
              </RouterLink>
            </div>

            <div
              v-else-if="!isUserLibraryTab && activePromptList.length > 0"
              class="profile-prompts__grid"
            >
              <article
                v-for="(prompt, index) in activePromptList"
                :key="prompt.id"
                class="profile-prompt-card"
              >
                <RouterLink
                  :to="promptDetailTarget(prompt)"
                  class="profile-prompt-card__cover"
                >
                  <img
                    :src="resolveCover(prompt, index)"
                    :alt="prompt.title"
                    class="profile-prompt-card__image"
                    width="1200"
                    height="675"
                    loading="lazy"
                    decoding="async"
                  >
                </RouterLink>
                <div class="profile-prompt-card__body">
                  <div class="profile-prompt-card__head">
                    <div class="profile-prompt-card__meta">
                      <span>{{ prompt.categoryName }}</span>
                      <span>{{ prompt.model }}</span>
                      <span
                        v-if="prompt.status === 0"
                        class="profile-prompt-card__badge"
                      >
                        草稿
                      </span>
                    </div>
                    <div
                      v-if="isOwnerView && (activeLibraryTab === 'published' || activeLibraryTab === 'drafts')"
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
                  <RouterLink :to="promptDetailTarget(prompt)">
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

            <button
              v-if="activeLibraryTab === 'history' && historyPrompts.length < historyTotal"
              type="button"
              class="profile-history-more"
              :disabled="historyLoadingMore"
              @click="loadMoreHistory"
            >
              {{ historyLoadingMore ? '加载中...' : '加载更多浏览记录' }}
            </button>

            <div
              v-else-if="(isUserLibraryTab && activeUserList.length === 0) || (!isUserLibraryTab && activePromptList.length === 0)"
              class="profile-empty"
            >
              <div class="profile-empty__title">
                暂无内容
              </div>
              <p class="profile-empty__desc">
                当前列表还没有内容。发布、收藏、点赞或关注创作者后会在这里形成你的记录。
              </p>
              <RouterLink
                v-if="activeLibraryTab === 'published' || activeLibraryTab === 'drafts'"
                to="/publish"
                class="btn-pill-primary profile-empty__cta"
              >
                {{ activeLibraryTab === 'drafts' ? '新建草稿' : '发布第一条' }}
              </RouterLink>
            </div>
          </section>
        </section>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.profile-page {
  @apply view-page;
}

.profile-container {
  @apply view-container;
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
  @apply rounded-[28px] border border-[var(--prompt-border)] bg-[var(--prompt-surface)] p-6;
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.06);
}

.profile-card__user {
  @apply flex items-start gap-4;
}

.profile-avatar {
  @apply relative flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-full bg-[var(--prompt-primary)] text-xl font-semibold text-[var(--prompt-primary-contrast)] transition disabled:cursor-default;
}

.profile-avatar--clickable {
  @apply cursor-pointer hover:ring-2 hover:ring-black/20;
}

.profile-avatar__image {
  @apply h-full w-full object-cover;
}

.profile-avatar__hint {
  @apply absolute inset-x-0 bottom-0 bg-black/70 py-0.5 text-[10px] font-medium text-white;
}

.profile-avatar-input {
  @apply hidden;
}

.profile-card__info {
  @apply min-w-0;
}

.profile-card__name {
  @apply text-2xl font-semibold text-[var(--prompt-text)];
}

.profile-card__email {
  @apply mt-1 text-sm text-[var(--prompt-text-faint)];
}

.profile-card__bio {
  @apply mt-3 text-sm leading-6 text-[var(--prompt-text-faint)];
}

.profile-stats {
  @apply mt-6 grid grid-cols-2 gap-3;
}

.profile-stat {
  @apply rounded-[18px] border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] p-4;
}

.profile-stat__label {
  @apply text-sm text-[var(--prompt-text-faint)];
}

.profile-stat__value {
  @apply mt-2 text-2xl font-semibold text-[var(--prompt-text)];
}

.profile-card__title {
  @apply text-lg font-semibold text-[var(--prompt-text)];
}

.profile-form {
  @apply mt-4 grid gap-3;
}

.profile-field {
  @apply grid gap-1 text-sm text-[var(--prompt-text-muted)];
}

.profile-input {
  @apply rounded-[14px] border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] px-3 py-2 text-sm text-[var(--prompt-text)] focus:border-[var(--prompt-border-strong)];
}

.profile-save {
  @apply disabled:opacity-60;
}

.profile-summary {
  @apply mt-4 space-y-3 text-sm text-[var(--prompt-text-faint)];
}

.profile-account-actions {
  display: grid;
  gap: 8px;
}

.profile-account-action {
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  min-height: 36px;
  padding: 8px 12px;
  border: 1px solid var(--color-border, #d9dde5);
  border-radius: 6px;
  background: transparent;
  color: var(--color-text, #1f2937);
  cursor: pointer;
}

.profile-account-action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.profile-account-action--danger {
  border-color: #e3a4a4;
  color: #a12b2b;
}

.profile-bind-status {
  @apply mt-4 flex items-center gap-2 text-sm font-medium text-[var(--prompt-text)];
}

.profile-bind-dot {
  @apply h-2.5 w-2.5 rounded-full bg-[var(--prompt-warning)];
}

.profile-bind-dot--active {
  @apply bg-[var(--prompt-success)];
}

.profile-bind-desc {
  @apply mt-3 text-sm leading-6 text-[var(--prompt-text-faint)];
}

.profile-bind-action {
  @apply mt-4 inline-flex px-5 py-3;
}

.profile-prompts {
  @apply min-w-0;
}

.profile-prompts__head {
  @apply flex flex-wrap items-end justify-between gap-3;
}

.profile-prompts__count {
  @apply mt-1 text-2xl font-semibold text-[var(--prompt-text)];
}

.profile-prompts__sort {
  @apply text-sm text-[var(--prompt-text-faint)];
}

.profile-library-tabs {
  @apply flex flex-wrap items-center gap-2;
}

.profile-library-tab {
  @apply rounded-full border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] px-3 py-1.5 text-xs text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border-strong)] hover:text-[var(--prompt-text)];
}

.profile-library-tab--active {
  @apply border-[var(--prompt-border)] bg-[var(--prompt-primary)] text-[var(--prompt-primary-contrast)] hover:text-[var(--prompt-primary-contrast)];
}

.profile-prompts__grid {
  @apply mt-6 grid gap-4 md:grid-cols-2;
}

.profile-history-more {
  @apply mt-6 w-full rounded-full border border-[var(--prompt-border)] px-4 py-3 text-sm text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border-strong)] disabled:cursor-not-allowed disabled:opacity-60;
}

.profile-users-list {
  @apply mt-6 grid gap-3;
}

.profile-user-row {
  @apply flex items-center gap-4 rounded-[20px] border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] p-4 transition hover:-translate-y-0.5 hover:border-[var(--prompt-border-strong)] hover:bg-[var(--prompt-surface)];
}

.profile-user-row__avatar {
  @apply flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-full bg-[var(--prompt-primary)] text-sm font-semibold text-[var(--prompt-primary-contrast)];
}

.profile-user-row__image {
  @apply h-full w-full object-cover;
}

.profile-user-row__body {
  @apply min-w-0 flex-1;
}

.profile-user-row__name {
  @apply text-sm font-semibold text-[var(--prompt-text)];
}

.profile-user-row__bio {
  @apply mt-1 line-clamp-1 text-sm text-[var(--prompt-text-faint)];
}

.profile-user-row__meta {
  @apply rounded-full border border-[var(--prompt-border)] bg-[var(--prompt-surface)] px-3 py-1 text-xs text-[var(--prompt-text-muted)];
}

.profile-skeleton {
  @apply h-[280px] animate-pulse rounded-[24px] bg-[var(--prompt-surface-muted)];
}

.profile-prompt-card {
  @apply overflow-hidden rounded-[24px] border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] transition hover:-translate-y-1;
}

.profile-prompt-card__cover {
  @apply block h-[180px] overflow-hidden bg-[var(--prompt-surface-muted)];
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
  @apply flex items-center gap-3 text-xs uppercase tracking-[0.14em] text-[var(--prompt-text-faint)];
}

.profile-prompt-card__badge {
  @apply rounded-full bg-[var(--prompt-primary)] px-2 py-0.5 text-[10px] font-medium text-[var(--prompt-primary-contrast)];
}

.profile-prompt-card__actions {
  @apply flex items-center gap-2;
}

.profile-action-btn {
  @apply rounded-full border border-[var(--prompt-border)] bg-[var(--prompt-surface)] px-3 py-1.5 text-xs text-[var(--prompt-text-muted)] transition hover:border-[var(--prompt-border-strong)] hover:text-[var(--prompt-text)];
}

.profile-action-btn--danger {
  @apply border-[var(--prompt-error)] text-[var(--prompt-error)] hover:border-[var(--prompt-error)] hover:bg-[var(--prompt-error-bg)];
}

.profile-prompt-card__title {
  @apply mt-3 line-clamp-2 text-xl font-semibold text-[var(--prompt-text)];
}

.profile-prompt-card__desc {
  @apply mt-2 line-clamp-2 text-sm leading-6 text-[var(--prompt-text-faint)];
}

.profile-prompt-card__footer {
  @apply mt-4 flex items-center justify-between gap-3 text-sm text-[var(--prompt-text-faint)];
}

.profile-prompt-card__stats {
  @apply flex items-center gap-3;
}

.profile-empty {
  @apply mt-6 rounded-[24px] border border-dashed border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] px-6 py-16 text-center;
}

.profile-empty__title {
  @apply text-xl font-semibold text-[var(--prompt-text)];
}

.profile-empty__desc {
  @apply mt-3 text-sm text-[var(--prompt-text-faint)];
}

.profile-empty__cta {
  @apply mt-6 inline-flex px-5 py-3;
}
</style>
