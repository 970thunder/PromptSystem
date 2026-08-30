<!-- 文件作用：挂载全局 Provider，并统一配置 PromptOS 的 Naive UI 主题。
     注意：naive-ui 的 themeOverrides 期望真实颜色字符串（内部用 rgba() 解析），
     因此不能用 CSS var()，必须按 light/dark 两套分别提供具体值。 -->
<script setup lang="ts">
import { computed } from 'vue'
import { darkTheme, NConfigProvider, NDialogProvider, NMessageProvider, NNotificationProvider } from 'naive-ui'
import GitHubBindReminder from '@/components/GitHubBindReminder.vue'
import { useSiteTheme } from '@/composables/useSiteTheme'
import { useRouteTransition } from '@/composables/useRouteTransition'

const { resolvedMode } = useSiteTheme()
const { name: routeTransitionName, key: routeTransitionKey } = useRouteTransition()

// naive-ui 需要真实颜色字符串（rgba / hex），不能用 var(--prompt-*)。
// 用 computed 直接产出两套具体值，跟随主题切换。
const themeOverrides = computed(() => {
  if (resolvedMode.value === 'dark') {
    return {
      common: {
        primaryColor: '#f5f5f5',
        primaryColorHover: '#ffffff',
        primaryColorPressed: '#e5e5e5',
        primaryColorSuppl: '#ffffff',
        bodyColor: '#121212',
        cardColor: '#1e1e1e',
        modalColor: '#1e1e1e',
        popoverColor: '#262626',
        inputColor: '#262626',
        actionColor: '#262626',
        tableHeaderColor: '#1e1e1e',
        borderColor: 'rgba(255, 255, 255, 0.12)',
        dividerColor: 'rgba(255, 255, 255, 0.12)',
        borderRadius: '16px',
        borderRadiusSmall: '12px',
        boxShadow1: '0 16px 40px rgba(0, 0, 0, 0.4)',
        boxShadow2: '0 20px 48px rgba(0, 0, 0, 0.5)',
        textColorBase: '#f5f5f5',
        textColor1: '#f5f5f5',
        textColor2: '#b3b3b3',
        textColor3: '#8a8a8a',
        successColor: '#34d399',
        warningColor: '#fbbf24',
        errorColor: '#f87171',
        infoColor: '#60a5fa'
      }
    }
  }
  return {
    common: {
      primaryColor: '#2563eb',
      primaryColorHover: '#1d4ed8',
      primaryColorPressed: '#1e40af',
      primaryColorSuppl: '#2563eb',
      bodyColor: '#f6f9ff',
      cardColor: '#ffffff',
      modalColor: '#ffffff',
      popoverColor: '#eef5ff',
      inputColor: '#eef5ff',
      actionColor: '#eef5ff',
      tableHeaderColor: '#ffffff',
      borderColor: 'rgba(37, 99, 235, 0.14)',
      dividerColor: 'rgba(37, 99, 235, 0.14)',
      borderRadius: '16px',
      borderRadiusSmall: '12px',
      boxShadow1: '0 16px 40px rgba(15, 23, 42, 0.06)',
      boxShadow2: '0 20px 48px rgba(15, 23, 42, 0.10)',
      textColorBase: '#0f172a',
      textColor1: '#0f172a',
      textColor2: '#475569',
      textColor3: '#64748b',
      successColor: '#22a06b',
      warningColor: '#b7791f',
      errorColor: '#c0392b',
      infoColor: '#2563eb'
    }
  }
})

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
            <RouterView v-slot="{ Component }">
              <Transition
                :name="routeTransitionName"
                mode="out-in"
              >
                <component
                  :is="Component"
                  :key="routeTransitionKey"
                />
              </Transition>
            </RouterView>
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

.route-forward-enter-active,
.route-back-enter-active {
  transition: opacity 420ms var(--prompt-ease-out),
    transform 420ms var(--prompt-ease-out);
}

.route-forward-leave-active,
.route-back-leave-active {
  transition: opacity 220ms var(--prompt-ease-in),
    transform 220ms var(--prompt-ease-in);
}

.route-forward-enter-from {
  opacity: 0;
  transform: translate3d(34px, 0, 0) rotate(0.6deg);
}

.route-forward-leave-to {
  opacity: 0;
  transform: translate3d(-26px, 0, 0) rotate(-0.4deg);
}

.route-back-enter-from {
  opacity: 0;
  transform: translate3d(-34px, 0, 0) rotate(-0.6deg);
}

.route-back-leave-to {
  opacity: 0;
  transform: translate3d(26px, 0, 0) rotate(0.4deg);
}

.route-replace-enter-active,
.route-replace-leave-active {
  transition: opacity var(--prompt-duration-fast) var(--prompt-ease-out),
    transform var(--prompt-duration-fast) var(--prompt-ease-out);
}

.route-replace-enter-from,
.route-replace-leave-to {
  opacity: 0;
  transform: translateY(6px);
}

@media (prefers-reduced-motion: reduce) {
  .route-forward-enter-active,
  .route-forward-leave-active,
  .route-back-enter-active,
  .route-back-leave-active,
  .route-replace-enter-active,
  .route-replace-leave-active {
    transition-duration: 1ms;
    transform: none;
  }
}
</style>
