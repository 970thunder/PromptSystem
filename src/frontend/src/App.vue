<!-- 文件作用：挂载全局 Provider，并统一配置 PromptOS 的 Naive UI 主题。 -->
<script setup lang="ts">
import { computed } from 'vue'
import { darkTheme, NConfigProvider, NDialogProvider, NMessageProvider, NNotificationProvider } from 'naive-ui'
import GitHubBindReminder from '@/components/GitHubBindReminder.vue'
import { useSiteTheme } from '@/composables/useSiteTheme'

const { resolvedMode } = useSiteTheme()

const themeOverrides = computed(() => ({
  common: {
    primaryColor: 'var(--prompt-primary)',
    primaryColorHover: 'var(--prompt-primary-hover)',
    bodyColor: 'var(--prompt-bg)',
    cardColor: 'var(--prompt-surface)',
    modalColor: 'var(--prompt-surface)',
    popoverColor: 'var(--prompt-surface)',
    inputColor: 'var(--prompt-surface-muted)',
    actionColor: 'var(--prompt-surface-muted)',
    borderColor: 'var(--prompt-border)',
    dividerColor: 'var(--prompt-border)',
    borderRadius: '16px',
    borderRadiusSmall: '12px',
    boxShadow1: 'var(--prompt-shadow-1)',
    boxShadow2: 'var(--prompt-shadow-2)',
    textColorBase: 'var(--prompt-text)',
    textColor1: 'var(--prompt-text)',
    textColor2: 'var(--prompt-text-muted)',
    textColor3: 'var(--prompt-text-faint)',
    successColor: 'var(--prompt-success)',
    warningColor: 'var(--prompt-warning)',
    errorColor: 'var(--prompt-error)',
    infoColor: 'var(--prompt-focus)'
  }
}))

const activeTheme = computed(() => (resolvedMode.value === 'dark' ? darkTheme : null))
</script>

<template>
  <NConfigProvider
    :theme="activeTheme"
    :theme-overrides="themeOverrides"
  >
    <NMessageProvider>
      <NDialogProvider>
        <NNotificationProvider>
          <GitHubBindReminder>
            <router-view />
          </GitHubBindReminder>
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
