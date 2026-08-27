import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useNavigationHistory } from '@/navigation/navigationHistory'

export const useRouteTransition = () => {
  const route = useRoute()
  const router = useRouter()
  const { direction } = useNavigationHistory(router)

  return {
    name: computed(() => `route-${direction.value}`),
    key: computed(() => route.fullPath)
  }
}
