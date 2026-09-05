<script setup lang="ts">
import { computed } from 'vue'
import { FileText, ShieldCheck } from 'lucide-vue-next'
import AppShell from '@/components/layout/AppShell.vue'

const props = defineProps<{
  document: 'privacy' | 'terms'
}>()

type LegalSection = {
  id: string
  title: string
  paragraphs: string[]
  items?: string[]
}

const content = computed(() => {
  if (props.document === 'privacy') {
    return {
      eyebrow: '隐私与数据',
      title: '隐私政策',
      summary: '说明 PromptOS 如何处理账户、内容与服务运行所需的数据。',
      updatedAt: '2026 年 9 月 5 日',
      icon: ShieldCheck,
      sections: [
        {
          id: 'scope',
          title: '1. 适用范围',
          paragraphs: ['本政策适用于你使用 PromptOS 网站、账户、发布、互动与内容浏览服务时产生的信息。使用服务即表示你理解本政策中与服务运行直接相关的处理方式。']
        },
        {
          id: 'data',
          title: '2. 我们处理的信息',
          paragraphs: ['我们仅处理提供和保护服务所需的信息。公开内容页面不会展示你的登录邮箱或第三方登录标识。'],
          items: ['账户资料：展示名称、头像、简介与注册邮箱。', '内容与互动：你发布的提示词、图片、评论、点赞、收藏、关注与举报记录。', '运行数据：为保障安全、排查故障与防止滥用而生成的访问日志、设备与请求信息。']
        },
        {
          id: 'use',
          title: '3. 使用方式',
          paragraphs: ['这些信息用于创建和维护账户、展示社区内容、处理互动与举报、预防滥用、改善产品体验，以及在法律要求时履行义务。我们不会将你的邮箱作为公开资料展示给其他用户。']
        },
        {
          id: 'storage',
          title: '4. 存储与安全',
          paragraphs: ['账户凭据以安全摘要方式处理；会话使用受保护的浏览器 Cookie 或兼容的访问令牌。上传内容会经过格式、权限与容量校验。我们采取合理措施保护数据，但互联网传输无法保证绝对安全。']
        },
        {
          id: 'rights',
          title: '5. 你的选择',
          paragraphs: ['你可以在个人中心编辑公开资料、管理已发布内容与草稿，也可以请求导出本人可访问的数据或注销账户。注销后，账户直接标识与认证资料会被匿名化或移除；为维护社区记录与处理争议，部分内容可能按适用规则保留。']
        },
        {
          id: 'contact',
          title: '6. 联系我们',
          paragraphs: ['涉及隐私、数据导出、删除或安全问题，请通过产品反馈渠道联系 PromptOS 运营方，并附上与你账户相关的必要信息，以便我们核验请求。']
        }
      ] satisfies LegalSection[]
    }
  }

  return {
    eyebrow: '社区规则',
    title: '服务条款',
    summary: '使用 PromptOS 前，请了解内容发布、互动与社区治理的基本约定。',
    updatedAt: '2026 年 9 月 5 日',
    icon: FileText,
    sections: [
      {
        id: 'service',
        title: '1. 服务说明',
        paragraphs: ['PromptOS 是用于发现、发布和交流 AI 提示词与创作方法的社区服务。当前服务不提供在线模型执行、交易结算、打赏、提现或托管代码运行。']
      },
      {
        id: 'account',
        title: '2. 账户使用',
        paragraphs: ['请使用真实可接收邮件的地址注册，并妥善保管账户凭据。你应对通过自己账户发生的发布、评论和互动负责；发现异常访问时应尽快修改密码并联系运营方。']
      },
      {
        id: 'content',
        title: '3. 内容与授权',
        paragraphs: ['你保留对自己发布内容所拥有的权利，同时授予 PromptOS 为展示、分发、搜索、审核和维护服务所必需的非独占许可。发布前请确认你拥有图片、文字、链接和其他素材的使用权。'],
        items: ['不得发布违法、侵权、欺诈、恶意攻击、骚扰或侵犯他人隐私的内容。', '不得上传恶意文件、规避安全检查或滥用自动化接口。', '复制或复用社区内容时，应遵守作者标注的许可和适用法律。']
      },
      {
        id: 'moderation',
        title: '4. 社区治理',
        paragraphs: ['用户可以举报违规内容。为保护社区，PromptOS 可根据举报、审核结果和适用法律限制曝光、下架内容、限制互动或停用账户。处理记录会用于安全审计和争议核查。']
      },
      {
        id: 'availability',
        title: '5. 服务变更',
        paragraphs: ['我们会持续改进服务，并可能调整功能、访问方式或社区规则。涉及重大影响的变更会通过站内方式更新说明。服务可能因维护、安全事件或第三方基础设施故障暂时不可用。']
      },
      {
        id: 'contact',
        title: '6. 联系与更新',
        paragraphs: ['如你认为内容或账户处理影响了自身权益，请通过产品反馈渠道联系 PromptOS 运营方。条款更新后会在本页面标注最新更新时间。']
      }
    ] satisfies LegalSection[]
  }
})
</script>

<template>
  <AppShell>
    <section class="legal-page">
      <div class="legal-page__hero">
        <div
          class="legal-page__hero-icon"
          aria-hidden="true"
        >
          <component
            :is="content.icon"
            :size="25"
            :stroke-width="1.8"
          />
        </div>
        <p class="section-eyebrow">
          {{ content.eyebrow }}
        </p>
        <h1>{{ content.title }}</h1>
        <p class="legal-page__summary">
          {{ content.summary }}
        </p>
        <p class="legal-page__updated">
          最后更新：{{ content.updatedAt }}
        </p>
      </div>

      <div class="legal-page__layout">
        <nav
          class="legal-page__toc"
          aria-label="页面目录"
        >
          <a
            v-for="section in content.sections"
            :key="section.id"
            :href="`#${section.id}`"
          >
            {{ section.title }}
          </a>
        </nav>

        <article class="legal-page__article">
          <section
            v-for="section in content.sections"
            :id="section.id"
            :key="section.id"
          >
            <h2>{{ section.title }}</h2>
            <p
              v-for="paragraph in section.paragraphs"
              :key="paragraph"
            >
              {{ paragraph }}
            </p>
            <ul v-if="section.items">
              <li
                v-for="item in section.items"
                :key="item"
              >
                {{ item }}
              </li>
            </ul>
          </section>
        </article>
      </div>
    </section>
  </AppShell>
</template>

<style scoped>
.legal-page {
  @apply mx-auto w-full px-4 pb-16 pt-10 sm:px-6 sm:pt-14 lg:px-8 xl:max-w-[95%] xl:px-10 2xl:max-w-[96vw];
}

.legal-page__hero {
  @apply mx-auto max-w-3xl border-b pb-10 text-center;
  border-color: var(--prompt-border);
}

.legal-page__hero-icon {
  @apply mx-auto mb-5 flex h-12 w-12 items-center justify-center rounded-[var(--prompt-radius)] border;
  color: var(--prompt-primary);
  background: var(--prompt-surface-muted);
  border-color: var(--prompt-border);
}

.legal-page h1 {
  @apply mt-3 text-3xl font-semibold sm:text-4xl;
  color: var(--prompt-text);
}

.legal-page__summary {
  @apply mx-auto mt-4 max-w-xl text-base leading-7 sm:text-lg;
  color: var(--prompt-text-muted);
}

.legal-page__updated {
  @apply mt-5 text-sm;
  color: var(--prompt-text-faint);
}

.legal-page__layout {
  @apply mx-auto mt-10 grid max-w-5xl gap-10 lg:grid-cols-[12rem_minmax(0,1fr)];
}

.legal-page__toc {
  @apply hidden h-fit gap-1 lg:sticky lg:top-24 lg:flex lg:flex-col;
}

.legal-page__toc a {
  @apply rounded-md px-3 py-2 text-sm transition-colors;
  color: var(--prompt-text-muted);
}

.legal-page__toc a:hover {
  color: var(--prompt-primary);
  background: var(--prompt-surface-muted);
}

.legal-page__article {
  @apply max-w-3xl;
}

.legal-page__article section {
  @apply scroll-mt-24 border-b py-8 first:pt-0 last:border-b-0;
  border-color: var(--prompt-border);
}

.legal-page__article h2 {
  @apply text-xl font-semibold;
  color: var(--prompt-text);
}

.legal-page__article p,
.legal-page__article li {
  @apply mt-4 text-[15px] leading-7;
  color: var(--prompt-text-muted);
}

.legal-page__article ul {
  @apply mt-4 space-y-2 pl-5;
  list-style: disc;
}

@media (max-width: 640px) {
  .legal-page { @apply pt-8; }
  .legal-page__hero { @apply text-left; }
  .legal-page__hero-icon { @apply mx-0; }
  .legal-page__summary { @apply mx-0; }
}
</style>
