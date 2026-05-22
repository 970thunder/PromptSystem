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
  <div class="min-h-screen bg-[#f5f3ee] text-[#111111]">
    <div class="mx-auto max-w-[1160px] px-4 pb-16 pt-6 sm:px-6 lg:px-8">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="text-sm uppercase tracking-[0.2em] text-[#7c7c7c]">
            个人主页
          </div>
          <h1 class="mt-2 text-3xl font-semibold">
            创作者工作台
          </h1>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <RouterLink
            to="/"
            class="rounded-full border border-black/10 bg-white px-4 py-2 text-sm text-[#555555] transition hover:border-black/20 hover:text-black"
          >
            返回首页
          </RouterLink>
          <RouterLink
            to="/publish"
            class="rounded-full bg-black px-4 py-2 text-sm font-medium text-white transition hover:bg-black/85"
          >
            发布提示词
          </RouterLink>
        </div>
      </header>

      <section class="mt-8 grid gap-6 lg:grid-cols-[0.95fr_1.05fr]">
        <aside class="grid gap-6 self-start">
          <section class="rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
            <div class="flex items-start gap-4">
              <div class="flex h-16 w-16 items-center justify-center rounded-full bg-black text-xl font-semibold text-white">
                {{ profileUser?.username?.slice(0, 1) || '?' }}
              </div>
              <div class="min-w-0">
                <div class="text-2xl font-semibold text-black">
                  {{ profileUser?.username || '创作者' }}
                </div>
                <div class="mt-1 text-sm text-[#666666]">
                  {{ profileUser?.email || '暂无邮箱' }}
                </div>
                <p class="mt-3 text-sm leading-6 text-[#5f5f5f]">
                  {{ profileUser?.bio || '还没有简介。发布几条提示词，让这个页面更像真正的创作者主页。' }}
                </p>
              </div>
            </div>

            <div class="mt-6 grid grid-cols-2 gap-3">
              <div
                v-for="stat in stats"
                :key="stat.label"
                class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4"
              >
                <div class="text-sm text-[#777777]">
                  {{ stat.label }}
                </div>
                <div class="mt-2 text-2xl font-semibold text-black">
                  {{ formatCount(stat.value) }}
                </div>
              </div>
            </div>
          </section>

          <section
            v-if="isOwnerView"
            class="rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]"
          >
            <div class="text-lg font-semibold text-black">
              编辑资料
            </div>
            <div class="mt-4 grid gap-3">
              <label class="grid gap-1 text-sm text-[#555555]">
                展示名称
                <input
                  v-model="profileForm.username"
                  type="text"
                  maxlength="20"
                  class="rounded-[14px] border border-black/10 bg-[#faf8f4] px-3 py-2 text-sm text-black outline-none focus:border-black/30"
                >
              </label>
              <label class="grid gap-1 text-sm text-[#555555]">
                简介
                <textarea
                  v-model="profileForm.bio"
                  rows="3"
                  maxlength="500"
                  class="rounded-[14px] border border-black/10 bg-[#faf8f4] px-3 py-2 text-sm text-black outline-none focus:border-black/30"
                />
              </label>
              <button
                class="rounded-full bg-black px-4 py-2 text-sm font-medium text-white transition hover:bg-black/85 disabled:opacity-60"
                :disabled="savingProfile"
                @click="handleSaveProfile"
              >
                {{ savingProfile ? '保存中...' : '保存资料' }}
              </button>
            </div>
          </section>

          <section class="rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
            <div class="text-lg font-semibold text-black">
              发布概况
            </div>
            <div class="mt-4 space-y-3 text-sm text-[#4f4f4f]">
              <div>加入时间：{{ profileUser?.createdAt || '-' }}</div>
              <div>等级：{{ profileUser?.level ?? '-' }}</div>
              <div>经验值：{{ profileUser?.experience ?? '-' }}</div>
              <div>常用模型：{{ favoriteModels.map(([name]) => name).join('、') || '暂无使用记录' }}</div>
            </div>
          </section>
        </aside>

        <section class="min-w-0 rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
          <div class="flex flex-wrap items-end justify-between gap-3">
            <div>
              <div class="text-sm text-[#777777]">
                已发布提示词
              </div>
              <div class="mt-1 text-2xl font-semibold text-black">
                {{ prompts.length }} 条
              </div>
            </div>
            <div class="text-sm text-[#666666]">
              按最新排序
            </div>
          </div>

          <div
            v-if="loading"
            class="mt-6 grid gap-4 md:grid-cols-2"
          >
            <div
              v-for="index in 4"
              :key="index"
              class="h-[280px] animate-pulse rounded-[24px] bg-black/6"
            />
          </div>

          <div
            v-else-if="prompts.length > 0"
            class="mt-6 grid gap-4 md:grid-cols-2"
          >
            <article
              v-for="(prompt, index) in prompts"
              :key="prompt.id"
              class="overflow-hidden rounded-[24px] border border-black/8 bg-[#faf8f4] transition hover:-translate-y-1"
            >
              <RouterLink :to="`/prompt/${prompt.id}`">
                <img
                  :src="resolveCover(prompt, index)"
                  :alt="prompt.title"
                  class="h-[180px] w-full object-cover"
                >
              </RouterLink>
              <div class="p-5">
                <div class="flex items-start justify-between gap-3">
                  <div class="flex items-center gap-3 text-xs uppercase tracking-[0.14em] text-[#7c7c7c]">
                    <span>{{ prompt.categoryName }}</span>
                    <span>{{ prompt.model }}</span>
                  </div>
                  <div
                    v-if="isOwnerView"
                    class="flex items-center gap-2"
                  >
                    <button
                      class="rounded-full border border-black/10 bg-white px-3 py-1.5 text-xs text-[#555555] transition hover:border-black/20 hover:text-black"
                      @click="handleEditPrompt(prompt.id)"
                    >
                      编辑
                    </button>
                    <button
                      class="rounded-full border border-red-200 bg-white px-3 py-1.5 text-xs text-red-600 transition hover:border-red-300 hover:bg-red-50"
                      @click="handleDeletePrompt(prompt.id)"
                    >
                      删除
                    </button>
                  </div>
                </div>
                <RouterLink :to="`/prompt/${prompt.id}`">
                  <h2 class="mt-3 line-clamp-2 text-xl font-semibold text-black">
                    {{ prompt.title }}
                  </h2>
                </RouterLink>
                <p class="mt-2 line-clamp-2 text-sm leading-6 text-[#5f5f5f]">
                  {{ prompt.description }}
                </p>
                <div class="mt-4 flex items-center justify-between gap-3 text-sm text-[#666666]">
                  <span>{{ prompt.createdAt }}</span>
                  <div class="flex items-center gap-3">
                    <span>{{ formatCount(prompt.likes) }} 赞</span>
                    <span>{{ formatCount(prompt.views) }} 浏览</span>
                  </div>
                </div>
              </div>
            </article>
          </div>

          <div
            v-else
            class="mt-6 rounded-[24px] border border-dashed border-black/12 bg-[#faf8f4] px-6 py-16 text-center"
          >
            <div class="text-xl font-semibold text-black">
              还没有已发布的提示词
            </div>
            <p class="mt-3 text-sm text-[#777777]">
              个人主页已就绪，发布前几条内容就能让它更有分量。
            </p>
            <RouterLink
              to="/publish"
              class="mt-6 inline-flex rounded-full bg-black px-5 py-3 text-sm font-medium text-white transition hover:bg-black/85"
            >
              发布第一条
            </RouterLink>
          </div>
        </section>
      </section>
    </div>
  </div>
</template>
