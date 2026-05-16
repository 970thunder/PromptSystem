<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { usePromptStore } from '@/stores/prompt'

const promptStore = usePromptStore()
const activeCategoryId = ref<number | 'all'>('all')

onMounted(() => {
  promptStore.loadHomeFeed()
})

const visiblePrompts = computed(() => {
  if (activeCategoryId.value === 'all') {
    return promptStore.prompts
  }

  return promptStore.prompts.filter((prompt) => prompt.categoryId === activeCategoryId.value)
})

const featuredPrompt = computed(() => promptStore.featuredPrompts[0] ?? null)

const spotlightPrompts = computed(() => promptStore.featuredPrompts.slice(1))

const communityStats = computed(() => [
  { label: '精选 Prompt', value: `${promptStore.prompts.length * 24}+` },
  { label: '活跃创作者', value: '320+' },
  { label: '今日收藏', value: '1.8k' }
])

const formatCount = (value: number) => {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}k`
  }

  return `${value}`
}
</script>

<template>
  <div class="min-h-screen bg-[radial-gradient(circle_at_top,#312E81_0%,#0B0F19_38%,#090B11_100%)] text-white">
    <div class="mx-auto max-w-7xl px-5 pb-16 pt-6 sm:px-8 lg:px-10">
      <header class="glass sticky top-4 z-20 mb-10 flex flex-col gap-4 rounded-card px-5 py-4 shadow-glass lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div class="text-xs uppercase tracking-[0.32em] text-cyan-300/80">
            Prompt Community
          </div>
          <h1 class="mt-1 text-2xl font-semibold text-white sm:text-3xl">
            PromptOS
          </h1>
        </div>

        <div class="flex flex-1 flex-col gap-3 lg:ml-10 lg:flex-row lg:items-center">
          <div class="flex-1 rounded-full border border-white/10 bg-white/5 px-4 py-3 text-sm text-neutral-300">
            搜索 Prompt、工作流、创作者
          </div>
          <div class="flex gap-3">
            <button class="rounded-full border border-white/15 px-4 py-3 text-sm text-white transition hover:bg-white/10">
              登录
            </button>
            <button class="rounded-full bg-cyan-400 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-cyan-300">
              发布 Prompt
            </button>
          </div>
        </div>
      </header>

      <section class="mb-8 grid gap-6 lg:grid-cols-[1.6fr_0.9fr]">
        <div class="overflow-hidden rounded-card border border-white/10 bg-white/6 p-6 shadow-glass">
          <div class="mb-4 flex flex-wrap items-center gap-3">
            <span class="rounded-full border border-cyan-300/40 bg-cyan-300/10 px-3 py-1 text-xs text-cyan-200">
              本周精选
            </span>
            <span
              v-if="promptStore.usingMockData"
              class="rounded-full border border-amber-300/30 bg-amber-300/10 px-3 py-1 text-xs text-amber-200"
            >
              Mock Feed
            </span>
          </div>

          <div
            v-if="featuredPrompt"
            class="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]"
          >
            <div>
              <p class="text-sm text-neutral-300">
                {{ featuredPrompt.categoryName }}
              </p>
              <h2 class="mt-3 max-w-xl text-3xl font-semibold leading-tight text-white sm:text-4xl">
                {{ featuredPrompt.title }}
              </h2>
              <p class="mt-4 max-w-2xl text-base leading-7 text-neutral-300">
                {{ featuredPrompt.description }}
              </p>

              <div class="mt-6 flex flex-wrap gap-2">
                <span
                  v-for="tag in featuredPrompt.tags"
                  :key="tag"
                  class="rounded-full bg-white/8 px-3 py-1 text-xs text-neutral-200"
                >
                  {{ tag }}
                </span>
              </div>

              <div class="mt-8 flex flex-wrap items-center gap-6 text-sm text-neutral-300">
                <span>{{ featuredPrompt.user.username }}</span>
                <span>{{ formatCount(featuredPrompt.likes) }} likes</span>
                <span>{{ formatCount(featuredPrompt.favorites) }} favorites</span>
              </div>
            </div>

            <div class="flex min-h-[280px] flex-col justify-between rounded-[24px] border border-white/10 bg-[linear-gradient(160deg,rgba(34,211,238,0.18),rgba(15,23,42,0.88))] p-5">
              <div class="text-xs uppercase tracking-[0.24em] text-cyan-100/80">
                {{ featuredPrompt.model }}
              </div>
              <div class="space-y-3">
                <div class="rounded-2xl bg-slate-950/40 p-4">
                  <div class="text-xs text-cyan-100/70">
                    System Prompt
                  </div>
                  <p class="mt-2 line-clamp-4 text-sm leading-6 text-white/90">
                    {{ featuredPrompt.systemPrompt }}
                  </p>
                </div>
                <div class="grid grid-cols-3 gap-3 text-center text-sm">
                  <div class="rounded-2xl bg-white/8 p-3">
                    <div class="text-lg font-semibold text-white">
                      {{ featuredPrompt.params.temperature ?? '-' }}
                    </div>
                    <div class="mt-1 text-xs text-neutral-300">
                      Temp
                    </div>
                  </div>
                  <div class="rounded-2xl bg-white/8 p-3">
                    <div class="text-lg font-semibold text-white">
                      {{ featuredPrompt.params.topP ?? '-' }}
                    </div>
                    <div class="mt-1 text-xs text-neutral-300">
                      Top P
                    </div>
                  </div>
                  <div class="rounded-2xl bg-white/8 p-3">
                    <div class="text-lg font-semibold text-white">
                      {{ featuredPrompt.params.maxTokens ?? '-' }}
                    </div>
                    <div class="mt-1 text-xs text-neutral-300">
                      Tokens
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="grid gap-4">
          <article
            v-for="prompt in spotlightPrompts"
            :key="prompt.id"
            class="rounded-card border border-white/10 bg-white/6 p-5 shadow-glass"
          >
            <div class="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-neutral-400">
              <span>{{ prompt.categoryName }}</span>
              <span>{{ prompt.model }}</span>
            </div>
            <h3 class="mt-3 text-xl font-semibold text-white">
              {{ prompt.title }}
            </h3>
            <p class="mt-3 text-sm leading-6 text-neutral-300">
              {{ prompt.description }}
            </p>
            <div class="mt-5 flex items-center justify-between text-sm text-neutral-400">
              <span>{{ prompt.user.username }}</span>
              <span>{{ formatCount(prompt.likes) }} likes</span>
            </div>
          </article>
        </div>
      </section>

      <section class="mb-8 grid gap-4 sm:grid-cols-3">
        <article
          v-for="stat in communityStats"
          :key="stat.label"
          class="rounded-3xl border border-white/10 bg-white/6 px-5 py-4"
        >
          <div class="text-sm text-neutral-400">
            {{ stat.label }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-white">
            {{ stat.value }}
          </div>
        </article>
      </section>

      <section class="mb-6 flex flex-wrap items-center gap-3">
        <button
          class="rounded-full px-4 py-2 text-sm transition"
          :class="
            activeCategoryId === 'all'
              ? 'bg-white text-slate-950'
              : 'border border-white/12 bg-white/5 text-neutral-200 hover:bg-white/10'
          "
          @click="activeCategoryId = 'all'"
        >
          全部分类
        </button>
        <button
          v-for="category in promptStore.categories"
          :key="category.id"
          class="rounded-full px-4 py-2 text-sm transition"
          :class="
            activeCategoryId === category.id
              ? 'bg-cyan-300 text-slate-950'
              : 'border border-white/12 bg-white/5 text-neutral-200 hover:bg-white/10'
          "
          @click="activeCategoryId = category.id"
        >
          {{ category.name }} · {{ category.count }}
        </button>
      </section>

      <section>
        <div class="mb-5 flex items-end justify-between gap-4">
          <div>
            <div class="text-sm text-neutral-400">
              今日社区流
            </div>
            <h2 class="mt-1 text-2xl font-semibold text-white">
              最新可用 Prompt
            </h2>
          </div>
          <div class="text-sm text-neutral-400">
            {{ visiblePrompts.length }} items
          </div>
        </div>

        <div
          v-if="promptStore.loading"
          class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
        >
          <div
            v-for="index in 6"
            :key="index"
            class="h-64 animate-pulse rounded-card border border-white/10 bg-white/5"
          />
        </div>

        <div
          v-else
          class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
        >
          <article
            v-for="prompt in visiblePrompts"
            :key="prompt.id"
            class="group flex h-full flex-col rounded-card border border-white/10 bg-white/6 p-5 transition hover:-translate-y-1 hover:border-cyan-300/30 hover:bg-white/8"
          >
            <div class="flex items-center justify-between text-xs uppercase tracking-[0.18em] text-neutral-400">
              <span>{{ prompt.categoryName }}</span>
              <span>{{ prompt.model }}</span>
            </div>

            <div class="mt-4 rounded-[24px] border border-white/8 bg-[linear-gradient(135deg,rgba(124,58,237,0.18),rgba(6,182,212,0.1),rgba(15,23,42,0.9))] p-4">
              <div class="text-xs text-neutral-300">
                Prompt Preview
              </div>
              <p class="mt-3 line-clamp-4 text-sm leading-6 text-white/90">
                {{ prompt.content }}
              </p>
            </div>

            <h3 class="mt-5 text-xl font-semibold text-white">
              {{ prompt.title }}
            </h3>
            <p class="mt-3 flex-1 text-sm leading-6 text-neutral-300">
              {{ prompt.description }}
            </p>

            <div class="mt-5 flex flex-wrap gap-2">
              <span
                v-for="tag in prompt.tags"
                :key="tag"
                class="rounded-full bg-white/8 px-3 py-1 text-xs text-neutral-200"
              >
                {{ tag }}
              </span>
            </div>

            <div class="mt-6 flex items-center justify-between border-t border-white/10 pt-4 text-sm text-neutral-400">
              <span>{{ prompt.user.username }}</span>
              <div class="flex gap-4">
                <span>{{ formatCount(prompt.views) }} views</span>
                <span>{{ formatCount(prompt.likes) }} likes</span>
              </div>
            </div>
          </article>
        </div>
      </section>
    </div>
  </div>
</template>
