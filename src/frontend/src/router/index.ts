import { createRouter, createWebHistory, type RouteLocationNormalized, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useSiteStore } from '@/stores/site'
import { isSafeInternalPath } from '@/composables/useBackNavigation'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Home',
    component: () => import('@/views/HomeView.vue'),
    meta: { title: '首页' }
  },
  {
    path: '/prompt/:id',
    name: 'PromptDetail',
    component: () => import('@/views/PromptDetailView.vue'),
    meta: { title: '提示词详情' }
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginView.vue'),
    meta: { title: '登录' }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/RegisterView.vue'),
    meta: { title: '注册' }
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('@/views/ForgotPasswordView.vue'),
    meta: { title: '找回密码' }
  },
  {
    path: '/auth/callback',
    name: 'AuthCallback',
    component: () => import('@/views/AuthCallbackView.vue'),
    meta: { title: '登录中' }
  },
  {
    path: '/publish',
    name: 'Publish',
    component: () => import('@/views/PublishView.vue'),
    meta: { requiresAuth: true, title: '发布' }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: () => import('@/views/ProfileView.vue'),
    meta: { requiresAuth: true, title: '个人主页' }
  },
  {
    path: '/profile/:userId',
    name: 'PublicProfile',
    component: () => import('@/views/ProfileView.vue'),
    meta: { title: '个人主页' }
  },
  {
    path: '/search',
    name: 'Search',
    component: () => import('@/views/SearchView.vue'),
    meta: { title: '搜索' }
  },
  {
    path: '/community',
    name: 'Community',
    component: () => import('@/views/CommunityView.vue'),
    meta: { title: '社区' }
  },
  {
    path: '/terms',
    name: 'Terms',
    component: () => import('@/views/LegalView.vue'),
    props: { document: 'terms' },
    meta: { title: '服务条款' }
  },
  {
    path: '/privacy',
    name: 'Privacy',
    component: () => import('@/views/LegalView.vue'),
    props: { document: 'privacy' },
    meta: { title: '隐私政策' }
  },
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('@/views/SetupView.vue'),
    meta: { title: '系统初始化' }
  },
  {
    path: '/admin',
    name: 'Admin',
    component: () => import('@/views/AdminView.vue'),
    meta: { requiresAuth: true, title: '审核控制台' }
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: '页面不存在' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    }

    if (to.path === from.path && to.hash === from.hash) {
      return false
    }

    return { top: 0 }
  }
})

// 会话恢复只在“必须知道登录态”的路由上阻塞首屏：公开页面先渲染，
// 恢复完成后顶栏等响应式 UI 自动更新。promise 缓存避免并发导航重复探测。
let sessionRestorePromise: Promise<boolean> | null = null
const ensureSession = (userStore: ReturnType<typeof useUserStore>) => {
  if (!sessionRestorePromise) {
    sessionRestorePromise = userStore.restoreSession()
  }
  return sessionRestorePromise
}

router.beforeEach(async (to) => {
  const userStore = useUserStore()

  const isAuthPage = to.path === '/login' || to.path === '/register' || to.path === '/forgot-password'
  const needsSessionNow = Boolean(to.meta.requiresAuth) || isAuthPage
  if (needsSessionNow && !userStore.sessionReady) {
    await ensureSession(userStore)
  } else {
    void ensureSession(userStore)
  }

  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    const redirect = isSafeInternalPath(to.fullPath)
    return {
      path: '/login',
      query: redirect === '/' ? undefined : { redirect }
    }
  }

  // 管理员路由守卫：确认过会话且明确非管理员的账号直接回首页，避免加载
  // 控制台后收到 403 空态。后端仍逐请求校验角色，这里只是体验层拦截。
  if (to.path === '/admin' && userStore.userInfo?.isAdmin === false) {
    return { path: '/' }
  }

  if (isAuthPage && userStore.isLoggedIn) {
    return '/'
  }

  return true
})

const applyDocumentTitle = (to: RouteLocationNormalized) => {
  const siteStore = useSiteStore()
  const pageTitle = typeof to.meta.title === 'string' ? to.meta.title : ''
  const siteName = siteStore.siteName
  document.title = pageTitle ? `${pageTitle} · ${siteName}` : siteName
}

router.afterEach((to) => {
  applyDocumentTitle(to)
  window.dispatchEvent(new CustomEvent('promptos:navigation'))

  // 站点名称来自 /site/info：首次加载完成后用真实名称刷新一次标题。
  const siteStore = useSiteStore()
  if (!siteStore.loaded) {
    void siteStore.ensureLoaded().then(() => applyDocumentTitle(to))
  }
})

export default router
