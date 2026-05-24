<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useMessage, NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import { githubAuthUrl } from '@/utils/authUrl'

const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

const emailPattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

const formValue = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  captcha: 'dev-bypass'
})

const passwordMismatch = computed(
  () => formValue.confirmPassword.length > 0 && formValue.password !== formValue.confirmPassword
)

const infoItems = [
  '密码至少 8 位，并使用 bcrypt 哈希存储。',
  '后端会校验邮箱唯一性，避免重复注册。',
  '未登录访问受保护页面时，将自动跳转到登录页。'
]

const handleSubmit = async () => {
  if (passwordMismatch.value) {
    return
  }

  if (!emailPattern.test(formValue.email.trim())) {
    message.error('请输入有效的邮箱地址')
    return
  }

  if (formValue.password.length < 8) {
    message.error('密码至少需要 8 个字符')
    return
  }

  await userStore.register({
    username: formValue.username,
    email: formValue.email,
    password: formValue.password,
    captcha: formValue.captcha
  })

  await router.push('/')
}

const handleGitHubLogin = () => {
  window.location.href = githubAuthUrl()
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-layout auth-layout--equal">
      <section class="auth-hero panel-card">
        <div class="section-eyebrow">
          创作者入驻
        </div>
        <h1 class="auth-hero__title auth-hero__title--compact">
          创建 PromptOS 账号
        </h1>
        <p class="auth-hero__desc">
          注册成功后将自动登录并建立 JWT 会话。验证码字段目前为开发占位，后续可替换为真实邮箱验证。
        </p>

        <div class="auth-info-list">
          <div
            v-for="item in infoItems"
            :key="item"
            class="auth-info-item"
          >
            {{ item }}
          </div>
        </div>
      </section>

      <section class="auth-form-wrap">
        <NCard class="auth-card panel-card">
          <div class="auth-card__header">
            <div class="text-muted-sm">
              新账号
            </div>
            <h2 class="auth-card__title">
              开始使用
            </h2>
          </div>

          <NForm @submit.prevent="handleSubmit">
            <NFormItem label="用户名">
              <NInput
                v-model:value="formValue.username"
                placeholder="输入你的展示名称"
                size="large"
              />
            </NFormItem>

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
                placeholder="至少 8 个字符"
                size="large"
              />
            </NFormItem>

            <NFormItem
              label="确认密码"
              :validation-status="passwordMismatch ? 'error' : undefined"
              :feedback="passwordMismatch ? '两次输入的密码不一致。' : ''"
            >
              <NInput
                v-model:value="formValue.confirmPassword"
                type="password"
                show-password-on="click"
                placeholder="再次输入密码"
                size="large"
              />
            </NFormItem>

            <NButton
              attr-type="submit"
              type="primary"
              size="large"
              block
              :loading="userStore.loading"
              :disabled="passwordMismatch"
            >
              创建账号
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

          <div class="auth-card__footer">
            已有账号？
            <RouterLink
              to="/login"
              class="auth-card__link"
            >
              去登录
            </RouterLink>
          </div>
        </NCard>
      </section>
    </div>
  </div>
</template>

<style scoped>
.auth-page {
  @apply view-page px-4 py-10 sm:px-6;
}

.auth-layout {
  @apply view-container--auth grid gap-6 lg:grid-cols-[1.05fr_0.95fr];
}

.auth-layout--equal {
  @apply lg:grid-cols-[1fr_1fr];
}

.auth-hero {
  @apply p-8;
}

.auth-hero__title {
  @apply mt-5 max-w-xl text-4xl font-semibold leading-tight text-black;
}

.auth-hero__title--compact {
  @apply mt-4;
}

.auth-hero__desc {
  @apply mt-5 max-w-xl text-base leading-7 text-[#555555];
}

.auth-info-list {
  @apply mt-8 space-y-4 text-sm text-[#555555];
}

.auth-info-item {
  @apply rounded-[18px] border border-black/10 bg-[#faf8f4] p-4;
}

.auth-form-wrap {
  @apply flex items-center;
}

.auth-card {
  @apply w-full !rounded-[28px] !border-black/10 !bg-white;
  box-shadow: 0 16px 40px rgba(15, 23, 42, 0.06) !important;
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

.auth-card__footer {
  @apply mt-6 text-sm text-[#555555];
}

.auth-card__link {
  @apply font-medium text-black underline-offset-2 transition hover:underline;
}
</style>
