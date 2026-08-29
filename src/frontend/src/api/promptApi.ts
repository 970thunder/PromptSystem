import request from '@/utils/request'
import type {
  ApiResponse,
  PageResponse,
  Prompt,
  PublishPromptRequest,
  Category,
  UploadImageResponse,
  PromptActionResponse,
  Comment,
  CreateCommentRequest,
  CommentActionResponse,
  ReportActionResponse
} from '@/types'

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
    tag?: string
  }): Promise<ApiResponse<PageResponse<Prompt>>> {
    return request.get('/prompts', { params })
  },

  // Get prompt detail
  getPromptDetail(id: number): Promise<ApiResponse<Prompt>> {
    return request.get(`/prompts/${id}`)
  },

  getRelatedPrompts(id: number): Promise<ApiResponse<Prompt[]>> {
    return request.get(`/prompts/${id}/related`)
  },

  // Publish a new prompt
  publishPrompt(data: PublishPromptRequest): Promise<ApiResponse<Prompt>> {
    return request.post('/prompts', data)
  },

  savePromptDraft(data: PublishPromptRequest): Promise<ApiResponse<Prompt>> {
    return request.post('/prompts', { ...data, status: 0 })
  },

  getMyPromptDetail(id: number): Promise<ApiResponse<Prompt>> {
    return request.get(`/user/prompts/${id}`)
  },

  getMyDraftPrompts(): Promise<ApiResponse<Prompt[]>> {
    return request.get('/user/drafts')
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

  unlikePrompt(id: number): Promise<ApiResponse<PromptActionResponse>> {
    return request.delete(`/prompts/${id}/unlike`)
  },

  // Favorite a prompt
  favoritePrompt(id: number): Promise<ApiResponse<PromptActionResponse>> {
    return request.post(`/prompts/${id}/favorite`)
  },

  unfavoritePrompt(id: number): Promise<ApiResponse<PromptActionResponse>> {
    return request.delete(`/prompts/${id}/unfavorite`)
  },

  getInteractionStatus(id: number): Promise<ApiResponse<{ liked: boolean; favorited: boolean }>> {
    return request.get(`/prompts/${id}/interaction`)
  },

  recordPromptView(id: number): Promise<ApiResponse<PromptActionResponse>> {
    return request.post(`/prompts/${id}/view`)
  },

  getPromptComments(id: number, sort?: string, page = 1, pageSize = 20): Promise<ApiResponse<PageResponse<Comment>>> {
    return request.get(`/prompts/${id}/comments`, {
      params: { sort, page, pageSize }
    })
  },

  createPromptComment(id: number, data: CreateCommentRequest): Promise<ApiResponse<Comment>> {
    return request.post(`/prompts/${id}/comments`, data)
  },

  likeComment(id: number): Promise<ApiResponse<CommentActionResponse>> {
    return request.post(`/comments/${id}/like`)
  },

  reportComment(
    id: number,
    data: { reason: string; detail?: string }
  ): Promise<ApiResponse<ReportActionResponse>> {
    return request.post(`/comments/${id}/report`, data)
  },

  reportPrompt(
    id: number,
    data: { reason: string; detail?: string }
  ): Promise<ApiResponse<ReportActionResponse>> {
    return request.post(`/prompts/${id}/report`, data)
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
    tag?: string
  }): Promise<ApiResponse<PageResponse<Prompt>>> {
    return request.get('/prompts/search', { params })
  }
}
