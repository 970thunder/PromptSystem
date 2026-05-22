<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const userStore = useUserStore()

onMounted(async () => {
  const error = typeof route.query.error === 'string' ? route.query.error : ''
  if (error) {
    message.error(error)
    await router.replace('/login')
    return
  }

  const token = typeof route.query.token === 'string' ? route.query.token : ''
  if (!token) {
    message.error('缺少登录令牌')
    await router.replace('/login')
    return
  }

  userStore.setToken(token)
  await userStore.fetchUserInfo()
  message.success('已通过 GitHub 登录')
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
  await router.replace(redirect)
})
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-[#f5f3ee] text-[#111111]">
    <p class="text-sm text-[#666666]">
      正在完成登录...
    </p>
  </div>
</template>
