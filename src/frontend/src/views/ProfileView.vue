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
    { label: 'Published', value: published },
    { label: 'Likes', value: totalLikes },
    { label: 'Saves', value: totalFavorites },
    { label: 'Views', value: totalViews }
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
    message.success('Profile updated')
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
    title: 'Delete prompt',
    content: 'This will remove the prompt from your published list.',
    positiveText: 'Delete',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      await promptApi.deletePrompt(promptId)
      promptStore.removePrompt(promptId)
      prompts.value = prompts.value.filter((item) => item.id !== promptId)
      message.success('Prompt deleted')
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
            Profile
          </div>
          <h1 class="mt-2 text-3xl font-semibold">
            Creator dashboard
          </h1>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <RouterLink
            to="/"
            class="rounded-full border border-black/10 bg-white px-4 py-2 text-sm text-[#555555] transition hover:border-black/20 hover:text-black"
          >
            Back home
          </RouterLink>
          <RouterLink
            to="/publish"
            class="rounded-full bg-black px-4 py-2 text-sm font-medium text-white transition hover:bg-black/85"
          >
            Publish prompt
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
                  {{ profileUser?.username || 'Creator' }}
                </div>
                <div class="mt-1 text-sm text-[#666666]">
                  {{ profileUser?.email || 'No email available' }}
                </div>
                <p class="mt-3 text-sm leading-6 text-[#5f5f5f]">
                  {{ profileUser?.bio || 'No bio yet. Publish a few prompts and shape this page into a real creator surface.' }}
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
              Edit profile
            </div>
            <div class="mt-4 grid gap-3">
              <label class="grid gap-1 text-sm text-[#555555]">
                Display name
                <input
                  v-model="profileForm.username"
                  type="text"
                  maxlength="20"
                  class="rounded-[14px] border border-black/10 bg-[#faf8f4] px-3 py-2 text-sm text-black outline-none focus:border-black/30"
                >
              </label>
              <label class="grid gap-1 text-sm text-[#555555]">
                Bio
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
                {{ savingProfile ? 'Saving...' : 'Save profile' }}
              </button>
            </div>
          </section>

          <section class="rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
            <div class="text-lg font-semibold text-black">
              Publishing profile
            </div>
            <div class="mt-4 space-y-3 text-sm text-[#4f4f4f]">
              <div>Joined: {{ profileUser?.createdAt || '-' }}</div>
              <div>Level: {{ profileUser?.level ?? '-' }}</div>
              <div>Experience: {{ profileUser?.experience ?? '-' }}</div>
              <div>Top models: {{ favoriteModels.map(([name]) => name).join(', ') || 'No model usage yet' }}</div>
            </div>
          </section>
        </aside>

        <section class="min-w-0 rounded-[28px] border border-black/8 bg-white p-6 shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
          <div class="flex flex-wrap items-end justify-between gap-3">
            <div>
              <div class="text-sm text-[#777777]">
                Published prompts
              </div>
              <div class="mt-1 text-2xl font-semibold text-black">
                {{ prompts.length }} items
              </div>
            </div>
            <div class="text-sm text-[#666666]">
              Sorted by latest
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
                      Edit
                    </button>
                    <button
                      class="rounded-full border border-red-200 bg-white px-3 py-1.5 text-xs text-red-600 transition hover:border-red-300 hover:bg-red-50"
                      @click="handleDeletePrompt(prompt.id)"
                    >
                      Delete
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
                    <span>{{ formatCount(prompt.likes) }} likes</span>
                    <span>{{ formatCount(prompt.views) }} views</span>
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
              No published prompts yet
            </div>
            <p class="mt-3 text-sm text-[#777777]">
              The profile is ready. It just needs the first few prompts to give it some gravity.
            </p>
            <RouterLink
              to="/publish"
              class="mt-6 inline-flex rounded-full bg-black px-5 py-3 text-sm font-medium text-white transition hover:bg-black/85"
            >
              Publish the first one
            </RouterLink>
          </div>
        </section>
      </section>
    </div>
  </div>
</template>
