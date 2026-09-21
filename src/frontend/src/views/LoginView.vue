<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { oidcAuthUrl, oidcEnabled } from '@/utils/authUrl'
import { useIsoumaoLogin } from '@/composables/useIsoumaoLogin'
import { isSafeInternalPath } from '@/composables/useBackNavigation'
import { useUserStore } from '@/stores/user'
import AppShell from '@/components/layout/AppShell.vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { openLoginDialog } = useIsoumaoLogin()

const returnTo = computed(() => isSafeInternalPath(route.query.redirect))
const loginUrl = computed(() => `${oidcAuthUrl()}?returnTo=${encodeURIComponent(returnTo.value)}`)
const errorMessage = ref('')

// 统一账号只走弹窗：不再整页跳转。直接访问 /login 时自动弹出一次，失败可手动重试。
async function openLogin() {
  try {
    await openLoginDialog({
      loginUrl: loginUrl.value,
      description: '使用 isoumao 统一账号登录，登录后即可发布与收藏。',
      onSuccess: async () => {
        await userStore.restoreSession()
        await router.replace(returnTo.value)
      }
    })
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '登录组件加载失败'
  }
}

onMounted(async () => {
  await userStore.restoreSession()
  if (userStore.isLoggedIn) {
    await router.replace(returnTo.value)
    return
  }
  if (oidcEnabled) await openLogin()
})
</script>

<template>
  <AppShell>
    <main class="auth-redirect">
      <h1 class="auth-redirect__title">登录 isoumao</h1>
      <button v-if="oidcEnabled" class="auth-redirect__button" type="button" @click="openLogin">打开登录窗口</button>
      <p v-else class="auth-redirect__notice" role="status">登录服务暂未开放，请稍后再试。</p>
      <p v-if="errorMessage" class="auth-redirect__error" role="alert">{{ errorMessage }}</p>
    </main>
  </AppShell>
</template>

<style scoped>
.auth-redirect__title { font-size: 1.6rem; font-weight: 700; margin: 0; }.auth-redirect { display: grid; min-height: 55vh; place-items: center; padding: 2rem; gap: .75rem; }
.auth-redirect__button { border: 0; border-radius: 999px; padding: .75rem 1.4rem; color: white; background: var(--prompt-primary); font-weight: 600; cursor: pointer; }
.auth-redirect__notice { color: var(--prompt-text-muted, #6b7280); }
.auth-redirect__error { color: #dc2626; }
</style>