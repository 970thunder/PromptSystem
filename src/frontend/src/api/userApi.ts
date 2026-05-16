import request from '@/utils/request'
import type { ApiResponse, LoginRequest, RegisterRequest, User } from '@/types'

export const userApi = {
  login(data: LoginRequest): Promise<ApiResponse<{ token: string; user: User }>> {
    return request.post('/user/login', data)
  },

  register(data: RegisterRequest): Promise<ApiResponse<{ token: string; user: User }>> {
    return request.post('/user/register', data)
  },

  getUserInfo(): Promise<ApiResponse<User>> {
    return request.get('/user/info')
  },

  updateUserInfo(data: Partial<User>): Promise<ApiResponse<User>> {
    return request.put('/user/info', data)
  },

  logout(): Promise<ApiResponse<null>> {
    return request.post('/user/logout')
  },

  sendCaptcha(email: string): Promise<ApiResponse<null>> {
    return request.post('/user/captcha', { email })
  }
}
