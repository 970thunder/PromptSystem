import request from '@/utils/request'
import type {
  ApiResponse,
  AuthExchangeResponse,
  FollowActionResponse,
  FollowStatus,
  PageResponse,
  Prompt,
  User,
  UserDataExport
} from '@/types'

export const userApi = {
  getUserInfo(): Promise<ApiResponse<User>> {
    return request.get('/user/info', { promptosSilent: true })
  },

  exportData(): Promise<ApiResponse<UserDataExport>> {
    return request.get('/user/data-export')
  },

  clearHistory(): Promise<ApiResponse<{ cleared: boolean }>> {
    return request.delete('/user/history')
  },

  deleteAccount(): Promise<ApiResponse<{ deleted: boolean }>> {
    return request.delete('/user/account')
  },

  updateUserInfo(data: Partial<User>): Promise<ApiResponse<User>> {
    return request.put('/user/info', data)
  },

  getFavoritePrompts(): Promise<ApiResponse<Prompt[]>> {
    return request.get('/user/favorites')
  },

  getLikedPrompts(): Promise<ApiResponse<Prompt[]>> {
    return request.get('/user/likes')
  },

  getHistoryPrompts(page = 1, pageSize = 24): Promise<ApiResponse<PageResponse<Prompt>>> {
    return request.get('/user/history', { params: { page, pageSize } })
  },

  getFollowingUsers(): Promise<ApiResponse<User[]>> {
    return request.get('/user/following')
  },

  getFollowerUsers(): Promise<ApiResponse<User[]>> {
    return request.get('/user/followers')
  },

  getFollowStatus(userId: number): Promise<ApiResponse<FollowStatus>> {
    return request.get(`/users/${userId}/follow-status`)
  },

  followUser(userId: number): Promise<ApiResponse<FollowActionResponse>> {
    return request.post(`/users/${userId}/follow`)
  },

  unfollowUser(userId: number): Promise<ApiResponse<FollowActionResponse>> {
    return request.delete(`/users/${userId}/follow`)
  },

  logout(): Promise<ApiResponse<null>> {
    return request.post('/user/logout')
  },

  exchangeGithubCode(code: string): Promise<ApiResponse<AuthExchangeResponse>> {
    return request.post('/auth/exchange', { code })
  }
}
