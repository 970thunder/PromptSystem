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
    message.error('Please enter a valid email address')
    return
  }

  if (formValue.password.length < 8) {
    message.error('Password must be at least 8 characters')
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
          Creator Onboarding
        </div>
        <h1 class="mt-4 text-4xl font-semibold leading-tight text-black">
          Create a new PromptOS account
        </h1>
        <p class="mt-5 max-w-xl text-base leading-7 text-[#555555]">
          Registration signs you in right away with a JWT-backed session. The verification code field is still a dev placeholder and can be replaced by real email verification later.
        </p>

        <div class="mt-8 space-y-4 text-sm text-[#555555]">
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            Passwords must be at least 8 characters and are stored with bcrypt hashing.
          </div>
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            The backend enforces unique email addresses to prevent duplicate accounts.
          </div>
          <div class="rounded-[18px] border border-black/8 bg-[#faf8f4] p-4">
            Protected pages automatically redirect to sign-in when you are not authenticated.
          </div>
        </div>
      </section>

      <section class="flex items-center">
        <NCard class="panel-card w-full !rounded-[28px] !border-black/8 !bg-white !shadow-[0_16px_40px_rgba(15,23,42,0.06)]">
          <div class="mb-6">
            <div class="text-sm text-[#777777]">
              New account
            </div>
            <h2 class="mt-2 text-2xl font-semibold text-black">
              Get started
            </h2>
          </div>

          <NForm @submit.prevent="handleSubmit">
            <NFormItem label="Username">
              <NInput
                v-model:value="formValue.username"
                placeholder="Enter your display name"
                size="large"
              />
            </NFormItem>

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
                placeholder="At least 8 characters"
                size="large"
              />
            </NFormItem>

            <NFormItem
              label="Confirm password"
              :validation-status="passwordMismatch ? 'error' : undefined"
              :feedback="passwordMismatch ? 'Passwords do not match.' : ''"
            >
              <NInput
                v-model:value="formValue.confirmPassword"
                type="password"
                show-password-on="click"
                placeholder="Enter the password again"
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
              Create account
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

          <div class="mt-6 text-sm text-[#555555]">
            Already have an account?
            <RouterLink
              to="/login"
              class="font-medium text-black underline-offset-2 transition hover:underline"
            >
              Sign in
            </RouterLink>
          </div>
        </NCard>
      </section>
    </div>
  </div>
</template>
