<script setup lang="ts">
import { NConfigProvider, NMessageProvider, NDialogProvider, NNotificationProvider, useDialog } from 'naive-ui'
import { computed, h, onMounted, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { githubAuthUrl } from '@/utils/authUrl'

const themeOverrides = computed(() => ({
  common: {
    primaryColor: '#111111',
    primaryColorHover: '#333333',
    primaryColorPressed: '#000000',
    bodyColor: '#f5f3ee',
    cardColor: '#ffffff',
    modalColor: '#ffffff',
    popoverColor: '#ffffff',
    inputColor: '#faf8f4',
    actionColor: '#f6f4ef',
    borderColor: 'rgba(0,0,0,0.1)',
    dividerColor: 'rgba(0,0,0,0.08)',
    textColorBase: '#111111',
    textColor1: '#111111',
    textColor2: '#555555',
    textColor3: '#777777',
  }
}))

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
  if (userStore.token && !userStore.userInfo) {
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
  <NConfigProvider :theme-overrides="themeOverrides">
    <NMessageProvider>
      <NDialogProvider>
        <NNotificationProvider>
          <router-view />
        </NNotificationProvider>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style>
#app {
  min-height: 100vh;
}
</style>
