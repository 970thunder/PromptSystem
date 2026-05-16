<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import type { Prompt } from '@/types'

const promptStore = usePromptStore()
const userStore = useUserStore()
const router = useRouter()
const activeCategoryId = ref<number | 'all'>('all')

const navItems = [
  { label: '发现', to: '/' },
  { label: '图片 Prompt', to: '/' },
  { label: '工作流', to: '/search?tab=workflow' },
  { label: 'Agent', to: '/search?tab=agent' }
]

const highlightTags = ['电影感', '电商海报', '3D 插画', '品牌视觉', '短视频镜头', '界面概念图']

const fallbackCoverMap: Record<number, string> = {
  101: 'https://images.unsplash.com/photo-1516321497487-e288fb19713f?auto=format&fit=crop&w=1200&q=80',
  102: 'https://images.unsplash.com/photo-1498050108023-c5249f4df085?auto=format&fit=crop&w=1200&q=80',
  103: 'https://images.unsplash.com/photo-1518770660439-4636190af475?auto=format&fit=crop&w=1200&q=80',
  104: 'https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=1200&q=80',
  105: 'https://images.unsplash.com/photo-1526379095098-d400fd0bf935?auto=format&fit=crop&w=1200&q=80',
  106: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?auto=format&fit=crop&w=1200&q=80'
}

const cardSizePattern = [
  'md:col-span-2 md:row-span-2',
  'md:col-span-1 md:row-span-1',
  'md:col-span-1 md:row-span-2',
  'md:col-span-1 md:row-span-1',
  'md:col-span-2 md:row-span-1',
  'md:col-span-1 md:row-span-1'
]

const imageHeightPattern = [
  'h-[420px] md:h-full',
  'h-[240px] md:h-full',
  'h-[320px] md:h-full',
  'h-[240px] md:h-full',
  'h-[260px] md:h-full',
  'h-[240px] md:h-full'
]

onMounted(() => {
  promptStore.loadHomeFeed()
})

const visiblePrompts = computed(() => {
  if (activeCategoryId.value === 'all') {
    return promptStore.prompts
  }

  return promptStore.prompts.filter((prompt) => prompt.categoryId === activeCategoryId.value)
})

const featuredPrompt = computed(() => visiblePrompts.value[0] ?? promptStore.prompts[0] ?? null)

const curatedPrompts = computed(() =>
  visiblePrompts.value.map((prompt, index) => ({
    ...prompt,
    image: resolveCover(prompt, index),
    cardClass: cardSizePattern[index % cardSizePattern.length],
    imageClass: imageHeightPattern[index % imageHeightPattern.length]
  }))
)

const communityStats = computed(() => [
  { label: '精选灵感', value: `${promptStore.prompts.length * 24}+` },
  { label: '活跃创作者', value: '320+' },
  { label: '本周收藏', value: '1.8k' }
])

const formatCount = (value: number) => {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}k`
  }

  return `${value}`
}

const resolveCover = (prompt: Prompt, index: number) => {
  if (/^https?:\/\//.test(prompt.cover) || prompt.cover.startsWith('data:image')) {
    return prompt.cover
  }

  return fallbackCoverMap[prompt.id] ?? fallbackCoverMap[101 + (index % Object.keys(fallbackCoverMap).length)]
}

const handlePublishClick = async () => {
  if (!userStore.isLoggedIn) {
    await router.push('/login?redirect=/publish')
    return
  }

  await router.push('/publish')
}

const handleLogout = async () => {
  userStore.logout()
  await router.push('/')
}
</script>

<template>
  <div class="min-h-screen bg-[#f5f3ee] text-[#111111]">
    <div class="mx-auto max-w-7xl px-4 pb-16 pt-5 sm:px-6 lg:px-8">
      <header class="sticky top-4 z-30 rounded-[24px] border border-black/10 bg-white/80 px-4 py-3 shadow-[0_16px_40px_rgba(15,23,42,0.08)] backdrop-blur md:px-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex items-center justify-between gap-4">
            <RouterLink
              to="/"
              class="text-lg font-semibold tracking-[0.08em] text-[#111111]"
            >
              PromptOS
            </RouterLink>
            <div class="hidden rounded-full border border-black/10 bg-black px-3 py-1 text-xs text-white sm:inline-flex">
              AI Visual Prompt Library
            </div>
          </div>

          <nav class="flex flex-wrap items-center gap-2">
            <RouterLink
              v-for="item in navItems"
              :key="item.label"
              :to="item.to"
              class="rounded-full px-3 py-2 text-sm text-[#555555] transition hover:bg-black hover:text-white"
            >
              {{ item.label }}
            </RouterLink>
          </nav>

          <div class="flex flex-wrap items-center justify-end gap-2">
            <RouterLink
              to="/search"
              class="rounded-full border border-black/10 bg-[#f7f5f0] px-4 py-2 text-sm text-[#555555] transition hover:border-black/20 hover:text-black"
            >
              搜索 Prompt / 风格 / 模型
            </RouterLink>
            <RouterLink
              v-if="!userStore.isLoggedIn"
              to="/login"
              class="rounded-full px-4 py-2 text-sm text-[#555555] transition hover:bg-black hover:text-white"
            >
              登录
            </RouterLink>
            <RouterLink
              v-else
              to="/profile"
              class="rounded-full px-4 py-2 text-sm text-[#555555] transition hover:bg-black hover:text-white"
            >
              {{ userStore.userInfo?.username ?? '个人中心' }}
            </RouterLink>
            <button
              class="rounded-full bg-black px-4 py-2 text-sm font-medium text-white transition hover:bg-black/85"
              @click="handlePublishClick"
            >
              发布
            </button>
            <button
              v-if="userStore.isLoggedIn"
              class="rounded-full border border-black/10 px-4 py-2 text-sm text-[#555555] transition hover:bg-black hover:text-white"
              @click="handleLogout"
            >
              退出
            </button>
          </div>
        </div>
      </header>

      <section class="grid gap-8 pb-10 pt-8 lg:grid-cols-[1.2fr_0.8fr] lg:items-end">
        <div>
          <p class="text-sm uppercase tracking-[0.24em] text-[#777777]">
            Curated visual prompts
          </p>
          <h1 class="mt-3 max-w-3xl text-4xl font-semibold leading-tight sm:text-5xl">
            先看图，再找到能直接开工的 AI Prompt。
          </h1>
          <p class="mt-4 max-w-2xl text-base leading-7 text-[#5f5f5f]">
            首页优先展示高质量 AI 视觉案例，导航保持克制，灵感、分类和作者信息都退到第二层，浏览会更轻一点。
          </p>

          <div class="mt-6 flex flex-wrap gap-2">
            <span
              v-for="tag in highlightTags"
              :key="tag"
              class="rounded-full border border-black/10 bg-white px-3 py-2 text-sm text-[#444444]"
            >
              {{ tag }}
            </span>
          </div>
        </div>

        <div class="grid gap-3 sm:grid-cols-3">
          <article
            v-for="stat in communityStats"
            :key="stat.label"
            class="rounded-[18px] border border-black/8 bg-white px-4 py-4"
          >
            <div class="text-sm text-[#777777]">
              {{ stat.label }}
            </div>
            <div class="mt-2 text-2xl font-semibold text-black">
              {{ stat.value }}
            </div>
          </article>
        </div>
      </section>

      <section
        v-if="featuredPrompt"
        class="grid gap-4 pb-8 lg:grid-cols-[1.15fr_0.85fr]"
      >
        <RouterLink
          :to="`/prompt/${featuredPrompt.id}`"
          class="group relative block overflow-hidden rounded-[28px] bg-black"
        >
          <img
            :src="resolveCover(featuredPrompt, 0)"
            :alt="featuredPrompt.title"
            class="h-[520px] w-full object-cover transition duration-500 group-hover:scale-[1.02]"
          >
          <div class="absolute inset-0 bg-gradient-to-t from-black via-black/15 to-transparent" />
          <div class="absolute inset-x-0 bottom-0 p-6 text-white sm:p-8">
            <div class="flex flex-wrap items-center gap-2 text-xs uppercase tracking-[0.2em] text-white/70">
              <span>{{ featuredPrompt.categoryName }}</span>
              <span>{{ featuredPrompt.model }}</span>
              <span v-if="promptStore.usingMockData">Mock Feed</span>
            </div>
            <h2 class="mt-3 max-w-2xl text-2xl font-semibold sm:text-3xl">
              {{ featuredPrompt.title }}
            </h2>
            <p class="mt-3 max-w-2xl text-sm leading-6 text-white/78 sm:text-base">
              {{ featuredPrompt.description }}
            </p>
            <div class="mt-5 flex flex-wrap items-center gap-4 text-sm text-white/74">
              <span>{{ featuredPrompt.user.username }}</span>
              <span>{{ formatCount(featuredPrompt.likes) }} likes</span>
              <span>{{ formatCount(featuredPrompt.favorites) }} saves</span>
            </div>
          </div>
        </RouterLink>

        <div class="grid gap-4">
          <div class="rounded-[28px] border border-black/8 bg-white p-5">
            <div class="flex items-center justify-between gap-3">
              <div>
                <div class="text-sm text-[#777777]">
                  热门分类
                </div>
                <div class="mt-1 text-xl font-semibold text-black">
                  今天适合从这些方向开始
                </div>
              </div>
              <div class="text-xs uppercase tracking-[0.2em] text-[#999999]">
                Explore
              </div>
            </div>

            <div class="mt-5 flex flex-wrap gap-2">
              <button
                class="rounded-full px-4 py-2 text-sm transition"
                :class="
                  activeCategoryId === 'all'
                    ? 'bg-black text-white'
                    : 'border border-black/10 bg-[#f6f4ef] text-[#555555] hover:border-black/20 hover:text-black'
                "
                @click="activeCategoryId = 'all'"
              >
                全部
              </button>
              <button
                v-for="category in promptStore.categories"
                :key="category.id"
                class="rounded-full px-4 py-2 text-sm transition"
                :class="
                  activeCategoryId === category.id
                    ? 'bg-black text-white'
                    : 'border border-black/10 bg-[#f6f4ef] text-[#555555] hover:border-black/20 hover:text-black'
                "
                @click="activeCategoryId = category.id"
              >
                {{ category.name }} · {{ category.count }}
              </button>
            </div>
          </div>

          <div class="rounded-[28px] border border-black/8 bg-[#111111] p-6 text-white">
            <div class="text-sm text-white/60">
              本周精选作者
            </div>
            <div class="mt-2 text-2xl font-semibold">
              {{ featuredPrompt.user.username }}
            </div>
            <p class="mt-3 text-sm leading-6 text-white/70">
              {{ featuredPrompt.user.bio || '专注把模型能力整理成可复用、可直接上线的创作流程。' }}
            </p>

            <div class="mt-6 grid grid-cols-3 gap-3 text-center">
              <div class="rounded-[18px] bg-white/8 px-3 py-4">
                <div class="text-lg font-semibold">
                  {{ formatCount(featuredPrompt.views) }}
                </div>
                <div class="mt-1 text-xs text-white/55">
                  浏览
                </div>
              </div>
              <div class="rounded-[18px] bg-white/8 px-3 py-4">
                <div class="text-lg font-semibold">
                  {{ formatCount(featuredPrompt.likes) }}
                </div>
                <div class="mt-1 text-xs text-white/55">
                  喜欢
                </div>
              </div>
              <div class="rounded-[18px] bg-white/8 px-3 py-4">
                <div class="text-lg font-semibold">
                  {{ featuredPrompt.model }}
                </div>
                <div class="mt-1 text-xs text-white/55">
                  模型
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="pb-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <div class="text-sm text-[#777777]">
              灵感画廊
            </div>
            <h2 class="mt-1 text-2xl font-semibold text-black">
              按图片浏览可直接使用的 Prompt
            </h2>
          </div>
          <div class="text-sm text-[#777777]">
            {{ visiblePrompts.length }} 个结果
          </div>
        </div>
      </section>

      <section
        v-if="promptStore.loading"
        class="grid auto-rows-[180px] grid-cols-1 gap-4 pb-8 md:grid-cols-3"
      >
        <div
          v-for="index in 6"
          :key="index"
          class="animate-pulse rounded-[24px] bg-black/6"
          :class="cardSizePattern[(index - 1) % cardSizePattern.length]"
        />
      </section>

      <section
        v-else-if="curatedPrompts.length > 0"
        class="grid auto-rows-[180px] grid-cols-1 gap-4 pb-8 md:grid-cols-3"
      >
        <RouterLink
          v-for="prompt in curatedPrompts"
          :key="prompt.id"
          :to="`/prompt/${prompt.id}`"
          class="group relative overflow-hidden rounded-[24px] bg-black"
          :class="prompt.cardClass"
        >
          <img
            :src="prompt.image"
            :alt="prompt.title"
            class="w-full object-cover transition duration-500 group-hover:scale-[1.03]"
            :class="prompt.imageClass"
          >
          <div class="absolute inset-0 bg-gradient-to-t from-black/88 via-black/18 to-transparent" />

          <div class="absolute inset-x-0 bottom-0 p-5 text-white">
            <div class="flex items-center justify-between gap-3 text-xs uppercase tracking-[0.16em] text-white/70">
              <span>{{ prompt.categoryName }}</span>
              <span>{{ prompt.model }}</span>
            </div>
            <h3 class="mt-3 text-xl font-semibold leading-tight">
              {{ prompt.title }}
            </h3>
            <p class="mt-2 line-clamp-2 text-sm leading-6 text-white/75">
              {{ prompt.description }}
            </p>
            <div class="mt-4 flex flex-wrap items-center gap-3 text-sm text-white/72">
              <span>{{ prompt.user.username }}</span>
              <span>{{ formatCount(prompt.likes) }} likes</span>
              <span>{{ formatCount(prompt.views) }} views</span>
            </div>
          </div>
        </RouterLink>
      </section>

      <section
        v-else
        class="rounded-[28px] border border-dashed border-black/12 bg-white px-6 py-16 text-center"
      >
        <div class="text-lg font-semibold text-black">
          这个分类还没有内容
        </div>
        <p class="mt-2 text-sm text-[#777777]">
          可以先看看其它分类，或者直接发布第一个 Prompt。
        </p>
      </section>
    </div>
  </div>
</template>
