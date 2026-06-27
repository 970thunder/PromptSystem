<script setup lang="ts">
import { computed } from 'vue'
import { NConfigProvider, NDialogProvider, NMessageProvider, NNotificationProvider } from 'naive-ui'
import GitHubBindReminder from '@/components/GitHubBindReminder.vue'

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
    textColor3: '#777777'
  }
}))
</script>

<template>
  <NConfigProvider :theme-overrides="themeOverrides">
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
