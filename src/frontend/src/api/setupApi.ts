// 文件作用：首次部署初始化向导与站点信息 API 层。对接后端 /setup、/site
// 与 /admin/settings 前缀：初始化状态、一次性引导、SMTP 连通性测试与
// 管理员站点设置读写。写请求经全局 CSRF 拦截器自动附带令牌。
import request from '@/utils/request'
import type { ApiResponse } from '@/types'
import type { ApiContractSetupStatus, ApiContractSiteInfo } from '@/types/api-contract.generated'

export type SetupStatus = ApiContractSetupStatus
export type SiteInfo = ApiContractSiteInfo

export interface SMTPConfig {
  host: string
  port: string
  user: string
  password: string
  from: string
}

export interface SetupInitializePayload {
  username: string
  email: string
  password: string
  siteName?: string
  siteDescription?: string
  smtp?: SMTPConfig
  registrationEnabled?: boolean
}

export interface AdminSiteSettings {
  siteName: string
  siteDescription: string
  smtpHost: string
  smtpPort: string
  smtpUser: string
  smtpPasswordSet: boolean
  smtpFrom: string
  registrationEnabled: boolean
  smtpFromEnv: boolean
}

export interface AdminSettingsPayload {
  siteName: string
  siteDescription: string
  smtpHost: string
  smtpPort: string
  smtpUser: string
  smtpPassword: string
  smtpFrom: string
  registrationEnabled?: boolean
}

export const setupApi = {
  getStatus(): Promise<ApiResponse<SetupStatus>> {
    return request.get('/setup/status')
  },

  initialize(data: SetupInitializePayload): Promise<ApiResponse<{ adminId: number }>> {
    return request.post('/setup/initialize', data)
  },

  testSMTP(data: { smtp: SMTPConfig; recipient: string }): Promise<ApiResponse<{ sent: boolean }>> {
    return request.post('/setup/test-smtp', data)
  },

  getSiteInfo(): Promise<ApiResponse<SiteInfo>> {
    return request.get('/site/info')
  },

  getAdminSettings(): Promise<ApiResponse<AdminSiteSettings>> {
    return request.get('/admin/settings')
  },

  updateAdminSettings(data: AdminSettingsPayload): Promise<ApiResponse<AdminSiteSettings>> {
    return request.put('/admin/settings', data)
  },

  adminTestSMTP(recipient: string): Promise<ApiResponse<{ sent: boolean }>> {
    return request.post('/admin/settings/test-smtp', { recipient })
  }
}
