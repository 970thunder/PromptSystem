<!-- 文件作用：全局站点导航，含下推式超级菜单与主题切换。 -->
<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import MegaMenu from './MegaMenu.vue'
import ThemeToggle from './ThemeToggle.vue'

const open = ref(false)
const triggerRef = ref<HTMLButtonElement | null>(null)

function toggleMenu() {
  open.value = !open.value
}

function closeMenu() {
  open.value = false
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && open.value) {
    closeMenu()
    triggerRef.value?.focus()
  }
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <header
    class="sticky top-0 z-30 border-b"
    :style="{ borderColor: 'var(--prompt-border)', backgroundColor: 'var(--prompt-bg)' }"
  >
    <div class="flex items-center justify-between px-4 py-3 sm:px-6 lg:px-8">
      <div class="flex items-center gap-6">
        <RouterLink
          to="/"
          class="text-lg font-bold"
          :style="{ color: 'var(--prompt-text)' }"
        >
          PromptOS
        </RouterLink>
        <button
          ref="triggerRef"
          type="button"
          class="btn-pill btn-pill-secondary"
          :aria-expanded="open"
          aria-controls="mega-menu"
          @click="toggleMenu"
        >
          菜单
        </button>
      </div>
      <div class="flex items-center gap-2">
        <RouterLink
          to="/search"
          class="btn-pill btn-pill-secondary"
        >
          搜索
        </RouterLink>
        <RouterLink
          to="/publish"
          class="btn-pill btn-pill-primary"
        >
          发布
        </RouterLink>
        <ThemeToggle />
      </div>
    </div>
    <MegaMenu
      id="mega-menu"
      :open="open"
      @navigate="closeMenu"
    />
  </header>
</template>
