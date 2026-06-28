<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useMessage, NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import { userApi } from '@/api/userApi'

const router = useRouter()
const message = useMessage()

const emailPattern = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

const formValue = reactive({
  email: '',
  captcha: '',
  password: '',
  confirmPassword: ''
})

const captchaLoading = ref(false)
const submitLoading = ref(false)
const captchaCountdown = ref(0)
let countdownTimer: number | undefined

const passwordMismatch = computed(
  () => formValue.confirmPassword.length > 0 && formValue.password !== formValue.confirmPassword
)

const canSendCaptcha = computed(
  () => emailPattern.test(formValue.email.trim()) && !captchaLoading.value && captchaCountdown.value === 0
)

const infoItems = [
  '通过邮箱验证码确认账号归属。',
  '新密码会重新使用 bcrypt 哈希写入。',
  '重置成功后请使用新密码登录。'
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
    if (response.data.devCode) {
      formValue.captcha = response.data.devCode
      message.success(`开发环境验证码已回填：${response.data.devCode}`)
    } else {
      message.success('验证码已发送，请查收邮箱')
    }
    startCaptchaCountdown(60)
  } finally {
    captchaLoading.value = false
  }
}

const handleSubmit = async () => {
  if (!emailPattern.test(formValue.email.trim())) {
    message.error('请输入有效的邮箱地址')
    return
  }
  if (formValue.captcha.trim().length !== 6) {
    message.error('请输入 6 位邮箱验证码')
    return
  }
  if (formValue.password.length < 8) {
    message.error('新密码至少需要 8 个字符')
    return
  }
  if (passwordMismatch.value) {
    message.error('两次输入的密码不一致')
    return
  }

  submitLoading.value = true
  try {
    await userApi.resetPassword({
      email: formValue.email.trim(),
      captcha: formValue.captcha.trim(),
      password: formValue.password
    })
    message.success('密码已重置，请使用新密码登录')
    await router.push('/login')
  } finally {
    submitLoading.value = false
  }
}

onBeforeUnmount(() => {
  if (countdownTimer) {
    window.clearInterval(countdownTimer)
  }
})
</script>

<template>
  <div class="auth-page">
    <div class="auth-layout auth-layout--equal">
      <section class="auth-hero panel-card">
        <div class="section-eyebrow">
          账号恢复
        </div>
        <h1 class="auth-hero__title auth-hero__title--compact">
          找回 PromptOS 密码
        </h1>
        <p class="auth-hero__desc">
          使用邮箱验证码重置密码。开发环境会直接回填验证码，便于本地验证完整账号恢复流程。
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
              重置密码
            </div>
            <h2 class="auth-card__title">
              验证邮箱
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

            <NFormItem label="新密码">
              <NInput
                v-model:value="formValue.password"
                type="password"
                show-password-on="click"
                placeholder="至少 8 个字符"
                size="large"
              />
            </NFormItem>

            <NFormItem
              label="确认新密码"
              :validation-status="passwordMismatch ? 'error' : undefined"
              :feedback="passwordMismatch ? '两次输入的密码不一致。' : ''"
            >
              <NInput
                v-model:value="formValue.confirmPassword"
                type="password"
                show-password-on="click"
                placeholder="再次输入新密码"
                size="large"
              />
            </NFormItem>

            <NButton
              attr-type="submit"
              type="primary"
              size="large"
              block
              :loading="submitLoading"
              :disabled="passwordMismatch"
            >
              重置密码
            </NButton>
          </NForm>

          <div class="auth-card__footer">
            想起密码了？
            <RouterLink
              to="/login"
              class="auth-card__link"
            >
              返回登录
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

.captcha-row {
  @apply grid w-full grid-cols-[1fr_auto] gap-3;
}

.captcha-row__button {
  @apply min-w-[118px];
}

.auth-card__footer {
  @apply mt-6 text-sm text-[#555555];
}

.auth-card__link {
  @apply font-medium text-black underline-offset-2 transition hover:underline;
}
</style>
