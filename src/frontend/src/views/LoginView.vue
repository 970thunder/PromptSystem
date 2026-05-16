<script setup lang="ts">
import { reactive } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import { NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import { useUserStore } from '@/stores/user'

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
</script>

<template>
  <div class="min-h-screen bg-[radial-gradient(circle_at_top,#312E81_0%,#0B0F19_38%,#090B11_100%)] px-5 py-10 text-white">
    <div class="mx-auto grid max-w-6xl gap-6 lg:grid-cols-[1.05fr_0.95fr]">
      <section class="flex min-h-[680px] flex-col justify-between rounded-card border border-white/10 bg-[linear-gradient(160deg,rgba(34,211,238,0.12),rgba(124,58,237,0.22),rgba(15,23,42,0.94))] p-8 shadow-glass">
        <div>
          <div class="inline-flex rounded-full border border-cyan-300/30 bg-cyan-300/10 px-3 py-1 text-xs text-cyan-100">
            Secure Access
          </div>
          <h1 class="mt-5 max-w-xl text-4xl font-semibold leading-tight text-white">
            登录 PromptOS，继续发布、收藏和管理你的 AI 资产
          </h1>
          <p class="mt-5 max-w-xl text-base leading-7 text-neutral-200">
            当前版本已接入密码哈希、JWT 和受保护路由。开发环境内置了演示账号，方便你直接验证整条登录链路。
          </p>
        </div>

        <div class="grid gap-4 sm:grid-cols-3">
          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            <div class="text-lg font-semibold text-white">
              bcrypt
            </div>
            <div class="mt-2 text-sm text-neutral-300">
              密码哈希存储
            </div>
          </div>
          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            <div class="text-lg font-semibold text-white">
              JWT
            </div>
            <div class="mt-2 text-sm text-neutral-300">
              鉴权与会话
            </div>
          </div>
          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            <div class="text-lg font-semibold text-white">
              Guard
            </div>
            <div class="mt-2 text-sm text-neutral-300">
              前端受保护路由
            </div>
          </div>
        </div>
      </section>

      <section class="flex items-center">
        <NCard class="w-full !rounded-[20px] !border !border-white/10 !bg-white/5 !shadow-none">
          <div class="mb-6">
            <div class="text-sm text-neutral-400">
              欢迎回来
            </div>
            <h2 class="mt-2 text-2xl font-semibold text-white">
              登录账号
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
          </NForm>

          <div class="mt-5 text-sm text-neutral-400">
            演示账号：`astra@example.com` / `PromptOS123!`
          </div>

          <div class="mt-6 text-sm text-neutral-300">
            还没有账号？
            <RouterLink
              to="/register"
              class="text-cyan-300 transition hover:text-cyan-200"
            >
              立即注册
            </RouterLink>
          </div>
        </NCard>
      </section>
    </div>
  </div>
</template>
