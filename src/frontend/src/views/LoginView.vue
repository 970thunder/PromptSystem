<script setup lang="ts">
import { reactive } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import { githubAuthUrl } from '@/utils/authUrl'

const userStore = useUserStore()
const route = useRoute()
const router = useRouter()

const formValue = reactive({
  email: 'astra@example.com',
  password: 'PromptOS123!'
})

const handleSubmit = async () => {
  await userStore.login(formValue)
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
  await router.push(redirect)
}

const handleGitHubLogin = () => {
  window.location.href = githubAuthUrl()
}
</script>

<template>
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
            当前版本已支持密码哈希、JWT 鉴权与受保护路由。演示账号已预填，便于快速体验完整流程。
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

          <div class="auth-card__hint text-muted-sm">
            演示账号：`astra@example.com` / `PromptOS123!`
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
</template>

<style scoped>
.auth-page {
  @apply min-h-screen bg-[#f5f3ee] px-4 py-10 text-[#111111] sm:px-6;
}

.auth-layout {
  @apply mx-auto grid max-w-6xl gap-6 lg:grid-cols-[1.05fr_0.95fr];
}

.auth-hero {
  @apply flex min-h-[680px] flex-col justify-between p-8;
}

.auth-hero__badge {
  @apply inline-flex rounded-full border border-black/10 bg-[#faf8f4] px-3 py-1 text-xs text-[#444444];
}

.auth-hero__title {
  @apply mt-5 max-w-xl text-4xl font-semibold leading-tight text-black;
}

.auth-hero__desc {
  @apply mt-5 max-w-xl text-base leading-7 text-[#555555];
}

.auth-hero__features {
  @apply grid gap-4 sm:grid-cols-3;
}

.auth-feature {
  @apply rounded-[18px] border border-black/10 bg-[#faf8f4] p-4;
}

.auth-feature__title {
  @apply text-lg font-semibold text-black;
}

.auth-feature__desc {
  @apply mt-2 text-sm text-[#666666];
}

.auth-form-wrap {
  @apply flex items-center;
}

.auth-card {
  @apply w-full !rounded-[28px] !border-black/10 !bg-white !shadow-[0_16px_40px_rgba(15,23,42,0.06)];
}

.auth-card__header {
  @apply mb-6;
}

.auth-card__title {
  @apply mt-2 text-2xl font-semibold text-black;
}

.auth-card__github {
  @apply !mt-3;
}

.auth-card__hint {
  @apply mt-5;
}

.auth-card__footer {
  @apply mt-6 text-sm text-[#555555];
}

.auth-card__link {
  @apply font-medium text-black underline-offset-2 transition hover:underline;
}
</style>
