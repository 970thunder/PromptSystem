<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useMessage, NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import { userApi } from '@/api/userApi'
import { githubAuthUrl, githubOAuthEnabled } from '@/utils/authUrl'
import AppShell from '@/components/layout/AppShell.vue'

const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

const emailPattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

const formValue = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  captcha: ''
})

const captchaLoading = ref(false)
const captchaCountdown = ref(0)
let countdownTimer: number | undefined

const passwordMismatch = computed(
  () => formValue.confirmPassword.length > 0 && formValue.password !== formValue.confirmPassword
)

const canSendCaptcha = computed(
  () => emailPattern.test(formValue.email.trim()) && !captchaLoading.value && captchaCountdown.value === 0
)

const infoItems = [
  '密码至少 8 位，并使用安全哈希存储。',
  '注册需要邮箱验证码，请查收邮件后填写。',
  '登录后即可发布、收藏和管理你的提示词。'
]

const startCaptchaCountdown = (seconds: number) => {
  captchaCountdown.value = Math.max(1, Math.min(seconds, 60))
  if (countdownTimer) {
    window.clearInterval(countdownTimer)
  }
  countdownTimer = window.setInterval(() => {
    captchaCountdown.value -= 1
    if (captchaCountdown.value <= 0 && countdownTimer) {
      window.clearInterval(countdownTimer)
      countdownTimer = undefined
    }
  }, 1000)
}

const handleSendCaptcha = async () => {
  if (!emailPattern.test(formValue.email.trim())) {
    message.error('请先输入有效的邮箱地址')
    return
  }

  captchaLoading.value = true
  try {
    const response = await userApi.sendCaptcha(formValue.email.trim())
    if (import.meta.env.DEV && response.data.devCode) {
      formValue.captcha = response.data.devCode
      message.success(`本地开发验证码已自动填入：${response.data.devCode}`)
    } else {
      message.success('验证码已发送，请查收邮箱')
    }
    startCaptchaCountdown(60)
  } catch {
    message.error('验证码发送失败，请稍后重试')
  } finally {
    captchaLoading.value = false
  }
}

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

  if (formValue.captcha.trim().length !== 6) {
    message.error('请输入 6 位邮箱验证码')
    return
  }

  try {
    await userStore.register({
      username: formValue.username.trim(),
      email: formValue.email.trim(),
      password: formValue.password,
      captcha: formValue.captcha.trim()
    })
    await router.push('/')
  } catch {
    message.error('注册失败，请检查验证码或稍后重试')
  }
}

const handleGitHubLogin = () => {
  window.location.href = githubAuthUrl()
}

onBeforeUnmount(() => {
  if (countdownTimer) {
    window.clearInterval(countdownTimer)
  }
})
</script>

<template>
  <AppShell>
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
            注册成功后将自动登录。请使用真实邮箱接收验证码，未配置邮件服务时请联系站点管理员。
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

              <NFormItem label="邮箱验证码">
                <div class="captcha-row">
                  <NInput
                    v-model:value="formValue.captcha"
                    placeholder="输入 6 位验证码"
                    size="large"
                    maxlength="6"
                  />
                  <NButton
                    class="captcha-row__button"
                    size="large"
                    secondary
                    :loading="captchaLoading"
                    :disabled="!canSendCaptcha"
                    @click="handleSendCaptcha"
                  >
                    {{ captchaCountdown > 0 ? `${captchaCountdown}s` : '获取验证码' }}
                  </NButton>
                </div>
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
                v-if="githubOAuthEnabled"
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
  </AppShell>
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
  @apply mt-5 max-w-xl text-4xl font-semibold leading-tight text-[var(--prompt-text)];
}

.auth-hero__title--compact {
  @apply mt-4;
}

.auth-hero__desc {
  @apply mt-5 max-w-xl text-base leading-7 text-[var(--prompt-text-muted)];
}

.auth-info-list {
  @apply mt-8 space-y-4 text-sm text-[var(--prompt-text-muted)];
}

.auth-info-item {
  @apply rounded-[18px] border border-[var(--prompt-border)] bg-[var(--prompt-surface-muted)] p-4;
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

.captcha-row {
  @apply grid w-full grid-cols-[1fr_auto] gap-3;
}

.captcha-row__button {
  @apply min-w-[118px];
}

.auth-card__footer {
  @apply mt-6 text-sm text-[var(--prompt-text-muted)];
}

.auth-card__link {
  @apply font-medium text-[var(--prompt-text)] underline-offset-2 transition hover:underline;
}
</style>
