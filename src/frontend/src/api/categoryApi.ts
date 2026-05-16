import request from '@/utils/request'
import type { ApiResponse, Category } from '@/types'

export const categoryApi = {
  getCategoryList(): Promise<ApiResponse<Category[]>> {
    return request.get('/categories')
  }
}
