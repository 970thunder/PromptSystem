import request from '@/utils/request'
import type { ApiResponse, PageResponse, Prompt, PublishPromptRequest, Category, UploadImageResponse, PromptActionResponse } from '@/types'

export const promptApi = {
  // Get prompt list with pagination
  getPromptList(params: {
    page: number
    pageSize: number
    categoryId?: number
    sort?: string
    userId?: number
    keyword?: string
    model?: string
  }): Promise<ApiResponse<PageResponse<Prompt>>> {
    return request.get('/prompts', { params })
  },

  // Get prompt detail
  getPromptDetail(id: number): Promise<ApiResponse<Prompt>> {
    return request.get(`/prompts/${id}`)
  },

  // Publish a new prompt
  publishPrompt(data: PublishPromptRequest): Promise<ApiResponse<Prompt>> {
    return request.post('/prompts', data)
  },

  uploadCover(file: File): Promise<ApiResponse<UploadImageResponse>> {
    const formData = new FormData()
    formData.append('file', file)

    return request.post('/uploads/images', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
  },

  // Update prompt
  updatePrompt(id: number, data: PublishPromptRequest): Promise<ApiResponse<Prompt>> {
    return request.put(`/prompts/${id}`, data)
  },

  // Delete prompt
  deletePrompt(id: number): Promise<ApiResponse<null>> {
    return request.delete(`/prompts/${id}`)
  },

  // Like a prompt
  likePrompt(id: number): Promise<ApiResponse<PromptActionResponse>> {
    return request.post(`/prompts/${id}/like`)
  },

  // Favorite a prompt
  favoritePrompt(id: number): Promise<ApiResponse<PromptActionResponse>> {
    return request.post(`/prompts/${id}/favorite`)
  },

  // Get categories
  getCategories(): Promise<ApiResponse<Category[]>> {
    return request.get('/categories')
  },

  // Search prompts
  searchPrompts(params: {
    keyword?: string
    page: number
    pageSize: number
    categoryId?: number
    model?: string
    sort?: string
  }): Promise<ApiResponse<PageResponse<Prompt>>> {
    return request.get('/prompts/search', { params })
  }
}
