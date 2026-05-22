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
  <div class="min-h-screen bg-[#f5f3ee] text-[#111111]">
    <div class="mx-auto max-w-[1160px] px-4 pb-16 pt-6 sm:px-6 lg:px-8">
      <header class="mb-8 flex flex-wrap items-center gap-3 text-sm text-[#777777]">
        <RouterLink
          to="/"
          class="rounded-full border border-black/10 bg-white px-4 py-2 transition hover:border-black/20 hover:text-black"
        >
          返回首页
        </RouterLink>
        <span>/</span>
        <span v-if="prompt">{{ prompt.categoryName }}</span>
        <span v-else>提示词详情</span>
        <span
          v-if="promptStore.usingMockData"
          class="rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-xs text-amber-800"
        >
          演示详情
        </span>
      </header>

      <section
        v-if="promptStore.detailLoading"
        class="grid gap-6 lg:grid-cols-[1.35fr_0.65fr]"
      >
        <div class="h-[520px] animate-pulse rounded-[28px] bg-black/6" />
        <div class="h-[520px] animate-pulse rounded-[28px] bg-black/6" />
      </section>

      <section
        v-else-if="prompt"
        class="grid gap-6 lg:grid-cols-[1.35fr_0.65fr]"
      >
        <div class="space-y-6">
          <section class="panel-card overflow-hidden">
            <div
              v-if="showCoverImage"
              class="border-b border-black/8"
            >
              <img
                :src="coverImage"
                :alt="prompt.title"
                class="max-h-[420px] w-full object-cover"
              >
            </div>

            <div class="grid gap-0 lg:grid-cols-[1.1fr_0.9fr]">
              <div class="min-h-[340px] bg-[#faf8f4] p-8">
                <div class="inline-flex rounded-full border border-black/10 bg-white px-3 py-1 text-xs text-[#444444]">
                  AI 效果预览
                </div>
                <h1 class="mt-5 max-w-2xl text-3xl font-semibold leading-tight text-black sm:text-4xl">
                  {{ prompt.title }}
                </h1>
                <p class="mt-4 max-w-xl text-base leading-7 text-[#555555]">
                  {{ prompt.description }}
                </p>

                <div class="mt-6 flex flex-wrap gap-2">
                  <span
                    v-for="tag in prompt.tags"
                    :key="tag"
                    class="rounded-full border border-black/10 bg-white px-3 py-1 text-xs text-[#444444]"
                  >
                    {{ tag }}
                  </span>
                </div>
              </div>

              <div class="flex flex-col justify-between gap-5 border-t border-black/8 bg-white p-8 lg:border-l lg:border-t-0">
                <div>
                  <div class="text-xs uppercase tracking-[0.2em] text-[#777777]">
                    创作者
                  </div>
                  <div class="mt-3 text-2xl font-semibold text-black">
                    {{ prompt.user.username }}
                  </div>
                  <p class="mt-3 text-sm leading-6 text-[#555555]">
                    {{ prompt.user.bio }}
                  </p>
                </div>

                <div class="flex flex-wrap gap-3">
                  <button
                    class="rounded-full bg-black px-4 py-2 text-sm font-medium text-white transition hover:bg-black/85 disabled:cursor-not-allowed disabled:opacity-70"
                    :disabled="liking"
                    @click="handleLike"
                  >
                    {{ liking ? '点赞中...' : `点赞 · ${prompt.likes.toLocaleString()}` }}
                  </button>
                  <button
                    class="rounded-full border border-black/10 bg-[#f6f4ef] px-4 py-2 text-sm font-medium text-[#333333] transition hover:border-black/20 hover:text-black disabled:cursor-not-allowed disabled:opacity-70"
                    :disabled="favoriting"
                    @click="handleFavorite"
                  >
                    {{ favoriting ? '收藏中...' : `收藏 · ${prompt.favorites.toLocaleString()}` }}
                  </button>
                </div>

                <div class="grid grid-cols-3 gap-3">
                  <div
                    v-for="stat in statCards"
                    :key="stat.label"
                    class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4 text-center"
                  >
                    <div class="text-lg font-semibold text-black">
                      {{ stat.value }}
                    </div>
                    <div class="mt-1 text-xs text-[#777777]">
                      {{ stat.label }}
                    </div>
                  </div>
                </div>

                <div class="rounded-[18px] border border-black/8 bg-[#111111] p-4 text-white">
                  <div class="text-xs uppercase tracking-[0.2em] text-white/60">
                    预期输出
                  </div>
                  <p class="mt-3 text-sm leading-6 text-white/75">
                    适合作为可直接落地的起点。上线前请替换为你的业务场景、品牌信息与约束条件。
                  </p>
                </div>
              </div>
            </div>
          </section>

          <section class="grid gap-6 xl:grid-cols-[1fr_1fr]">
            <article class="panel-card p-6">
              <div class="flex items-center justify-between gap-3">
                <div class="text-xs uppercase tracking-[0.2em] text-[#777777]">
                  提示词正文
                </div>
                <button
                  class="rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1.5 text-xs text-[#333333] transition hover:border-black/20 hover:text-black"
                  @click="copyText('提示词', prompt.content)"
                >
                  复制提示词
                </button>
              </div>
              <pre class="mt-4 whitespace-pre-wrap text-sm leading-7 text-[#444444]">{{ prompt.content }}</pre>
            </article>

            <article class="panel-card p-6">
              <div class="flex items-center justify-between gap-3">
                <div class="text-xs uppercase tracking-[0.2em] text-[#777777]">
                  系统提示词
                </div>
                <button
                  class="rounded-full border border-black/10 bg-[#f6f4ef] px-3 py-1.5 text-xs text-[#333333] transition hover:border-black/20 hover:text-black"
                  @click="copyText('系统提示词', prompt.systemPrompt)"
                >
                  复制系统提示词
                </button>
              </div>
              <pre class="mt-4 whitespace-pre-wrap text-sm leading-7 text-[#444444]">{{ prompt.systemPrompt }}</pre>
            </article>
          </section>

          <section
            v-if="relatedPrompts.length > 0"
            class="panel-card p-6"
          >
            <div class="flex items-end justify-between gap-4">
              <div>
                <div class="text-xs uppercase tracking-[0.2em] text-[#777777]">
                  相关推荐
                </div>
                <h2 class="mt-2 text-2xl font-semibold text-black">
                  同分类更多内容
                </h2>
              </div>
            </div>

            <div class="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              <RouterLink
                v-for="item in relatedPrompts"
                :key="item.id"
                :to="`/prompt/${item.id}`"
                class="rounded-[20px] border border-black/8 bg-[#faf8f4] p-5 transition hover:-translate-y-1 hover:border-black/20 hover:bg-white"
              >
                <div class="text-xs uppercase tracking-[0.18em] text-[#7c7c7c]">
                  {{ item.model }}
                </div>
                <div class="mt-3 text-lg font-semibold text-black">
                  {{ item.title }}
                </div>
                <p class="mt-3 text-sm leading-6 text-[#555555]">
                  {{ item.description }}
                </p>
              </RouterLink>
            </div>
          </section>
        </div>

        <aside class="space-y-6">
          <section class="panel-card p-6">
            <div class="text-xs uppercase tracking-[0.2em] text-[#777777]">
              提示词信息
            </div>
            <div class="mt-5 space-y-4">
              <div
                v-for="item in promptMeta"
                :key="item.label"
                class="flex items-start justify-between gap-4 border-b border-black/8 pb-4 last:border-b-0 last:pb-0"
              >
                <div class="text-sm text-[#777777]">
                  {{ item.label }}
                </div>
                <div class="text-right text-sm text-black">
                  {{ item.value }}
                </div>
              </div>
            </div>
          </section>

          <section class="panel-card p-6">
            <div class="text-xs uppercase tracking-[0.2em] text-[#777777]">
              参数
            </div>
            <div class="mt-5 grid grid-cols-3 gap-3">
              <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4 text-center">
                <div class="text-lg font-semibold text-black">
                  {{ prompt.params.temperature ?? '-' }}
                </div>
                <div class="mt-1 text-xs text-[#777777]">
                  温度
                </div>
              </div>
              <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4 text-center">
                <div class="text-lg font-semibold text-black">
                  {{ prompt.params.topP ?? '-' }}
                </div>
                <div class="mt-1 text-xs text-[#777777]">
                  Top P
                </div>
              </div>
              <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4 text-center">
                <div class="text-lg font-semibold text-black">
                  {{ prompt.params.maxTokens ?? '-' }}
                </div>
                <div class="mt-1 text-xs text-[#777777]">
                  最大 Token
                </div>
              </div>
            </div>
          </section>

          <section class="panel-card p-6">
            <div class="text-xs uppercase tracking-[0.2em] text-[#777777]">
              使用说明
            </div>
            <ul class="mt-5 space-y-3 text-sm leading-6 text-[#555555]">
              <li>保留整体结构，再替换为你的业务场景、目标用户与约束条件。</li>
              <li>若输出过于发散，可先降低温度，再补充更强示例。</li>
              <li>上线前请用真实生产输入至少跑一遍工作流回归验证。</li>
            </ul>
          </section>
        </aside>
      </section>

      <section
        v-else
        class="rounded-[28px] border border-dashed border-black/12 bg-white px-6 py-16 text-center"
      >
        <h1 class="text-2xl font-semibold text-black">
          未找到该提示词
        </h1>
        <p class="mt-3 text-sm text-[#777777]">
          内容可能已被删除，或链接已失效。
        </p>
        <RouterLink
          to="/"
          class="mt-6 inline-flex rounded-full border border-black/10 bg-[#f6f4ef] px-5 py-3 text-sm text-[#333333] transition hover:border-black/20 hover:text-black"
        >
          返回首页
        </RouterLink>
      </section>
    </div>
  </div>
</template>
