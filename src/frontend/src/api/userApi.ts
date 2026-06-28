import request from '@/utils/request'
import type {
  ApiResponse,
  CaptchaResponse,
  FollowActionResponse,
  FollowStatus,
  LoginRequest,
  LoginResponse,
  Prompt,
  RegisterRequest,
  ResetPasswordRequest,
  User
} from '@/types'

export const userApi = {
  login(data: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    return request.post('/user/login', data)
  },

  register(data: RegisterRequest): Promise<ApiResponse<LoginResponse>> {
    return request.post('/user/register', data)
  },

  resetPassword(data: ResetPasswordRequest): Promise<ApiResponse<null>> {
    return request.post('/user/password/reset', data)
  },

  getUserInfo(): Promise<ApiResponse<User>> {
    return request.get('/user/info')
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

  getHistoryPrompts(): Promise<ApiResponse<Prompt[]>> {
    return request.get('/user/history')
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

  sendCaptcha(email: string): Promise<ApiResponse<CaptchaResponse>> {
    return request.post('/user/captcha', { email })
  }
}
