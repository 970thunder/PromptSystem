import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useNavigationHistory } from '@/navigation/navigationHistory'

export const useRouteTransition = () => {
  const route = useRoute()
  const router = useRouter()
  const { direction } = useNavigationHistory(router)

  return {
    name: computed(() => `route-${direction.value}`),
    // 以 path 为 key：查询串变化（筛选/翻页）不再整页卸载重挂载，
    // 视图内部已 watch route.fullPath 做响应式同步。
    key: computed(() => route.path)
  }
}
