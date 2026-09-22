<!-- 文件作用：全局站点导航。顶级入口（发现 / 社区 / 我的）直接铺开在顶栏，
     不聚合进单一按钮；鼠标悬停任意一级入口即整页下推展开超级菜单（文档流内，
     非浮层），支持键盘展开/关闭。 -->
<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import MegaMenu from './MegaMenu.vue'
import ThemeToggle from './ThemeToggle.vue'
import { useUserStore } from '@/stores/user'

type NavEntry = 'home' | 'discover' | 'community' | 'workspace'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

// 当前悬停/展开的一级入口；null 表示菜单关闭。
const activeEntry = ref<NavEntry | null>(null)
const open = computed(() => activeEntry.value !== null)

const closeTimer = ref<number | null>(null)
const lastFocused = ref<NavEntry | null>(null)

const openMenu = (entry: NavEntry, focus = false) => {
  if (closeTimer.value !== null) {
    window.clearTimeout(closeTimer.value)
    closeTimer.value = null
  }
  activeEntry.value = entry
  if (focus) {
    lastFocused.value = entry
  }
  document.documentElement.dataset.menuOpen = 'true'
}

const scheduleClose = () => {
  if (closeTimer.value !== null) {
    return
  }
  closeTimer.value = window.setTimeout(() => {
    activeEntry.value = null
    closeTimer.value = null
    document.documentElement.dataset.menuOpen = 'false'
  }, 180)
}

const cancelClose = () => {
  if (closeTimer.value !== null) {
    window.clearTimeout(closeTimer.value)
    closeTimer.value = null
  }
}

const closeMenu = () => {
  cancelClose()
  activeEntry.value = null
  document.documentElement.dataset.menuOpen = 'false'
}

const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && open.value) {
    closeMenu()
    if (lastFocused.value) {
      const el = document.querySelector<HTMLElement>(`[data-nav-entry="${lastFocused.value}"]`)
      el?.focus()
    }
  }
}

// 当前路由对应的一级入口，用于高亮顶栏项与超菜单分组。
const currentEntry = computed<NavEntry | null>(() => {
  const path = route.path
  if (path === '/' || path.startsWith('/prompt')) {
    return null
  }
  if (path === '/community' || path === '/publish') {
    return 'community'
  }
  if (path.startsWith('/search')) {
    const tag = String(route.query.tag ?? '')
    if (tag === '流程' || tag === '智能体') {
      return 'community'
    }
    return 'discover'
  }
  if (path.startsWith('/profile')) {
    return 'workspace'
  }
  return null
})

const userInitial = computed(() => userStore.userInfo?.username?.slice(0, 1)?.toUpperCase() ?? 'U')

const handleLogout = async () => {
  await userStore.logoutServer()
  closeMenu()
  await router.push('/')
}

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
    @mouseleave="scheduleClose"
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

        <nav
          class="site-header__primary"
          aria-label="主导航"
        >
          <RouterLink
            to="/"
            data-nav-entry="home"
            class="site-nav"
            :class="{ 'site-nav--active': route.path === '/' || activeEntry === 'home' }"
            aria-haspopup="true"
            :aria-expanded="activeEntry === 'home'"
            @mouseenter="openMenu('home')"
            @keydown.enter.prevent="openMenu('home', true)"
            @keydown.space.prevent="openMenu('home', true)"
          >
            首页
          </RouterLink>
          <RouterLink
            to="/search"
            data-nav-entry="discover"
            class="site-nav"
            :class="{ 'site-nav--active': currentEntry === 'discover' || activeEntry === 'discover' }"
            aria-haspopup="true"
            :aria-expanded="activeEntry === 'discover'"
            @mouseenter="openMenu('discover')"
            @keydown.enter.prevent="openMenu('discover', true)"
            @keydown.space.prevent="openMenu('discover', true)"
          >
            发现
          </RouterLink>
          <RouterLink
            to="/community"
            data-nav-entry="community"
            class="site-nav"
            :class="{ 'site-nav--active': currentEntry === 'community' || activeEntry === 'community' }"
            aria-haspopup="true"
            :aria-expanded="activeEntry === 'community'"
            @mouseenter="openMenu('community')"
            @keydown.enter.prevent="openMenu('community', true)"
            @keydown.space.prevent="openMenu('community', true)"
          >
            社区
          </RouterLink>
          <RouterLink
            :to="userStore.isLoggedIn ? '/profile' : '/login?redirect=/profile'"
            data-nav-entry="workspace"
            class="site-nav"
            :class="{ 'site-nav--active': currentEntry === 'workspace' || activeEntry === 'workspace' }"
            aria-haspopup="true"
            :aria-expanded="activeEntry === 'workspace'"
            @mouseenter="openMenu('workspace')"
            @keydown.enter.prevent="openMenu('workspace', true)"
            @keydown.space.prevent="openMenu('workspace', true)"
          >
            我的
          </RouterLink>
        </nav>
      </div>

      <div class="site-header__spacer" />

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
        <button
          v-if="userStore.isLoggedIn"
          type="button"
          class="header-link"
          @click="handleLogout"
        >
          退出
        </button>
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
      :active-section="activeEntry"
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
  justify-content: flex-start;
  gap: 1rem;
  padding: 0.75rem 1rem;
}

.site-header__spacer {
  flex: 1 1 auto;
  min-width: 0;
}

.site-header__left {
  display: flex;
  align-items: center;
  gap: 1rem;
  min-width: 0;
}

.site-header__logo {
  font-size: 1.125rem;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--prompt-text);
  white-space: nowrap;
}

.site-header__primary {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

/* 小屏下收窄一级入口，保证三个入口铺开不溢出 */
@media (max-width: 639px) {
  .site-header__bar {
    flex-wrap: wrap;
    row-gap: 0.5rem;
  }

  .site-header__left {
    flex: 1 1 auto;
  }

  .site-nav {
    padding: 0.4rem 0.65rem;
    font-size: 0.875rem;
  }
}

.site-nav {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  border-radius: var(--prompt-radius-sm);
  padding: 0.45rem 0.9rem;
  font-size: 0.925rem;
  font-weight: 500;
  color: var(--prompt-text-muted);
  white-space: nowrap;
  transition: background-color var(--prompt-duration-fast) var(--prompt-ease-out),
    color var(--prompt-duration-fast) var(--prompt-ease-out);
}

.site-nav:hover {
  background-color: var(--prompt-surface-muted);
  color: var(--prompt-text);
}

.site-nav--active {
  background-color: var(--prompt-surface-muted);
  color: var(--prompt-text);
  font-weight: 600;
}

.site-nav--active::after {
  content: '';
  display: block;
  height: 2px;
  border-radius: 9999px;
  background-color: var(--prompt-primary);
  margin-top: 1px;
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
