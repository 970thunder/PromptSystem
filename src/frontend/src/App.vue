<script setup lang="ts">
import { NConfigProvider, NMessageProvider, NDialogProvider, NNotificationProvider } from 'naive-ui'
import { computed } from 'vue'

const themeOverrides = computed(() => ({
  common: {
    primaryColor: '#7C3AED',
    primaryColorHover: '#9333EA',
    primaryColorPressed: '#6D28D9',
    bodyColor: '#0B0F19',
    cardColor: '#111827',
    modalColor: '#111827',
    popoverColor: '#1F2937',
    inputColor: '#1F2937',
    actionColor: '#27272A',
    borderColor: '#3F3F46',
    dividerColor: '#3F3F46',
    textColorBase: '#FFFFFF',
    textColor1: '#FFFFFF',
    textColor2: '#A1A1AA',
    textColor3: '#71717A',
  }
}))
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
