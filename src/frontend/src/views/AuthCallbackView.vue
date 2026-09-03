<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { userApi } from '@/api/userApi'
import { useUserStore } from '@/stores/user'
import { isSafeInternalPath } from '@/composables/useBackNavigation'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

const exchanging = ref(true)

onMounted(async () => {
  const error = typeof route.query.error === 'string' ? route.query.error : ''
  if (error) {
    message.error(error)
    await router.replace('/login')
    return
  }

  const code = typeof route.query.code === 'string' ? route.query.code : ''
  if (!code) {
    message.error('缺少登录凭据')
    await router.replace('/login')
    return
  }

  try {
    const response = await userApi.exchangeGithubCode(code)
    userStore.setUserInfo(response.data.user)
    userStore.setToken(response.data.token || '')
    message.success('已通过 GitHub 登录')
    const redirect = isSafeInternalPath(route.query.redirect)
    await router.replace(redirect)
  } catch {
    message.error('登录凭据无效或已过期，请重新登录')
    await router.replace('/login')
  } finally {
    exchanging.value = false
  }
})
</script>

<template>
  <div class="callback-page">
    <p class="callback-page__text">
      {{ exchanging ? '正在完成登录...' : '' }}
    </p>
  </div>
</template>

<style scoped>
.callback-page {
  @apply view-page flex items-center justify-center;
}

.callback-page__text {
  @apply text-sm text-[var(--prompt-text-faint)];
}
</style>
