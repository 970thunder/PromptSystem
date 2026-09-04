import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useUserStore } from '@/stores/user'
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

router.beforeEach(async (to) => {
  const userStore = useUserStore()

  if (!userStore.sessionReady) {
    await userStore.restoreSession()
  }

  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    const redirect = isSafeInternalPath(to.fullPath)
    return {
      path: '/login',
      query: redirect === '/' ? undefined : { redirect }
    }
  }

  if ((to.path === '/login' || to.path === '/register' || to.path === '/forgot-password') && userStore.isLoggedIn) {
    return '/'
  }

  return true
})

router.afterEach((to) => {
  const pageTitle = typeof to.meta.title === 'string' ? to.meta.title : ''
  document.title = pageTitle ? `${pageTitle} · PromptOS` : 'PromptOS'
  window.dispatchEvent(new CustomEvent('promptos:navigation'))
})

export default router
