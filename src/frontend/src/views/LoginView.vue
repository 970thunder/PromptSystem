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
  <div class="min-h-screen bg-[#f5f3ee] px-4 py-10 text-[#111111] sm:px-6">
    <div class="mx-auto grid max-w-6xl gap-6 lg:grid-cols-[1.05fr_0.95fr]">
      <section class="panel-card flex min-h-[680px] flex-col justify-between p-8">
        <div>
          <div class="inline-flex rounded-full border border-black/10 bg-[#faf8f4] px-3 py-1 text-xs text-[#444444]">
            安全访问
          </div>
          <h1 class="mt-5 max-w-xl text-4xl font-semibold leading-tight text-black">
            登录后即可发布、收藏并管理你的 AI 提示词库
          </h1>
          <p class="mt-5 max-w-xl text-base leading-7 text-[#555555]">
            当前版本已支持密码哈希、JWT 鉴权与受保护路由。演示账号已预填，便于快速体验完整流程。
          </p>
        </div>

        <div class="grid gap-4 sm:grid-cols-3">
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            <div class="text-lg font-semibold text-black">
              bcrypt
            </div>
            <div class="mt-2 text-sm text-[#666666]">
              密码哈希
            </div>
          </div>
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            <div class="text-lg font-semibold text-black">
              JWT
            </div>
            <div class="mt-2 text-sm text-[#666666]">
              会话鉴权
            </div>
          </div>
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            <div class="text-lg font-semibold text-black">
              Guard
            </div>
            <div class="mt-2 text-sm text-[#666666]">
              路由守卫
            </div>
          </div>
        </div>
      </section>

      <section class="flex items-center">
        <NCard class="panel-card w-full !rounded-[28px] !border-black/8 !bg-white !shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
          <div class="mb-6">
            <div class="text-sm text-[#777777]">
              欢迎回来
            </div>
            <h2 class="mt-2 text-2xl font-semibold text-black">
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
              class="!mt-3"
              size="large"
              block
              secondary
              @click="handleGitHubLogin"
            >
              使用 GitHub 继续
            </NButton>
          </NForm>

          <div class="mt-5 text-sm text-[#777777]">
            演示账号：`astra@example.com` / `PromptOS123!`
          </div>

          <div class="mt-6 text-sm text-[#555555]">
            还没有账号？
            <RouterLink
              to="/register"
              class="font-medium text-black underline-offset-2 transition hover:underline"
            >
              立即注册
            </RouterLink>
          </div>
        </NCard>
      </section>
    </div>
  </div>
</template>
