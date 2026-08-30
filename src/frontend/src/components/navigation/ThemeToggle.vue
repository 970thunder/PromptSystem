<!-- 文件作用：主题切换按钮，触发 useSiteTheme 的圆环扩散切换。 -->
<script setup lang="ts">
import { computed } from 'vue'
import { Moon, Sun } from 'lucide-vue-next'
import { useSiteTheme } from '@/composables/useSiteTheme'

const { resolvedMode, switchTheme } = useSiteTheme()

const nextModeLabel = computed(() => resolvedMode.value === 'dark' ? '切换到浅色主题' : '切换到深色主题')

function toggle(event: MouseEvent) {
  const next = resolvedMode.value === 'dark' ? 'light' : 'dark'
  switchTheme(next, event)
}
</script>

<template>
  <button
    type="button"
    class="theme-toggle"
    :aria-label="nextModeLabel"
    :aria-pressed="resolvedMode === 'dark'"
    :title="nextModeLabel"
    @click="toggle"
  >
    <Sun
      v-if="resolvedMode === 'dark'"
      aria-hidden="true"
    />
    <Moon
      v-else
      aria-hidden="true"
    />
  </button>
</template>

<style scoped>
.theme-toggle {
  display: inline-flex;
  width: 2.5rem;
  height: 2.5rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--prompt-border);
  border-radius: 9999px;
  background: var(--prompt-surface-muted);
  color: var(--prompt-text-muted);
  transition: background-color var(--prompt-duration-fast) var(--prompt-ease-out),
    border-color var(--prompt-duration-fast) var(--prompt-ease-out),
    color var(--prompt-duration-fast) var(--prompt-ease-out),
    transform var(--prompt-duration-fast) var(--prompt-ease-out);
}

.theme-toggle:hover {
  border-color: var(--prompt-border-strong);
  background: var(--prompt-surface);
  color: var(--prompt-primary);
  transform: translateY(-1px);
}

.theme-toggle:active {
  transform: translateY(0) scale(0.94);
}

.theme-toggle svg {
  width: 1.05rem;
  height: 1.05rem;
}

@media (prefers-reduced-motion: reduce) {
  .theme-toggle {
    transition: none;
  }
}
</style>
