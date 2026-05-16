<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

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

  await userStore.register({
    username: formValue.username,
    email: formValue.email,
    password: formValue.password,
    captcha: formValue.captcha
  })

  await router.push('/')
}
</script>

<template>
  <div class="min-h-screen bg-[radial-gradient(circle_at_top,#312E81_0%,#0B0F19_38%,#090B11_100%)] px-5 py-10 text-white">
    <div class="mx-auto grid max-w-6xl gap-6 lg:grid-cols-[1fr_1fr]">
      <section class="rounded-card border border-white/10 bg-white/6 p-8 shadow-glass">
        <div class="text-sm uppercase tracking-[0.24em] text-cyan-200/80">
          Creator Onboarding
        </div>
        <h1 class="mt-4 text-4xl font-semibold leading-tight text-white">
          创建一个新的 PromptOS 账号
        </h1>
        <p class="mt-5 max-w-xl text-base leading-7 text-neutral-300">
          注册完成后会立即签发 JWT，并进入已登录状态。当前开发环境的验证码字段先作为占位，后面可以接邮箱验证码。
        </p>

        <div class="mt-8 space-y-4 text-sm text-neutral-300">
          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            密码至少 8 位，后端使用 bcrypt 哈希保存。
          </div>
          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            服务端会校验邮箱唯一性，避免重复注册。
          </div>
          <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
            受保护页面会在未登录时自动跳转到登录页。
          </div>
        </div>
      </section>

      <section class="flex items-center">
        <NCard class="w-full !rounded-[20px] !border !border-white/10 !bg-white/5 !shadow-none">
          <div class="mb-6">
            <div class="text-sm text-neutral-400">
              新账号注册
            </div>
            <h2 class="mt-2 text-2xl font-semibold text-white">
              立即开始
            </h2>
          </div>

          <NForm @submit.prevent="handleSubmit">
            <NFormItem label="用户名">
              <NInput
                v-model:value="formValue.username"
                placeholder="输入你的昵称"
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
                placeholder="至少 8 位"
                size="large"
              />
            </NFormItem>

            <NFormItem
              label="确认密码"
              :validation-status="passwordMismatch ? 'error' : undefined"
              :feedback="passwordMismatch ? '两次输入的密码不一致' : ''"
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
          </NForm>

          <div class="mt-6 text-sm text-neutral-300">
            已有账号？
            <RouterLink
              to="/login"
              class="text-cyan-300 transition hover:text-cyan-200"
            >
              去登录
            </RouterLink>
          </div>
        </NCard>
      </section>
    </div>
  </div>
</template>
