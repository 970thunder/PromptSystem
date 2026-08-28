<script setup lang="ts">
import { reactive } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import { githubAuthUrl } from '@/utils/authUrl'
import { isSafeInternalPath } from '@/composables/useBackNavigation'
import AppShell from '@/components/layout/AppShell.vue'

const userStore = useUserStore()
const route = useRoute()
const router = useRouter()

const formValue = reactive({
  email: '',
  password: ''
})

const handleSubmit = async () => {
  await userStore.login(formValue)
  const redirect = isSafeInternalPath(route.query.redirect)
  await router.push(redirect)
}

const handleGitHubLogin = () => {
  window.location.href = githubAuthUrl()
}
</script>

<template>
  <AppShell>
    <div class="auth-page">
      <div class="auth-layout">
        <section class="auth-hero panel-card">
          <div>
            <div class="auth-hero__badge">
              安全访问
            </div>
            <h1 class="auth-hero__title">
              登录后即可发布、收藏并管理你的 AI 提示词库
            </h1>
            <p class="auth-hero__desc">
              登录后即可发布、收藏并管理你的 AI 提示词库。
            </p>
          </div>

          <div class="auth-hero__features">
            <div
              v-for="item in [{ title: 'bcrypt', desc: '密码哈希' }, { title: 'JWT', desc: '会话鉴权' }, { title: 'Guard', desc: '路由守卫' }]"
              :key="item.title"
              class="auth-feature"
            >
              <div class="auth-feature__title">
                {{ item.title }}
              </div>
              <div class="auth-feature__desc">
                {{ item.desc }}
              </div>
            </div>
          </div>
        </section>

        <section class="auth-form-wrap">
          <NCard class="auth-card panel-card">
            <div class="auth-card__header">
              <div class="text-muted-sm">
                欢迎回来
              </div>
              <h2 class="auth-card__title">
                登录
              </h2>
            </div>

            <NForm @submit.prevent="handleSubmit">
              <NFormItem label="邮箱">
                <NInput
                  v-model:value="formValue.email"
                  placeholder="you@example.com"
                  size="large"
                />
              </NFormItem>

              <NFormItem label="密码">
                <NInput
                  v-model:value="formValue.password"
                  type="password"
                  show-password-on="click"
                  placeholder="请输入密码"
                  size="large"
                />
              </NFormItem>

              <NButton
                attr-type="submit"
                type="primary"
                size="large"
                block
                :loading="userStore.loading"
              >
                登录
              </NButton>

              <NButton
                class="auth-card__github"
                size="large"
                block
                secondary
                @click="handleGitHubLogin"
              >
                使用 GitHub 继续
              </NButton>
            </NForm>

            <div class="auth-card__actions">
              <RouterLink
                to="/forgot-password"
                class="auth-card__link"
              >
                忘记密码？
              </RouterLink>
            </div>

            <div class="auth-card__footer">
              还没有账号？
              <RouterLink
                to="/register"
                class="auth-card__link"
              >
                立即注册
              </RouterLink>
            </div>
          </NCard>
        </section>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.auth-page {
  @apply view-page px-4 py-10 sm:px-6;
}

.auth-layout {
  @apply view-container--auth grid gap-6 lg:grid-cols-[1.05fr_0.95fr];
}

.auth-hero {
  @apply flex min-h-[680px] flex-col justify-between p-8;
}

.auth-hero__badge {
  @apply inline-flex rounded-full border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] px-3 py-1 text-xs text-[var(--prompt-text-muted)];
}

.auth-hero__title {
  @apply mt-5 max-w-xl text-4xl font-semibold leading-tight text-[var(--prompt-text)];
}

.auth-hero__desc {
  @apply mt-5 max-w-xl text-base leading-7 text-[var(--prompt-text-muted)];
}

.auth-hero__features {
  @apply grid gap-4 sm:grid-cols-3;
}

.auth-feature {
  @apply rounded-[18px] border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] p-4;
}

.auth-feature__title {
  @apply text-lg font-semibold text-[var(--prompt-text)];
}

.auth-feature__desc {
  @apply mt-2 text-sm text-[var(--prompt-text-faint)];
}

.auth-form-wrap {
  @apply flex items-center;
}

.auth-card {
  @apply w-full !rounded-[28px] !border-[var(--prompt-border)] !bg-[var(--prompt-surface)];
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.06) !important;
}

.auth-card__header {
  @apply mb-6;
}

.auth-card__title {
  @apply mt-2 text-2xl font-semibold text-[var(--prompt-text)];
}

.auth-card__github {
  @apply !mt-3;
}

.auth-card__hint {
  @apply mt-5;
}

.auth-card__actions {
  @apply mt-4 text-right text-sm;
}

.auth-card__footer {
  @apply mt-6 text-sm text-[var(--prompt-text-muted)];
}

.auth-card__link {
  @apply font-medium text-[var(--prompt-text)] underline-offset-2 transition hover:underline;
}
</style>
