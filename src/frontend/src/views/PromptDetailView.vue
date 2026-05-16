<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { usePromptStore } from '@/stores/prompt'

const route = useRoute()
const promptStore = usePromptStore()

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
    { label: '最近更新', value: prompt.value.updatedAt }
  ]
})

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
  <div class="min-h-screen bg-[radial-gradient(circle_at_top,#312E81_0%,#0B0F19_38%,#090B11_100%)] text-white">
    <div class="mx-auto max-w-7xl px-5 pb-16 pt-6 sm:px-8 lg:px-10">
      <header class="mb-8 flex flex-wrap items-center gap-3 text-sm text-neutral-400">
        <RouterLink
          to="/"
          class="rounded-full border border-white/10 bg-white/5 px-4 py-2 transition hover:bg-white/10 hover:text-white"
        >
          返回首页
        </RouterLink>
        <span>/</span>
        <span v-if="prompt">{{ prompt.categoryName }}</span>
        <span v-else>Prompt 详情</span>
        <span
          v-if="promptStore.usingMockData"
          class="rounded-full border border-amber-300/30 bg-amber-300/10 px-3 py-1 text-xs text-amber-200"
        >
          Mock Detail
        </span>
      </header>

      <section
        v-if="promptStore.detailLoading"
        class="grid gap-6 lg:grid-cols-[1.35fr_0.65fr]"
      >
        <div class="h-[520px] animate-pulse rounded-card border border-white/10 bg-white/5" />
        <div class="h-[520px] animate-pulse rounded-card border border-white/10 bg-white/5" />
      </section>

      <section
        v-else-if="prompt"
        class="grid gap-6 lg:grid-cols-[1.35fr_0.65fr]"
      >
        <div class="space-y-6">
          <section class="overflow-hidden rounded-card border border-white/10 bg-white/6 shadow-glass">
            <div class="grid gap-0 lg:grid-cols-[1.1fr_0.9fr]">
              <div class="min-h-[340px] bg-[linear-gradient(145deg,rgba(34,211,238,0.18),rgba(124,58,237,0.24),rgba(15,23,42,0.92))] p-8">
                <div class="inline-flex rounded-full border border-cyan-300/30 bg-cyan-300/10 px-3 py-1 text-xs text-cyan-100">
                  AI Result Preview
                </div>
                <h1 class="mt-5 max-w-2xl text-3xl font-semibold leading-tight text-white sm:text-4xl">
                  {{ prompt.title }}
                </h1>
                <p class="mt-4 max-w-xl text-base leading-7 text-neutral-200">
                  {{ prompt.description }}
                </p>

                <div class="mt-6 flex flex-wrap gap-2">
                  <span
                    v-for="tag in prompt.tags"
                    :key="tag"
                    class="rounded-full border border-white/10 bg-white/8 px-3 py-1 text-xs text-neutral-200"
                  >
                    {{ tag }}
                  </span>
                </div>
              </div>

              <div class="flex flex-col justify-between gap-5 p-8">
                <div>
                  <div class="text-xs uppercase tracking-[0.24em] text-neutral-400">
                    创作者
                  </div>
                  <div class="mt-3 text-2xl font-semibold text-white">
                    {{ prompt.user.username }}
                  </div>
                  <p class="mt-3 text-sm leading-6 text-neutral-300">
                    {{ prompt.user.bio }}
                  </p>
                </div>

                <div class="grid grid-cols-3 gap-3">
                  <div
                    v-for="stat in statCards"
                    :key="stat.label"
                    class="rounded-2xl border border-white/10 bg-white/5 p-4 text-center"
                  >
                    <div class="text-lg font-semibold text-white">
                      {{ stat.value }}
                    </div>
                    <div class="mt-1 text-xs text-neutral-400">
                      {{ stat.label }}
                    </div>
                  </div>
                </div>

                <div class="rounded-2xl border border-white/10 bg-slate-950/30 p-4">
                  <div class="text-xs uppercase tracking-[0.24em] text-neutral-400">
                    输出预期
                  </div>
                  <p class="mt-3 text-sm leading-6 text-neutral-200">
                    适合直接复制到模型中使用，再按你的业务背景补充变量、品牌信息和边界约束。
                  </p>
                </div>
              </div>
            </div>
          </section>

          <section class="grid gap-6 xl:grid-cols-[1fr_1fr]">
            <article class="rounded-card border border-white/10 bg-white/6 p-6 shadow-glass">
              <div class="text-xs uppercase tracking-[0.24em] text-neutral-400">
                Prompt 正文
              </div>
              <pre class="mt-4 whitespace-pre-wrap text-sm leading-7 text-neutral-200">{{ prompt.content }}</pre>
            </article>

            <article class="rounded-card border border-white/10 bg-white/6 p-6 shadow-glass">
              <div class="text-xs uppercase tracking-[0.24em] text-neutral-400">
                System Prompt
              </div>
              <pre class="mt-4 whitespace-pre-wrap text-sm leading-7 text-neutral-200">{{ prompt.systemPrompt }}</pre>
            </article>
          </section>

          <section
            v-if="relatedPrompts.length > 0"
            class="rounded-card border border-white/10 bg-white/6 p-6 shadow-glass"
          >
            <div class="flex items-end justify-between gap-4">
              <div>
                <div class="text-xs uppercase tracking-[0.24em] text-neutral-400">
                  Related
                </div>
                <h2 class="mt-2 text-2xl font-semibold text-white">
                  同分类相关推荐
                </h2>
              </div>
            </div>

            <div class="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              <RouterLink
                v-for="item in relatedPrompts"
                :key="item.id"
                :to="`/prompt/${item.id}`"
                class="rounded-3xl border border-white/10 bg-white/5 p-5 transition hover:-translate-y-1 hover:border-cyan-300/30 hover:bg-white/8"
              >
                <div class="text-xs uppercase tracking-[0.18em] text-neutral-400">
                  {{ item.model }}
                </div>
                <div class="mt-3 text-lg font-semibold text-white">
                  {{ item.title }}
                </div>
                <p class="mt-3 text-sm leading-6 text-neutral-300">
                  {{ item.description }}
                </p>
              </RouterLink>
            </div>
          </section>
        </div>

        <aside class="space-y-6">
          <section class="rounded-card border border-white/10 bg-white/6 p-6 shadow-glass">
            <div class="text-xs uppercase tracking-[0.24em] text-neutral-400">
              Prompt 信息
            </div>
            <div class="mt-5 space-y-4">
              <div
                v-for="item in promptMeta"
                :key="item.label"
                class="flex items-start justify-between gap-4 border-b border-white/8 pb-4 last:border-b-0 last:pb-0"
              >
                <div class="text-sm text-neutral-400">
                  {{ item.label }}
                </div>
                <div class="text-right text-sm text-white">
                  {{ item.value }}
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-card border border-white/10 bg-white/6 p-6 shadow-glass">
            <div class="text-xs uppercase tracking-[0.24em] text-neutral-400">
              参数展示
            </div>
            <div class="mt-5 grid grid-cols-3 gap-3">
              <div class="rounded-2xl border border-white/10 bg-white/5 p-4 text-center">
                <div class="text-lg font-semibold text-white">
                  {{ prompt.params.temperature ?? '-' }}
                </div>
                <div class="mt-1 text-xs text-neutral-400">
                  Temp
                </div>
              </div>
              <div class="rounded-2xl border border-white/10 bg-white/5 p-4 text-center">
                <div class="text-lg font-semibold text-white">
                  {{ prompt.params.topP ?? '-' }}
                </div>
                <div class="mt-1 text-xs text-neutral-400">
                  Top P
                </div>
              </div>
              <div class="rounded-2xl border border-white/10 bg-white/5 p-4 text-center">
                <div class="text-lg font-semibold text-white">
                  {{ prompt.params.maxTokens ?? '-' }}
                </div>
                <div class="mt-1 text-xs text-neutral-400">
                  Tokens
                </div>
              </div>
            </div>
          </section>

          <section class="rounded-card border border-white/10 bg-white/6 p-6 shadow-glass">
            <div class="text-xs uppercase tracking-[0.24em] text-neutral-400">
              使用建议
            </div>
            <ul class="mt-5 space-y-3 text-sm leading-6 text-neutral-300">
              <li>先保留原始结构，只替换你的业务背景、目标用户和约束条件。</li>
              <li>如果输出太发散，优先降低 Temperature，再增加示例输入。</li>
              <li>上线前至少做一次真实业务语境回归，确认风格和风险边界。</li>
            </ul>
          </section>
        </aside>
      </section>

      <section
        v-else
        class="rounded-card border border-dashed border-white/15 bg-white/5 px-6 py-16 text-center"
      >
        <h1 class="text-2xl font-semibold text-white">
          没找到这个 Prompt
        </h1>
        <p class="mt-3 text-sm text-neutral-400">
          可能已经下线，或者链接地址不对。
        </p>
        <RouterLink
          to="/"
          class="mt-6 inline-flex rounded-full border border-white/10 bg-white/5 px-5 py-3 text-sm text-white transition hover:bg-white/10"
        >
          返回首页继续逛
        </RouterLink>
      </section>
    </div>
  </div>
</template>
