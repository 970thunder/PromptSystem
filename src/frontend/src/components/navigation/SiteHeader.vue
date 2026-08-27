<!-- 文件作用：全局站点导航（非回调页面统一渲染）。顶部结构为品牌 Logo、一级入口触发按钮
     （展开文档流内超级菜单）、搜索入口、发布入口、主题按钮与用户入口。 -->
<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import MegaMenu from './MegaMenu.vue'
import ThemeToggle from './ThemeToggle.vue'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const userStore = useUserStore()

const open = ref(false)
const triggerRef = ref<HTMLButtonElement | null>(null)

const toggleMenu = () => {
  open.value = !open.value
  if (open.value) {
    // 菜单展开后，下方内容会被真实推开；这里记录基准供测量（不输出日志）。
    document.documentElement.dataset.menuOpen = 'true'
  } else {
    document.documentElement.dataset.menuOpen = 'false'
  }
}

const closeMenu = () => {
  if (!open.value) {
    return
  }
  open.value = false
  document.documentElement.dataset.menuOpen = 'false'
}

const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && open.value) {
    closeMenu()
    triggerRef.value?.focus()
  }
}

// 当前路由对应的一级入口，用于高亮触发按钮（发现 / 工作台）。
const activeSection = computed<'discover' | 'workspace' | null>(() => {
  const path = route.path
  if (path.startsWith('/search') || path.startsWith('/prompt') || path === '/') {
    return 'discover'
  }
  if (path.startsWith('/profile')) {
    return 'workspace'
  }
  return null
})

const userInitial = computed(() => userStore.userInfo?.username?.slice(0, 1)?.toUpperCase() ?? 'U')

watch(() => route.fullPath, closeMenu)
window.addEventListener('promptos:navigation', closeMenu)

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('promptos:navigation', closeMenu)
})
</script>

<template>
  <header
    class="site-header"
    :class="{ 'site-header--menu-open': open }"
  >
    <div class="site-header__bar">
      <div class="site-header__left">
        <RouterLink
          to="/"
          class="site-header__logo"
          aria-label="PromptOS 首页"
        >
          PromptOS
        </RouterLink>
        <button
          ref="triggerRef"
          type="button"
          class="nav-trigger"
          :class="{ 'nav-trigger--active': activeSection }"
          :aria-expanded="open"
          aria-controls="mega-menu"
          aria-label="展开站点导航菜单"
          @click="toggleMenu"
        >
          <span>菜单</span>
          <span
            class="nav-trigger__caret"
            aria-hidden="true"
          >{{ open ? '▴' : '▾' }}</span>
        </button>
      </div>

      <nav
        class="site-header__quick"
        aria-label="快捷入口"
      >
        <RouterLink
          to="/search"
          class="header-link"
        >
          搜索
        </RouterLink>
        <RouterLink
          to="/publish"
          class="header-link header-link--primary"
        >
          发布
        </RouterLink>
      </nav>

      <div class="site-header__right">
        <ThemeToggle />
        <RouterLink
          v-if="userStore.isLoggedIn"
          :to="`/profile/${userStore.userInfo?.id}`"
          class="header-avatar"
          :aria-label="`进入 ${userStore.userInfo?.username ?? '个人'} 主页`"
        >
          {{ userInitial }}
        </RouterLink>
        <RouterLink
          v-else
          to="/login"
          class="header-link"
        >
          登录
        </RouterLink>
      </div>
    </div>

    <MegaMenu
      :open="open"
      :active-section="activeSection"
      @navigate="closeMenu"
    />
  </header>
</template>

<style scoped>
.site-header {
  position: sticky;
  top: 0;
  z-index: 30;
  background-color: var(--prompt-bg);
  border-bottom: 1px solid var(--prompt-border);
  transition: box-shadow var(--prompt-duration-base) var(--prompt-ease-out);
}

.site-header--menu-open {
  box-shadow: var(--prompt-shadow-1);
}

.site-header__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 1rem;
}

.site-header__left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.site-header__logo {
  font-size: 1.125rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--prompt-text);
}

.site-header__quick {
  display: none;
  align-items: center;
  gap: 0.5rem;
}

@media (min-width: 640px) {
  .site-header__quick {
    display: flex;
  }
}

.site-header__right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.nav-trigger {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border-radius: 9999px;
  border: 1px solid var(--prompt-border);
  background-color: var(--prompt-surface-muted);
  padding: 0.4rem 0.9rem;
  font-size: 0.875rem;
  color: var(--prompt-text-muted);
  transition: border-color var(--prompt-duration-fast) var(--prompt-ease-out),
    color var(--prompt-duration-fast) var(--prompt-ease-out),
    background-color var(--prompt-duration-fast) var(--prompt-ease-out);
}

.nav-trigger:hover {
  border-color: var(--prompt-border-strong);
  color: var(--prompt-text);
}

.nav-trigger--active {
  border-color: transparent;
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
}

.nav-trigger__caret {
  font-size: 0.7rem;
  line-height: 1;
}

.header-link {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  border: 1px solid var(--prompt-border);
  background-color: var(--prompt-surface-muted);
  padding: 0.4rem 0.9rem;
  font-size: 0.875rem;
  color: var(--prompt-text-muted);
  transition: border-color var(--prompt-duration-fast) var(--prompt-ease-out),
    color var(--prompt-duration-fast) var(--prompt-ease-out);
}

.header-link:hover {
  border-color: var(--prompt-border-strong);
  color: var(--prompt-text);
}

.header-link--primary {
  border-color: transparent;
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
}

.header-link--primary:hover {
  background-color: var(--prompt-primary-hover);
  color: var(--prompt-primary-contrast);
}

.header-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.25rem;
  height: 2.25rem;
  border-radius: 9999px;
  border: 1px solid var(--prompt-border);
  background-color: var(--prompt-primary);
  color: var(--prompt-primary-contrast);
  font-size: 0.875rem;
  font-weight: 600;
}

@media (min-width: 640px) {
  .site-header__bar {
    padding-left: 1.5rem;
    padding-right: 1.5rem;
  }
}

@media (min-width: 1024px) {
  .site-header__bar {
    padding-left: 2rem;
    padding-right: 2rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .site-header {
    transition: none;
  }
}
</style>
