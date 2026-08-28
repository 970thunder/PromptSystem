<!-- 文件作用：社区页（与发现/搜索页差异化）。
     内容聚焦「工作流」「智能体」「社区参与」三个核心能力，不是单纯的列表筛选。 -->
<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { usePromptStore } from '@/stores/prompt'
import { useUserStore } from '@/stores/user'
import AppShell from '@/components/layout/AppShell.vue'
import PromptGrid from '@/components/prompt/PromptGrid.vue'
import PageLoading from '@/components/feedback/PageLoading.vue'
import PageError from '@/components/feedback/PageError.vue'
import { isDisplayableCover, resolveMediaUrl } from '@/utils/mediaUrl'

const promptStore = usePromptStore()
const userStore = useUserStore()
const router = useRouter()

// 社区页只展示与社区能力相关的内容；标签过滤由后端完成。
const workflows = computed(() =>
  promptStore.prompts.filter((prompt) => prompt.tags.some((tag) => tag.includes('流程') || tag.includes('工作流'))),
)
const agents = computed(() =>
  promptStore.prompts.filter((prompt) => prompt.tags.some((tag) => tag.includes('智能体') || tag.includes('agent'))),
)
const latest = computed(() => promptStore.prompts.slice(0, 6))

const goPublish = async () => {
  if (!userStore.isLoggedIn) {
    await router.push('/login?redirect=/publish')
    return
  }
  await router.push('/publish')
}

const coverUrl = (prompt: typeof latest.value[number] | undefined) => {
  if (!prompt) return ''
  return isDisplayableCover(prompt.cover) ? resolveMediaUrl(prompt.cover) : ''
}

onMounted(() => {
  void promptStore.loadHomeFeed()
})
</script>

<template>
  <AppShell>
    <div class="community-page">
      <div class="community-container">
        <!-- 大屏参与 CTA -->
        <section
          class="community-hero"
          aria-labelledby="community-hero-title"
        >
          <p class="community-hero__eyebrow">
            社区参与
          </p>
          <h1
            id="community-hero-title"
            class="community-hero__title"
          >
            发布、复用、一起把提示词打磨得更好。
          </h1>
          <p class="community-hero__desc">
            工作流和智能体是社区最常被复用的两类内容；写下你的方法，让别人站在你肩上继续前进。
          </p>
          <div class="community-hero__actions">
            <button
              type="button"
              class="community-hero__primary"
              @click="goPublish"
            >
              发布提示词
            </button>
            <RouterLink
              to="/search"
              class="community-hero__secondary"
            >
              去发现浏览全部
            </RouterLink>
          </div>
        </section>

        <hr
          class="community-divider"
          aria-hidden="true"
        >

        <!-- 工作流精选 -->
        <section
          class="community-section"
          aria-labelledby="community-workflows-title"
        >
          <header class="community-section__head">
            <div>
              <p class="section-eyebrow">
                工作流
              </p>
              <h2
                id="community-workflows-title"
                class="community-section__title"
              >
                多人协作的工作流
              </h2>
              <p class="community-section__desc">
                一组可拆解的多步提示词组合，演示如何把任务拆成可复用的章节。
              </p>
            </div>
            <RouterLink
              to="/search?tag=流程"
              class="community-section__more"
            >
              查看全部工作流
            </RouterLink>
          </header>

          <PageLoading
            v-if="promptStore.loading && workflows.length === 0 && agents.length === 0"
            label="正在加载社区精选"
            variant="blocks"
          />

          <PromptGrid
            v-else-if="workflows.length"
            :prompts="workflows.slice(0, 6)"
            :loading="promptStore.loading"
            :has-more="false"
            end-label="已展示工作流精选"
          />

          <PageError
            v-else
            kind="empty"
            title="还没有工作流类提示词"
            description="率先发布一个，定义社区的第一个工作流模板。"
            action-label="去发布"
            @action="goPublish"
          />
        </section>

        <hr
          class="community-divider"
          aria-hidden="true"
        >

        <!-- 智能体精选 -->
        <section
          class="community-section"
          aria-labelledby="community-agents-title"
        >
          <header class="community-section__head">
            <div>
              <p class="section-eyebrow">
                智能体
              </p>
              <h2
                id="community-agents-title"
                class="community-section__title"
              >
                角色化与指令化智能体
              </h2>
              <p class="community-section__desc">
                带身份、目标和约束的智能体提示词，适合长对话、客服或内容生成的场景。
              </p>
            </div>
            <RouterLink
              to="/search?tag=智能体"
              class="community-section__more"
            >
              查看全部智能体
            </RouterLink>
          </header>

          <PromptGrid
            v-if="agents.length"
            :prompts="agents.slice(0, 6)"
            :loading="promptStore.loading"
            :has-more="false"
            end-label="已展示智能体精选"
          />

          <PageError
            v-else
            kind="empty"
            title="还没有智能体类提示词"
            description="把你的角色化指令变成可被复用的智能体模板。"
            action-label="去发布"
            @action="goPublish"
          />
        </section>

        <hr
          class="community-divider"
          aria-hidden="true"
        >

        <!-- 社区最新动态 + 发布 CTA -->
        <section
          class="community-section"
          aria-labelledby="community-latest-title"
        >
          <header class="community-section__head">
            <div>
              <p class="section-eyebrow">
                社区动态
              </p>
              <h2
                id="community-latest-title"
                class="community-section__title"
              >
                最新提示词
              </h2>
              <p class="community-section__desc">
                大家最近发布了什么。先看一眼最近六条，去发现页查看更多。
              </p>
            </div>
            <RouterLink
              to="/search?sort=latest"
              class="community-section__more"
            >
              去发现全部
            </RouterLink>
          </header>

          <ul
            v-if="latest.length"
            class="community-list"
          >
            <li
              v-for="prompt in latest"
              :key="prompt.id"
              class="community-list__item"
            >
              <RouterLink
                :to="`/prompt/${prompt.id}`"
                class="community-list__link"
              >
                <span class="community-list__title">
                  {{ prompt.title }}
                </span>
                <span class="community-list__meta">
                  {{ prompt.user?.username ?? '作者' }}
                  <span aria-hidden="true">·</span>
                  {{ prompt.categoryName || '未分类' }}
                  <span aria-hidden="true">·</span>
                  {{ prompt.likes }} 点赞
                </span>
                <span
                  v-if="coverUrl(prompt)"
                  class="community-list__thumb"
                  :style="{ backgroundImage: `url(${coverUrl(prompt)})` }"
                />
              </RouterLink>
            </li>
          </ul>

          <PageError
            v-else
            kind="empty"
            title="社区还没有内容"
            description="发布第一条提示词，让这里热闹起来。"
            action-label="去发布"
            @action="goPublish"
          />
        </section>

        <!-- 大屏发布引导 -->
        <section
          class="community-publish"
          aria-label="发布提示词"
        >
          <div>
            <h2 class="community-publish__title">
              把你的方法沉淀为可复用模板
            </h2>
            <p class="community-publish__desc">
              结构化填写基本信息、内容、模型与封面，一次发布即可被社区搜索、收藏与引用。
            </p>
          </div>
          <button
            type="button"
            class="community-publish__cta"
            @click="goPublish"
          >
            开始发布 →
          </button>
        </section>

        <div
          v-if="promptStore.usingMockData"
          class="community-demo"
          role="status"
        >
          <span>当前为演示数据 / 离线预览，未连接实时服务。</span>
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.community-page {
  @apply view-page;
}

.community-container {
  @apply view-container--home;
}

/* 大屏参与 CTA */
.community-hero {
  @apply relative w-full py-16 sm:py-24;
}

.community-hero__eyebrow {
  @apply text-xs font-semibold uppercase tracking-[0.22em] text-[var(--prompt-text-faint)];
}

.community-hero__title {
  @apply mt-4 text-3xl font-semibold leading-tight sm:text-5xl text-[var(--prompt-text)];
}

.community-hero__desc {
  @apply mt-6 max-w-2xl text-base leading-7 text-[var(--prompt-text-muted)] sm:text-lg;
}

.community-hero__actions {
  @apply mt-10 flex flex-wrap items-center gap-4;
}

.community-hero__primary,
.community-hero__secondary {
  @apply inline-flex min-h-[44px] items-center justify-center rounded-full px-6 py-3 text-sm font-medium transition;
}

.community-hero__primary {
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
}

.community-hero__primary:hover {
  background-color: var(--prompt-primary-hover);
}

.community-hero__secondary {
  color: var(--prompt-text-muted);
  text-decoration: underline;
  text-underline-offset: 4px;
}

.community-hero__secondary:hover {
  color: var(--prompt-text);
}

/* 分割线 */
.community-divider {
  @apply my-12 border-0 border-t border-[var(--prompt-border)];
}

/* 通用分区 */
.community-section {
  @apply grid gap-6;
}

.community-section__head {
  @apply grid gap-2 sm:grid-cols-[1fr_auto] sm:items-end;
}

.community-section__title {
  @apply mt-1 text-xl font-semibold sm:text-2xl text-[var(--prompt-text)];
}

.community-section__desc {
  @apply max-w-2xl text-sm leading-6 text-[var(--prompt-text-muted)] sm:text-base;
}

.community-section__more {
  @apply shrink-0 self-end text-sm text-[var(--prompt-text-muted)] underline-offset-4 hover:text-[var(--prompt-text)] hover:underline;
}

/* 动态列表（无圆角包裹） */
.community-list {
  @apply grid gap-3;
}

.community-list__item {
  @apply border-b border-[var(--prompt-border)];
}

.community-list__link {
  @apply relative grid grid-cols-[1fr_auto] items-center gap-4 py-4 transition hover:bg-[var(--prompt-surface-muted)];
}

.community-list__title {
  @apply truncate text-base font-medium text-[var(--prompt-text)];
}

.community-list__meta {
  @apply col-span-1 mt-1 flex flex-wrap items-center gap-2 text-sm text-[var(--prompt-text-faint)];
}

.community-list__thumb {
  @apply block h-16 w-24 bg-cover bg-center;
}

/* 大屏发布引导 */
.community-publish {
  @apply mt-12 flex flex-col items-start gap-6 py-10 sm:flex-row sm:items-center sm:justify-between sm:py-14;
}

.community-publish__title {
  @apply text-2xl font-semibold sm:text-3xl text-[var(--prompt-text)];
}

.community-publish__desc {
  @apply mt-3 max-w-xl text-base leading-7 text-[var(--prompt-text-muted)];
}

.community-publish__cta {
  @apply inline-flex min-h-[48px] shrink-0 items-center rounded-full bg-[var(--prompt-primary)] px-7 py-3 text-sm font-semibold text-[var(--prompt-primary-contrast)] transition hover:bg-[var(--prompt-primary-hover)];
}

.community-demo {
  @apply mt-8 text-sm text-[var(--prompt-text-faint)];
}

@media (prefers-reduced-motion: reduce) {
  .community-hero__primary,
  .community-hero__secondary,
  .community-publish__cta,
  .community-section__more,
  .community-list__link {
    transition: none;
  }
}
</style>