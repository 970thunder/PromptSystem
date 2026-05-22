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
  <div class="min-h-screen bg-[#f5f3ee] px-4 py-10 text-[#111111] sm:px-6">
    <div class="mx-auto grid max-w-6xl gap-6 lg:grid-cols-[1fr_1fr]">
      <section class="panel-card p-8">
        <div class="text-sm uppercase tracking-[0.2em] text-[#7c7c7c]">
          创作者入驻
        </div>
        <h1 class="mt-4 text-4xl font-semibold leading-tight text-black">
          创建 PromptOS 账号
        </h1>
        <p class="mt-5 max-w-xl text-base leading-7 text-[#555555]">
          注册成功后将自动登录并建立 JWT 会话。验证码字段目前为开发占位，后续可替换为真实邮箱验证。
        </p>

        <div class="mt-8 space-y-4 text-sm text-[#555555]">
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            密码至少 8 位，并使用 bcrypt 哈希存储。
          </div>
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            后端会校验邮箱唯一性，避免重复注册。
          </div>
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            未登录访问受保护页面时，将自动跳转到登录页。
          </div>
        </div>
      </section>

      <section class="flex items-center">
        <NCard class="panel-card w-full !rounded-[28px] !border-black/8 !bg-white !shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
          <div class="mb-6">
            <div class="text-sm text-[#777777]">
              新账号
            </div>
            <h2 class="mt-2 text-2xl font-semibold text-black">
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
              class="!mt-3"
              size="large"
              block
              secondary
              @click="handleGitHubLogin"
            >
              使用 GitHub 继续
            </NButton>
          </NForm>

          <div class="mt-6 text-sm text-[#555555]">
            已有账号？
            <RouterLink
              to="/login"
              class="font-medium text-black underline-offset-2 transition hover:underline"
            >
              去登录
            </RouterLink>
          </div>
        </NCard>
      </section>
    </div>
  </div>
</template>
