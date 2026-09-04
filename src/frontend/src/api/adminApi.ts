// 文件作用：管理员审核 API 层。对接后端 /admin 前缀接口：举报列表、举报审核、
// 内容/用户处置与审计链查询。所有写请求经全局 CSRF 拦截器自动附带令牌。
import request from '@/utils/request'
import type {
  ApiResponse,
  AuditEvent,
  PageResponse,
  Report
} from '@/types'

export type ReportStatusFilter = 'pending' | 'reviewed' | 'rejected'

export interface ReviewReportPayload {
  status: 'reviewed' | 'rejected'
  action: 'none' | 'remove'
  note: string
}

export interface ModerationStatusPayload {
  status: -1 | 0 | 1
  reason: string
}

export const adminApi = {
  listReports(status: ReportStatusFilter | '', page = 1, pageSize = 20): Promise<ApiResponse<PageResponse<Report>>> {
    return request.get('/admin/reports', { params: { status: status || undefined, page, pageSize } })
  },

  reviewReport(id: number, data: ReviewReportPayload): Promise<ApiResponse<Report>> {
    return request.patch(`/admin/reports/${id}`, data)
  },

  setPromptStatus(id: number, data: ModerationStatusPayload): Promise<ApiResponse<{ updated: boolean }>> {
    return request.patch(`/admin/prompts/${id}`, data)
  },

  setUserStatus(id: number, data: ModerationStatusPayload): Promise<ApiResponse<{ updated: boolean }>> {
    return request.patch(`/admin/users/${id}`, data)
  },

  listAuditEvents(page = 1, pageSize = 20): Promise<ApiResponse<PageResponse<AuditEvent>>> {
    return request.get('/admin/audit', { params: { page, pageSize } })
  }
}
