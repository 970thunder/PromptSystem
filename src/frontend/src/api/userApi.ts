import request from '@/utils/request'
import type { ApiResponse, LoginRequest, LoginResponse, Prompt, RegisterRequest, User } from '@/types'

export const userApi = {
  login(data: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    return request.post('/user/login', data)
  },

  register(data: RegisterRequest): Promise<ApiResponse<LoginResponse>> {
    return request.post('/user/register', data)
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

  logout(): Promise<ApiResponse<null>> {
    return request.post('/user/logout')
  },

  sendCaptcha(email: string): Promise<ApiResponse<null>> {
    return request.post('/user/captcha', { email })
  }
}
