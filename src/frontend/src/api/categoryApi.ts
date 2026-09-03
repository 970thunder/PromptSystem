import request from '@/utils/request'
import type { ApiResponse, Category } from '@/types'

export const categoryApi = {
  getCategoryList(signal?: AbortSignal): Promise<ApiResponse<Category[]>> {
    return request.get('/categories', { signal })
  }
}
