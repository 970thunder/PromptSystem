<script setup lang="ts">
import { h, onMounted, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useDialog } from 'naive-ui'
import { useUserStore } from '@/stores/user'
import { githubAuthUrl } from '@/utils/authUrl'

const userStore = useUserStore()
const dialog = useDialog()

const openBindGitHubPrompt = () => {
  if (!userStore.shouldPromptBindGitHub()) {
    return
  }

  userStore.markBindPromptShown()
  dialog.info({
    title: '绑定 GitHub',
    content: '当前账号尚未绑定 GitHub。绑定后可以直接使用 GitHub 登录，并减少账号找回和多端登录的摩擦。',
    positiveText: '去绑定',
    negativeText: '稍后再说',
    onPositiveClick: () => {
      window.location.href = githubAuthUrl()
    },
    action: () =>
      h(
        RouterLink,
        {
          to: '/profile'
        },
        {
          default: () => '查看账号'
        }
      )
  })
}

onMounted(async () => {
  if (!userStore.sessionReady) {
    await userStore.restoreSession()
    await userStore.fetchUserInfo()
  }

  openBindGitHubPrompt()
})

watch(
  () => userStore.userInfo,
  () => {
    openBindGitHubPrompt()
  }
)
</script>

<template>
  <slot />
</template>
