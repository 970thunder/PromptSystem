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
            Secure Access
          </div>
          <h1 class="mt-5 max-w-xl text-4xl font-semibold leading-tight text-black">
            Sign in to publish, save, and manage your AI prompt library
          </h1>
          <p class="mt-5 max-w-xl text-base leading-7 text-[#555555]">
            This build already uses password hashing, JWT auth, and protected routes. The demo account is prefilled so you can test the flow quickly.
          </p>
        </div>

        <div class="grid gap-4 sm:grid-cols-3">
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            <div class="text-lg font-semibold text-black">
              bcrypt
            </div>
            <div class="mt-2 text-sm text-[#666666]">
              Password hashing
            </div>
          </div>
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            <div class="text-lg font-semibold text-black">
              JWT
            </div>
            <div class="mt-2 text-sm text-[#666666]">
              Session auth
            </div>
          </div>
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            <div class="text-lg font-semibold text-black">
              Guard
            </div>
            <div class="mt-2 text-sm text-[#666666]">
              Protected routes
            </div>
          </div>
        </div>
      </section>

      <section class="flex items-center">
        <NCard class="panel-card w-full !rounded-[28px] !border-black/8 !bg-white !shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
          <div class="mb-6">
            <div class="text-sm text-[#777777]">
              Welcome back
            </div>
            <h2 class="mt-2 text-2xl font-semibold text-black">
              Sign in
            </h2>
          </div>

          <NForm @submit.prevent="handleSubmit">
            <NFormItem label="Email">
              <NInput
                v-model:value="formValue.email"
                placeholder="you@example.com"
                size="large"
              />
            </NFormItem>

            <NFormItem label="Password">
              <NInput
                v-model:value="formValue.password"
                type="password"
                show-password-on="click"
                placeholder="Enter your password"
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
              Sign in
            </NButton>

            <NButton
              class="!mt-3"
              size="large"
              block
              secondary
              @click="handleGitHubLogin"
            >
              Continue with GitHub
            </NButton>
          </NForm>

          <div class="mt-5 text-sm text-[#777777]">
            Demo account: `astra@example.com` / `PromptOS123!`
          </div>

          <div class="mt-6 text-sm text-[#555555]">
            Need an account?
            <RouterLink
              to="/register"
              class="font-medium text-black underline-offset-2 transition hover:underline"
            >
              Create one
            </RouterLink>
          </div>
        </NCard>
      </section>
    </div>
  </div>
</template>
