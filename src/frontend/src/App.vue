<!-- 文件作用：挂载全局 Provider，并统一配置 PromptOS 的 Naive UI 主题。 -->
<script setup lang="ts">
import { computed } from 'vue'
import { darkTheme, NConfigProvider, NDialogProvider, NMessageProvider, NNotificationProvider } from 'naive-ui'
import GitHubBindReminder from '@/components/GitHubBindReminder.vue'
import { useSiteTheme } from '@/composables/useSiteTheme'
import { useRouteTransition } from '@/composables/useRouteTransition'

const { resolvedMode } = useSiteTheme()
const { name: routeTransitionName, key: routeTransitionKey } = useRouteTransition()

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
